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
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"

	pkgdevice "jinr.ru/greenlab/go-adc/pkg/device"
	deviceifc "jinr.ru/greenlab/go-adc/pkg/device/ifc"
	"jinr.ru/greenlab/go-adc/pkg/srv/control/ifc"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
)

const (
	RegPort         = 33300
	RegReadInterval = 30
)

type ControlServer struct {
	srv.Server
	seq     uint16
	state   ifc.State
	api     ifc.ApiServer
	devices map[string]*pkgdevice.Device
}

var _ ifc.ControlServer = &ControlServer{}

// NewControlServer ...
func NewControlServer(ctx context.Context, cfg *config.Config) (ifc.ControlServer, error) {
	log.Info("Initializing control server with address: %s port: %d", cfg.IP, RegPort)

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.IP, RegPort))
	if err != nil {
		return nil, err
	}

	state, err := NewState(ctx, cfg)
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
		seq:   0,
		state: state,
	}

	devices := make(map[string]*pkgdevice.Device)
	for _, cfgDevice := range cfg.Devices {
		device, newdevErr := pkgdevice.NewDevice(cfgDevice, s, state)
		if newdevErr != nil {
			return nil, err
		}
		devices[cfgDevice.Name] = device
	}
	s.devices = devices

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
			log.Debug("Received packet from %s", udpAddr)
			ipAddr := net.ParseIP(strings.Split(addr.String(), ":")[0])
			device, getdevErr := s.GetDeviceByIP(ipAddr)
			if getdevErr != nil {
				log.Debug("Drop packet. Device not found for given IP: %s ", ipAddr.String())
				continue
			}

			captureInfo := gopacket.CaptureInfo{
				Length:        length,
				CaptureLength: length,
				Timestamp:     time.Now(),
				AncillaryData: []interface{}{udpAddr, device.Name},
			}
			packet := srv.InPacket{CaptureInfo: captureInfo, Data: make([]byte, length)}
			copy(packet.Data, buffer[:length])
			s.ChIn <- packet
		}
	}()

	// Read captured packets from input queue, parse them and update the device state
	go func() {
		source := gopacket.NewPacketSource(s, layers.MLinkLayerType)
		for packet := range source.Packets() {
			deviceName, packetErr := srv.GetDeviceName(packet)
			if packetErr != nil {
				log.Error(packetErr.Error())
				continue
			}
			log.Debug("Received packet from device: %s packet: %s", deviceName, hex.EncodeToString(packet.Data()))
			log.Debug(packet.Dump())
			device, ok := s.devices[deviceName]
			if !ok {
				log.Error("Packet unknown device: %s", deviceName)
				continue
			}
			layer := packet.Layer(layers.RegLayerType)
			if layer != nil {
				layer, ok := layer.(*layers.RegLayer)
				if !ok {
					log.Error("Error while asserting to RegLayer")
					continue
				}
				for _, op := range layer.RegOps {
					upregErr := device.UpdateReg(op.Reg)
					if upregErr != nil {
						log.Error("Fail to update device: device = %s", deviceName)
						continue
					}
				}
			}
		}
	}()

	// Read packets from output queue and send them to wire
	go func() {
		for {
			outPacket := <-s.ChOut
			log.Debug("Sending packet to %s", outPacket.UDPAddr)
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

	// Periodically read all registers from all devices
	go func() {
		for {
			var ops []*layers.RegOp
			for i := pkgdevice.RegAlias(0); i < pkgdevice.RegAliasLimit; i++ {
				addr := pkgdevice.RegMap[i]
				ops = append(ops, &layers.RegOp{Read: true, Reg: &layers.Reg{Addr: addr}})
			}
			for _, device := range s.devices {
				regreqErr := s.RegRequest(ops, device.IP)
				if regreqErr != nil {
					log.Error("Error while sending reg request to device %s", device.IP)
				}
			}
			time.Sleep(RegReadInterval * time.Second)
		}
	}()

	select {
	case <-s.Context.Done():
		return s.Context.Err()
	case err = <-errChan:
		return err
	}
}

// NextSeq ...
func (s *ControlServer) NextSeq() uint16 {
	seq := s.seq
	s.seq++
	return seq
}

// RegRequest ...
func (s *ControlServer) RegRequest(ops []*layers.RegOp, ip *net.IP) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, RegPort))
	if err != nil {
		return err
	}
	bytes, err := layers.RegOpsToBytes(ops, s.NextSeq())
	if err != nil {
		log.Error("Error while serializing layers when sending register r/w request to %s", udpAddr)
		return err
	}
	log.Debug("Put Reg request to queue: udpaddr: %s request: %s", udpAddr, hex.EncodeToString(bytes))
	s.ChOut <- srv.OutPacket{
		Data:    bytes,
		UDPAddr: udpAddr,
	}
	return nil
}

// MemRequest ...
func (s *ControlServer) MemRequest(op *layers.MemOp, ip *net.IP) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, RegPort))
	if err != nil {
		return err
	}
	bytes, err := layers.MemOpToBytes(op, s.NextSeq())
	if err != nil {
		log.Error("Error while serializing layers when sending memory r/w request to %s", udpAddr)
		return err
	}
	s.ChOut <- srv.OutPacket{
		Data:    bytes,
		UDPAddr: udpAddr,
	}
	return nil
}

// GetDeviceByName ...
func (s *ControlServer) GetDeviceByName(deviceName string) (deviceifc.Device, error) {
	device, ok := s.devices[deviceName]
	if !ok {
		return nil, srv.ErrDeviceNotFound{What: deviceName}
	}
	return device, nil
}

// GetAllDevices ...
func (s *ControlServer) GetAllDevices() map[string]deviceifc.Device {
	result := make(map[string]deviceifc.Device)
	for n, d := range s.devices {
		result[n] = d
	}
	return result
}

// RegRequestByDeviceName ...
func (s *ControlServer) RegRequestByDeviceName(ops []*layers.RegOp, deviceName string) error {
	device, err := s.Config.GetDeviceByName(deviceName)
	if err != nil {
		return err
	}
	return s.RegRequest(ops, device.IP)
}

// RegRequestByDeviceName ...
func (s *ControlServer) MemRequestByDeviceName(op *layers.MemOp, deviceName string) error {
	device, err := s.Config.GetDeviceByName(deviceName)
	if err != nil {
		return err
	}
	return s.MemRequest(op, device.IP)
}
