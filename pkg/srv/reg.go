/*
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package srv

import (
	"context"
	"fmt"
	"hash/crc32"
	"net"
	"time"

	"github.com/google/gopacket"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	RegPort = 33300
)

const (
	CtrlReg = 0x40
	CtrlDefault = 0x0000

	WinReg = 0x41
	StatusReg = 0x42 // read only
	WinDefault = 0xFFFF

	TrigMultReg = 0x43
	SerialIDHiReg = 0x46 // read only
	LiveMagic = 0x48

	ChDpmKsReg = 0x4A // read only

	TemperatureReg = 0x4B // read only
	FwVerReg = 0x4C // read only
	FwRevReg = 0x4D // read only

	SerialIDReg = 0x4E // read only
	MstreamCfg = 0x4F // read only

	TsReadA = 0x50
	TsSetReg64 = 0x54
	TsReadB = 0x58
	TsReadReg64 = 0x5C
)

type RegServer struct {
	Server
	Seq uint16
	RegState map[uint16]uint16
	chRegStateOp chan RegStateOp
}

type RegStateOp struct {
	Read bool
	RegNum uint16
	RegValue uint16
}

func NewRegServer(cfg *config.Config) (*RegServer, error) {
	log.Debug("Initializing reg server with address: %s port: %d", cfg.IP, RegPort)

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.IP, RegPort))
	if err != nil {
		return nil, err
	}

	s := &RegServer{
		Server: Server{
			Context:      context.Background(),
			Config: cfg,
			UDPAddr:      uaddr,
			chCaptured:   make(chan Captured),
			chSend:       make(chan Send),
		},
		Seq: 0,
		RegState: make(map[uint16]uint16),
		chRegStateOp: make(chan RegStateOp),
	}
	return s, nil
}

func (s *RegServer) Run() error {

	conn, err := net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}

	defer conn.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 65536)

	go func() {
		source := gopacket.NewPacketSource(s, layers.MLinkLayerType)
		for packet := range source.Packets() {
			log.Debug("Reg packet received")
			reg := packet.Layer(layers.RegLayerType)
			if reg != nil {
				log.Debug("Reg response successfully parsed")
				reg := reg.(*layers.RegLayer)
				s.SetRegState(reg.RegNum, reg.RegValue)
			}
		}
	}()

	// TODO deduplicate this
	go func() {
		for {
			length, addr, readErr := conn.ReadFrom(buffer)
			if readErr != nil {
				errChan <- readErr
				return
			}
			peerUDPAddr, readErr := net.ResolveUDPAddr("udp", addr.String())
			if readErr != nil {
				errChan <- readErr
				return
			}
			ci := gopacket.CaptureInfo{
				Length: length,
				CaptureLength: length,
				Timestamp: time.Now(),
				AncillaryData: []interface{}{peerUDPAddr},
			}

			s.chCaptured <- Captured{Data: buffer[:length], CaptureInfo: ci}
		}
	}()

	// TODO deduplicate this
	go func() {
		for {
			send := <-s.chSend
			_, sendErr := conn.WriteToUDP(send.Data, send.UDPAddr)
			if sendErr != nil {
				log.Error("Error while sending data to %s", send.UDPAddr)
				errChan <- sendErr
				return
			}
		}
	}()

	go func() {
		for {
			regStateOp := <-s.chRegStateOp
			log.Debug("Register operation: read: %t regnum: %x regvalue: %x",
				regStateOp.Read, regStateOp.RegNum, regStateOp.RegValue)
			if regStateOp.Read {
				regValue, ok := s.RegState[regStateOp.RegNum]
				if ok {
					log.Debug("Register: %x = %x\n", regStateOp.RegNum, regValue)
				}
			} else {
				s.RegState[regStateOp.RegNum] = regStateOp.RegValue
			}
		}
	}()

	// read packets for the packets channel and send them
	select {
	case <-s.Context.Done():
		return s.Context.Err()
	case err = <-errChan:
		return err
	}
}

func (s *RegServer) NextSeq() uint16 {
	seq := s.Seq; s.Seq++; return seq
}

func (s *RegServer) SetRegState(regNum, regValue uint16) {
	log.Debug("SetRegState: %x %x", regNum, regValue)
	regStateOp := RegStateOp{
		Read: false,
		RegNum: regNum,
		RegValue: regValue,
	}
	s.chRegStateOp <- regStateOp
}

func (s *RegServer) GetRegState(regNum uint16) {
	log.Debug("GetRegState: %x", regNum)
	regStateOp := RegStateOp{
		Read: true,
		RegNum: regNum,
	}
	s.chRegStateOp <- regStateOp
}

func (s *RegServer) SendRequest(read bool, regNum, regValue uint16, udpAddr *net.UDPAddr) error {
	ml := &layers.MLinkLayer{}
	ml.Type = layers.MLinkTypeRegRequest
	ml.Sync = layers.MLinkSync
	// 3 words for MLink header + 1 word CRC + 1 word for request
	ml.Len = uint16(5)
	ml.Seq = s.NextSeq()
	ml.Src = layers.MLinkHostAddr
	ml.Dst = layers.MLinkDeviceAddr

	// Calculate crc32 checksum
	mlHeaderBytes := make([]byte, 12)
	ml.SerializeHeader(mlHeaderBytes)

	req := &layers.RegLayer{}
	req.Read = read
	req.RegNum = regNum
	req.RegValue = regValue
	reqBytes := make([]byte, 4)
	req.Serialize(reqBytes)

	ml.Crc = crc32.ChecksumIEEE(append(mlHeaderBytes, reqBytes...))

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, ml, req)
	if err != nil {
		log.Error("Error while serializing layers when sending register r/w request to %s", udpAddr)
		return err
	}

	s.chSend <- Send{
		Data: buf.Bytes(),
		UDPAddr: udpAddr,
	}
	return nil
}

