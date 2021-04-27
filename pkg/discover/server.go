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

package discover

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"jinr.ru/greenlab/go-adc/pkg/common"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

/*
 As defined by the IANA, the leftmost 24 bits of an IPv4 multicast
 MAC address are 0x01005E, the 25th bit is 0, and the rightmost 23
 bits are mapped to the rightmost 23 bits of a multicast IPv4 address.
 For example, if the IPv4 multicast address of a group is 224.0.1.1,
 the IPv4 multicast MAC address of this group is 01-00-5E-00-01-01.`
 */


type Server struct {
	context.Context
	Address string
	Port string
	IfaceName string
	*net.Interface
	*net.UDPAddr
	chCaptured chan common.Captured
}

func NewServer(address, port, ifaceName string) (*Server, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", address, port))
	if err != nil {
		return nil, err
	}

	s := &Server{
		Context: context.Background(),
		Address: address,
		Port: port,
		IfaceName: ifaceName,
		Interface: iface,
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

	conn, err := net.ListenMulticastUDP("udp", s.Interface, s.UDPAddr)
	if err != nil {
		return err
	}

	defer conn.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 2048)

	go func() {
		source := gopacket.NewPacketSource(s, layers.LayerTypeLinkLayerDiscovery)
		for packet := range source.Packets() {
			dd := &DeviceDescription{}
			layer, ok := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo).(*layers.LinkLayerDiscoveryInfo)
			if !ok {
				log.Info("Wrong discovery packet received. Can not parse.")
				continue
			}
			decodeOrgSpecific(layer.OrgTLVs, dd)
			fmt.Print(dd.String())

		}
	}()

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
				InterfaceIndex: s.Interface.Index,
				Timestamp: time.Now(),
				AncillaryData: []interface{}{addr},
			}

			s.chCaptured <- common.Captured{Data: buffer[:length], CaptureInfo: ci}
		}
	}()

	select {
	case <-s.Context.Done():
		return s.Context.Err()
	case err = <-errChan:
		return err
	}

}

