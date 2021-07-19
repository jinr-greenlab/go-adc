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
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"hash/crc32"
	"net"
	"time"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	RegPort = 33300
	ApiPort = 8000
	BucketName = "reg"
)

//const (
//	BOARD_REG_COUNT
//	MEM_BIT_SELECT_CTRL
//	MEM_CH_CTRL
//	MEM_CH_BLC_THR_HI
//	MEM_CH_BLC_THR_LO
//)

func uint16ToByte(v uint16) []byte {
    b := make([]byte, 2)
    binary.BigEndian.PutUint16(b, v)
    return b
}

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

type RegDBOp struct {
	ChResponse chan Reg
	ChError chan error
	Update bool
	Reg
}

type RegServer struct {
	Server
	*mux.Router
	Seq uint16
	DB *bbolt.DB
}

func NewRegServer(cfg *config.Config) (*RegServer, error) {
	log.Debug("Initializing reg server with address: %s port: %d", cfg.IP, RegPort)

	ctx := context.Background()

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.IP, RegPort))
	if err != nil {
		return nil, err
	}

	// open register database
	db, err := bbolt.Open(cfg.DBPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	// create bucket in the register database
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucket([]byte(BucketName))
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	s := &RegServer{
		Server: Server{
			Context: ctx,
			Config: cfg,
			UDPAddr:      uaddr,
			chCaptured:   make(chan Captured),
			chSend:       make(chan Send),
		},
		Seq: 0,
		DB: db,

	}
	return s, nil
}

func (s *RegServer) Run() error {

	conn, err := net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}

	defer conn.Close()
	defer s.DB.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 65536)

	// Read messages from network and update register database
	go func() {
		source := gopacket.NewPacketSource(s, layers.MLinkLayerType)
		for packet := range source.Packets() {
			log.Debug("Reg packet received")
			reg := packet.Layer(layers.RegLayerType)
			if reg != nil {
				log.Debug("Reg response successfully parsed")
				reg := reg.(*layers.RegLayer)
				for _, op := range reg.RegOps {
					s.SetRegState(op.RegNum, op.RegValue)
				}
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
		s.StartApiServer()
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

func (s *RegServer) SetRegState(regNum, regValue uint16) error {
	log.Debug("SetRegState: RegNum: %x RegValue: %x", regNum, regValue)
	if err := s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", BucketName))
		}
		if err := b.Put(uint16ToByte(regNum), uint16ToByte(regValue)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *RegServer) GetRegState(regNum uint16) (uint16, error) {
	log.Debug("GetRegState: RegNum: %x", regNum)
	var regValue uint16
	if err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", BucketName))
		}
		value := b.Get(uint16ToByte(regNum))
		if value == nil {
			return errors.New(fmt.Sprintf("Key not found: %d", regNum))
		}
		regValue = binary.BigEndian.Uint16(value)
		return nil
	}); err != nil {
		return 0, err
	}
	return regValue, nil
}

func (s *RegServer) RegRequest(ops []*layers.RegOp, udpAddr *net.UDPAddr) error {
	ml := &layers.MLinkLayer{}
	ml.Type = layers.MLinkTypeRegRequest
	ml.Sync = layers.MLinkSync
	// 3 words for MLink header + 1 word CRC + N words for request
	ml.Len = uint16(4 + len(ops))
	ml.Seq = s.NextSeq()
	ml.Src = layers.MLinkHostAddr
	ml.Dst = layers.MLinkDeviceAddr

	// Calculate crc32 checksum
	mlHeaderBytes := make([]byte, 12)
	ml.SerializeHeader(mlHeaderBytes)

	reg := &layers.RegLayer{}
	reg.RegOps = ops
	regBytes := make([]byte, len(ops) * 4)
	reg.Serialize(regBytes)

	ml.Crc = crc32.ChecksumIEEE(append(mlHeaderBytes, regBytes...))

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, ml, reg)
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

func (s *RegServer) MemRequest(op *layers.MemOp, udpAddr *net.UDPAddr) error {
	ml := &layers.MLinkLayer{}
	ml.Type = layers.MLinkTypeMemRequest
	ml.Sync = layers.MLinkSync
	// 3 words for MLink header + 1 word CRC + 1 word MemOp header + N words MemOp data
	ml.Len = uint16(4 + op.Size + 1)
	ml.Seq = s.NextSeq()
	ml.Src = layers.MLinkHostAddr
	ml.Dst = layers.MLinkDeviceAddr

	// Calculate crc32 checksum
	mlHeaderBytes := make([]byte, 12)
	ml.SerializeHeader(mlHeaderBytes)

	mem := &layers.MemLayer{}
	mem.MemOp = op
	memBytes := make([]byte, (1 + op.Size) * 4)
	mem.Serialize(memBytes)

	ml.Crc = crc32.ChecksumIEEE(append(mlHeaderBytes, memBytes...))

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, ml, mem)
	if err != nil {
		log.Error("Error while serializing layers when sending memory r/w request to %s", udpAddr)
		return err
	}

	s.chSend <- Send{
		Data: buf.Bytes(),
		UDPAddr: udpAddr,
	}
	return nil
}



func (s *RegServer) RegRequestToAllDevices(ops []*layers.RegOp) error {
	for _, device := range s.Config.Devices {
		udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, RegPort))
		if err != nil {
			return err
		}
		err = s.RegRequest(ops, udpAddr)
		if err != nil {
			return err
		}
	}
	return nil
}

// see DominoDevice::writeSettings()

func (s *RegServer) StopMStream() error {
	ops := []*layers.RegOp{
		{
			RegNum: CtrlReg,
			RegValue: 1,
		},
		{
			RegNum: CtrlReg,
			RegValue: 0,
		},
	}
	return s.RegRequestToAllDevices(ops)
}

func (s *RegServer) StartMStream() error {
	ops := []*layers.RegOp{
		{
			RegNum: CtrlReg,
			RegValue: 0,
		},
		{
			RegNum: CtrlReg,
			RegValue: 0x8000,
		},
	}
	return s.RegRequestToAllDevices(ops)
}


//func chBaseMemAddr(ch uint32) uint32 {
//	return ch << 14;
//}
//
//func (s *RegServer) MemWrite(offset uint32, data []uint32) {
//}
//
//
//func (s *RegServer) WriteChReg(ch uint32, addr uint16, data uint32) {
//	writeAddr := MEM_BIT_SELECT_CTRL
//	writeAddr |= addr
//	writeAddr |= chBaseMemAddr(ch)
//	s.MemWrite(writeAddr, data)
//}
