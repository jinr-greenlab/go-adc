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
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	MStreamPort = 33301
)

type MStreamServer struct {
	Server
}

func NewMStreamServer(cfg *config.Config) (*MStreamServer, error) {
	log.Debug("Initializing mstream server with address: %s port: %d", cfg.IP, MStreamPort)

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.IP, MStreamPort))
	if err != nil {
		return nil, err
	}

	s := &MStreamServer{
		Server: Server{
			Context:    context.Background(),
			Config:     cfg,
			UDPAddr:    uaddr,
			ChIn:       make(chan InPacket),
			ChOut:      make(chan OutPacket),
		},
	}

	return s, nil
}

func (s *MStreamServer) Run() error {

	conn, err := net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 65536)

	fileSuffix := time.Now().UTC().Format("20060102_150405")

	eventHandler := NewEventHandler(fileSuffix)
	defer eventHandler.Close()

	// Read packets from wire and put them to input queue
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

			s.ChIn <- InPacket{Data: buffer[:length], CaptureInfo: captureInfo}
		}
	}()

	// Read packets from input queue and handle them properly
	go func() {
		source := gopacket.NewPacketSource(s, layers.MLinkLayerType)
		defragmenter := layers.NewMStreamDefragmenter()
		for packet := range source.Packets() {
			log.Debug("MStream frame received")
			log.Debug(packet.Dump())
			layer := packet.Layer(layers.MStreamLayerType)
			if layer != nil {
				log.Debug("MStream frame successfully parsed")
				layer := layer.(*layers.MStreamLayer)

				deviceName, err := GetDeviceName(packet);
				if err != nil {
					log.Error("Error while trying to get device name from packet")
					continue
				}

				udpaddr, err := GetAddrPort(packet)
				if err != nil {
					log.Error("Error while getting udpaddr for a packet from input queue")
					continue
				}

				for _, f := range layer.Fragments {
					log.Debug("Handling fragment: FragmentID: 0x%04x FragmentOffset: 0x%04x LastFragment: %t",
						f.FragmentID, f.FragmentOffset, f.LastFragment())

					err := s.SendAck(f.FragmentID, f.FragmentOffset, udpaddr)
					if err != nil {
						log.Error("Error while sending Ack for fragment: ID: %d Offset: %d Length: %d",
							f.FragmentID, f.FragmentOffset, f.FragmentLength)
					}

					if f.Subtype == layers.MStreamTriggerSubtype && !f.LastFragment() {
						log.Error("!!! Something really bad is happening. Trigger data is fragmented.")
						continue
					}

					assembled, err := defragmenter.Defrag(f, deviceName)
					if err != nil {
						log.Error("Error while trying to handle MStream fragment")
						continue
					} else if assembled == nil {
						log.Debug("This was MStream fragment, we don't have full frame yet, do nothing")
						continue
					}

					log.Debug("Assembled fragment: ID: %d Offset: %d Lenght: %d",
						assembled.FragmentID, assembled.FragmentOffset, assembled.FragmentLength)

					if err = assembled.DecodePayload(); err != nil {
						log.Error("Error while decoding MStream fragment payload")
					}

					eventHandler.SetFragment(assembled)
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

	err = s.ConnectToDevices()
	if err != nil {
		return err
	}

	// read packets for the packets channel and send them
	select {
	case <-s.Context.Done():
		return s.Context.Err()
	case err = <-errChan:
		return err
	}
}

func (s *MStreamServer) SendAck(fragmentID, fragmentOffset uint16, udpAddr *net.UDPAddr) error {
	ml := &layers.MLinkLayer{}
	ml.Type = layers.MLinkTypeMStream
	ml.Sync = layers.MLinkSync
	// 3 words for MLink header + 1 word CRC + 2 words for MStream header
	ml.Len = 6
	ml.Seq = 0
	// Since this is ACK message SRC and DST are reversed.
	ml.Src = layers.MLinkDeviceAddr
	ml.Dst = layers.MLinkHostAddr
	ml.Crc = 0

	ms := &layers.MStreamLayer{
		Fragments: []*layers.MStreamFragment{
			{
				DeviceID:       1,
				Subtype:        0,
				Flags:          0b00010000,
				FragmentLength: 0,
				FragmentID:     fragmentID,
				FragmentOffset: fragmentOffset,
				Data:           []byte{},
			},
		},
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, ml, ms)
	if err != nil {
		log.Error("Error while serializing layers when sending MStream ack message to device %s", udpAddr)
		return err
	}

	log.Debug("Put MStream Ack to output queue: udpaddr: %s ack: %s", udpAddr, hex.EncodeToString(buf.Bytes()))

	s.ChOut <- OutPacket{
		Data: buf.Bytes(),
		UDPAddr: udpAddr,
	}
	return nil
}

func (s *MStreamServer) ConnectToDevices() error {
	// to connect to peer devices it is enough to send them an MStream ack
	// message with empty payload and with fragmentID = -1 and fragmentOffset = -1
	for _, device := range s.Config.Devices {
		udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, MStreamPort))
		if err != nil {
			return err
		}
		err = s.SendAck(0xffff, 0xffff, udpAddr)
		if err != nil {
			log.Error("Error while connecting to MStream device %s:%s", device.IP, MStreamPort)
			return err
		}
	}
	return nil
}
