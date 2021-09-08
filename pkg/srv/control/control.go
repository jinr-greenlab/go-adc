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

package control

import (
	"context"
	"fmt"
	"github.com/google/gopacket"
	"hash/crc32"
	"jinr.ru/greenlab/go-adc/pkg/srv/control/ifc"
	"net"
	"strings"
	"time"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
)

const (
	RegPort = 33300
)

type ControlServer struct {
	srv.Server
	seq uint16
	state *RegState
	api ifc.ApiServer
}

var _ ifc.ControlServer = &ControlServer{}

// NewControlServer ...
func NewControlServer(ctx context.Context, cfg *config.Config) (ifc.ControlServer, error) {
	log.Debug("Initializing control server with address: %s port: %d", cfg.IP, RegPort)

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.IP, RegPort))
	if err != nil {
		return nil, err
	}

	regState, err := NewRegState(ctx, cfg)
	if err != nil {
		return nil, err
	}

	s := &ControlServer{
		Server: srv.Server{
			Context: ctx,
			Config:  cfg,
			UDPAddr: uaddr,
			ChIn:    make(chan srv.InPacket),
			ChOut:   make(chan srv.OutPacket),
		},
		seq: 0,
		state: regState,
	}

	apiServer, err := NewApiServer(ctx, cfg, s)
	if err != nil {
		return nil, err
	}
	s.api = apiServer

	return s, nil
}

func (s *ControlServer) Run() error {
	conn, err := net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}

	defer conn.Close()
	defer s.state.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 65536)

	// Read UDP packets from wire and put them to input queue
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
			device, err := s.GetDeviceByIP(ipAddr)
			if err != nil {
				log.Debug("Drop packet. Device not found for given IP: %s ", ipAddr.String())
				continue
			}

			captureInfo := gopacket.CaptureInfo{
				Length: length,
				CaptureLength: length,
				Timestamp: time.Now(),
				AncillaryData: []interface{}{udpAddr, device.Name},
			}

			s.ChIn <- srv.InPacket{Data: buffer[:length], CaptureInfo: captureInfo}
		}
	}()

	// Read captured packets from input queue, parse them and update the state
	go func() {
		source := gopacket.NewPacketSource(s, layers.MLinkLayerType)
		for packet := range source.Packets() {
			log.Debug("Packet received")
			deviceName, packetErr := srv.GetDeviceName(packet)
			if packetErr != nil {
				log.Error(packetErr.Error())
				continue
			}
			reg := packet.Layer(layers.RegLayerType)
			if reg != nil {
				log.Debug("Packet parsed")
				reg := reg.(*layers.RegLayer)
				for _, op := range reg.RegOps {
					s.state.SetReg(op.Reg, deviceName)
				}
			}
		}
	}()

	// Read packets from output queue and send them to wire
	go func() {
		for {
			outPacket := <-s.ChOut
			_, sendErr := conn.WriteToUDP(outPacket.Data, outPacket.UDPAddr)
			if sendErr != nil {
				log.Error("Error while sending data to %s", outPacket.UDPAddr)
				errChan <- sendErr
				return
			}
		}
	}()

	go func() {
		s.api.Run()
	}()

	select {
	case <-s.Context.Done():
		return s.Context.Err()
	case err = <-errChan:
		return err
	}
}

func (s *ControlServer) NextSeq() uint16 {
	seq := s.seq; s.seq++; return seq
}


func (s *ControlServer) RegRequest(ops []*layers.RegOp, deviceName string) error {
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

	s.ChOut <- srv.OutPacket{
		Data: buf.Bytes(),
		UDPAddr: udpAddr,
	}
	return nil
}

func (s *ControlServer) MemRequest(op *layers.MemOp, deviceName string) error {
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

	s.ChOut <- srv.OutPacket{
		Data: buf.Bytes(),
		UDPAddr: udpAddr,
	}
	return nil
}


func (s *ControlServer) RegRead(addr uint16, device string) (*layers.Reg, error) {
	return s.state.GetReg(addr, device)
}

func (s *ControlServer) RegReadAll(device string) ([]*layers.Reg, error) {
	return s.state.GetRegAll(device)
}

func (s *ControlServer) RegWrite(reg *layers.Reg, device string) error {
	ops := []*layers.RegOp{
		{
			Read: false,
			Reg: reg,
		},
	}
	return s.RegRequest(ops, device)
}

// for details how to start and stop streaming data see DominoDevice::writeSettings()

// MStreamStart ...
func (s *ControlServer) MStreamStart(device string) error {
	var ops []*layers.RegOp
	ops = []*layers.RegOp{
		{
			Reg: &layers.Reg{
				Addr:  RegMap[RegCtrl],
				Value: 0,
			},
		},
		{
			Reg: &layers.Reg{
				Addr:  RegMap[RegCtrl],
				Value: 0x8000,
			},
		},
	}

	return s.RegRequest(ops, device)
}

// MStreamStartAll ...
func (s *ControlServer) MStreamStartAll() error {
	for _, d := range s.Devices {
		err := s.MStreamStart(d.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

// MStreamStop ...
func (s *ControlServer) MStreamStop(deviceName string) error {
	ops := []*layers.RegOp{
		{
			Reg: &layers.Reg{
				Addr:  RegMap[RegCtrl],
				Value: 1,
			},
		},
		{
			Reg: &layers.Reg{
				Addr: RegMap[RegCtrl],
				Value: 0,
			},
		},
	}
	return s.RegRequest(ops, deviceName)
}

// MStreamStopAll ...
func (s *ControlServer) MStreamStopAll() error {
	for _, d := range s.Devices {
		err := s.MStreamStop(d.Name)
		if err != nil {
			return err
		}
	}
	return nil
}
