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

type MStreamServer struct {
	context.Context
	*config.MStreamConfig
	*net.UDPAddr
	chCaptured chan Captured
	chSend chan Send
}

type Send struct {
	Data []byte
	*net.UDPAddr
}

func NewMStreamServer(cfg *config.MStreamConfig) (*MStreamServer, error) {
	log.Debug("Initializing mstream server with address: %s port: %d", cfg.Address, cfg.Port)

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.Address, cfg.Port))
	if err != nil {
		return nil, err
	}

	s := &MStreamServer{
		Context: context.Background(),
		MStreamConfig: cfg,
		UDPAddr: uaddr,
		chCaptured: make(chan Captured),
		chSend: make(chan Send),
	}

	return s, nil
}

func (s *MStreamServer) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	captured := <-s.chCaptured
	data = captured.Data
	ci = captured.CaptureInfo
	return
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
			ms := packet.Layer(layers.MStreamLayerType)
			if ms != nil {
				log.Debug("MStream frame successfully parsed")
				ms := ms.(*layers.MStreamLayer)
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
				peerUDPAddr, handleErr := GetAddrPort(packet)
				if handleErr != nil {
					continue
				}
				s.SendAck(ms.FragmentID, ms.FragmentOffset, peerUDPAddr)
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

	err = s.ConnectToPeers()
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

func (s *MStreamServer) SendAck(fragmentID, fragmentOffset uint16, peer *net.UDPAddr) error {
	ml := &layers.MLinkLayer{}
	ml.Type = layers.MLinkTypeMStream
	ml.Sync = layers.MLinkSync
	// 3 words for MLink header + 1 word CRC + 2 words for MStream header
	ml.Len = 6
	ml.Seq = 0
	ml.Src = 0xfefe
	ml.Dst = 1
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
		log.Error("Error while serializing layers when sending MStream ack message to peer %s", peer)
		return err
	}

	s.chSend <- Send{
		Data: buf.Bytes(),
		UDPAddr: peer,
	}
	return nil
}

func (s *MStreamServer) ConnectToPeers() error {
	// to connect to peer devices it is enough to send them an MStream ack
	// message with empty payload and with fragmentID = -1 and fragmentOffset = -1
	for _, peer := range s.MStreamConfig.Peers {
		peerUDPAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", peer.Address, peer.Port))
		if err != nil {
			return err
		}
		err = s.SendAck(0xffff, 0xffff, peerUDPAddr)
		if err != nil {
			log.Error("Error while connecting to MStream peer %s:%s", peer.Address, peer.Port)
			return err
		}
	}
	return nil
}
