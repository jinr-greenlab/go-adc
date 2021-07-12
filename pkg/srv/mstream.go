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
	"net"
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
			chCaptured: make(chan Captured),
			chSend:     make(chan Send),
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


	// read packets from the chCaptured packet channel and handle them
	go func() {
		source := gopacket.NewPacketSource(s, layers.MLinkLayerType)
		defragmenter := layers.NewMStreamDefragmenter()
		for packet := range source.Packets() {
			log.Debug("MStream frame received")
			ms := packet.Layer(layers.MStreamLayerType)
			if ms != nil {
				log.Debug("MStream frame successfully parsed")
				ms := ms.(*layers.MStreamLayer)
				// empty flow is ok until we work with the only device
				// once we have many devices we have to use non empty flow
				flow := &gopacket.Flow{}
				out, handleErr := defragmenter.Defrag(ms, flow)
				if handleErr != nil {
					log.Error("Error while trying to defragment MStream frame")
					continue
				} else if out == nil {
					// this was MStream fragment, we don't have full frame yet, do nothing
					continue
				}
				log.Debug("Successfully decoded and defragmented MStream frame")
				udpaddr, handleErr := GetAddrPort(packet)
				if handleErr != nil {
					continue
				}
				s.SendAck(ms.FragmentID, ms.FragmentOffset, udpaddr)
			}
		}
	}()

	// receive data from the wire and put them to the chCaptured packet channel
	go func() {
		for {
			// TODO Use ReadFromUDP
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
			ci := gopacket.CaptureInfo{
				Length: length,
				CaptureLength: length,
				Timestamp: time.Now(),
				AncillaryData: []interface{}{udpAddr},
			}

			s.chCaptured <- Captured{Data: buffer[:length], CaptureInfo: ci}
		}
	}()

	// read data from the chSend channel and send them to the wire
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

	ms := &layers.MStreamLayer{}
	ms.DeviceID = 1
	ms.Subtype = 0
	ms.Flags = 0b00010000
	ms.FragmentLength = 0
	ms.FragmentID = fragmentID
	ms.FragmentOffset = fragmentOffset

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, ml, ms)
	if err != nil {
		log.Error("Error while serializing layers when sending MStream ack message to device %s", udpAddr)
		return err
	}

	s.chSend <- Send{
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
