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

package mstream

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"

	"jinr.ru/greenlab/go-adc/pkg/common"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

type Server struct {
	context.Context
	Address string
	Port string
	*net.UDPAddr
	chCaptured chan common.Captured
	chToSend chan Send
}

type Send struct {
	Data []byte
}

func NewServer(address, port string) (*Server, error) {
	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", address, port))
	if err != nil {
		return nil, err
	}

	s := &Server{
		Context: context.Background(),
		Address: address,
		Port: port,
		UDPAddr: uaddr,
		chCaptured: make(chan common.Captured),
	}

	return s, nil
}

func (s *Server) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	captured := <-s.chCaptured
	data = captured.Data
	ci = captured.CaptureInfo
	return
}

func (s *Server) Run() error {

	conn, err := net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}

	defer conn.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 65536)

	// read packets from the handler channel and handle them
	go func() {
		source := gopacket.NewPacketSource(s, MLinkLayerType)
		defragmenter := NewMStreamDefragmenter()
		for packet := range source.Packets() {
			ml, ok := packet.Layer(MLinkLayerType).(*MLinkLayer)
			if !ok {
				log.Error("Error while parsing MLink layer")
			}
			ms, ok := packet.Layer(MStreamLayerType).(*MStreamLayer)
			if !ok {
				log.Error("Error while parsing MStream layer")
			}
			out, err := defragmenter.Defrag(ms, ml)
			if err != nil {
				log.Error("Error while trying to defragment MStream frame")
			} else if out == nil {
				// this was MStream fragment, we don't have full frame yet, do nothing
				continue
			}
		}
	}()

	// receive data from the wire and put them to the handler channel
	go func() {
		for {
			length, addr, err := conn.ReadFrom(buffer)
			if err != nil {
				errChan <- err
				return
			}

			ci := gopacket.CaptureInfo{
				Length: length,
				CaptureLength: length,
				Timestamp: time.Now(),
				AncillaryData: []interface{}{addr},
			}

			s.chCaptured <- common.Captured{Data: buffer[:length], CaptureInfo: ci}
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



//func ConnectToHardware() error {
//	info := &MLinkInfo{}
//	info.FragmentID = -1
//	info.FragmentOffset = -1
//	info.Seq = 0
//	info.Src = 1
//	info.Dst = 0xfefe
//	return SendAck(info)
//}
