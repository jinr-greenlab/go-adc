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
	"strings"
	"time"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	RegPort = 33300
	ApiPort = 8000
	BucketNamePrefix = "reg_"
	MStreamActionStart = "start"
	MStreamActionStop = "stop"
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
	// create buckets in the register database for all devices
	if err := db.Update(func(tx *bbolt.Tx) error {
		for _, device := range cfg.Devices {
			_, err = tx.CreateBucketIfNotExists([]byte(bucketName(device.Name)))
			if err != nil {
				return err
			}
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
			deviceName, packetErr := GetDeviceName(packet)
			if packetErr != nil {
				log.Error(packetErr.Error())
				continue
			}
			reg := packet.Layer(layers.RegLayerType)
			if reg != nil {
				log.Debug("Reg packet successfully parsed")
				reg := reg.(*layers.RegLayer)
				for _, op := range reg.RegOps {
					s.SetRegState(op.RegNum, op.RegValue, deviceName)
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
			udpAddr, readErr := net.ResolveUDPAddr("udp", addr.String())
			if readErr != nil {
				errChan <- readErr
				return
			}
			ipAddr := net.ParseIP(strings.Split(addr.String(), ":")[0])
			device, err := s.GetDeviceByIP(&ipAddr)
			if err != nil {
				log.Debug("Device not found: %s", ipAddr.String())
				continue
			}

			ci := gopacket.CaptureInfo{
				Length: length,
				CaptureLength: length,
				Timestamp: time.Now(),
				AncillaryData: []interface{}{udpAddr, device.Name},
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

// SetRegState
func (s *RegServer) SetRegState(regNum, regValue uint16, deviceName string) error {
	log.Debug("SetRegState: RegNum: %x RegValue: %x", regNum, regValue)
	if err := s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName(deviceName)))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", bucketName(deviceName)))
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

// GetRegState
func (s *RegServer) GetRegState(regNum uint16, deviceName string) (uint16, error) {
	log.Debug("GetRegState: RegNum: %x", regNum)
	var regValue uint16
	if err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName(deviceName)))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", bucketName(deviceName)))
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

func (s *RegServer) RegRequest(ops []*layers.RegOp, deviceName string) error {
	device, err := s.Config.GetDeviceByName(deviceName)
	if err != nil {
		return err
	}

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, RegPort))
	if err != nil {
		return err
	}

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
	err = gopacket.SerializeLayers(buf, opts, ml, reg)
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

func (s *RegServer) MemRequest(op *layers.MemOp, deviceName string) error {
	device, err := s.Config.GetDeviceByName(deviceName)
	if err != nil {
		return err
	}

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, RegPort))
	if err != nil {
		return err
	}

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
	err = gopacket.SerializeLayers(buf, opts, ml, mem)
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


// see DominoDevice::writeSettings()


func (s *RegServer) MStreamAction(action, deviceName string) error {
	var ops []*layers.RegOp
	switch action {
	case MStreamActionStart:
		ops = []*layers.RegOp{
			{
				RegNum:   CtrlReg,
				RegValue: 0,
			},
			{
				RegNum:   CtrlReg,
				RegValue: 0x8000,
			},
		}
	case MStreamActionStop:
		ops = []*layers.RegOp{
			{
				RegNum: CtrlReg,
				RegValue: 1,
			},
			{
				RegNum: CtrlReg,
				RegValue: 0,
			},
		}
	default:
		return errors.New(fmt.Sprintf("Unknown MStream action: %s", action))
	}

	err := s.RegRequest(ops, deviceName)
	if err != nil {
		return err
	}

	return nil
}

func bucketName(deviceName string) string {
	return fmt.Sprintf("%s%s", BucketNamePrefix, deviceName)
}