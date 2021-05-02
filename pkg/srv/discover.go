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
	"jinr.ru/greenlab/go-adc/pkg/config"
	"net"
	"time"

	"github.com/google/gopacket"
	gopacketlayers "github.com/google/gopacket/layers"

	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

/*
 As defined by the IANA, the leftmost 24 bits of an IPv4 multicast
 MAC address are 0x01005E, the 25th bit is 0, and the rightmost 23
 bits are mapped to the rightmost 23 bits of a multicast IPv4 address.
 For example, if the IPv4 multicast address of a group is 224.0.1.1,
 the IPv4 multicast MAC address of this group is 01-00-5E-00-01-01.`
 */

type DiscoverServer struct {
	context.Context
	*config.DiscoverConfig
	*net.Interface
	*net.UDPAddr
	chCaptured chan Captured
}

func NewDiscoverServer(cfg *config.DiscoverConfig) (*DiscoverServer, error) {
	log.Debug("Initializing discover server with address: %s port: %d iface: %s",
		cfg.Address, cfg.Port, cfg.Interface)

	iface, err := net.InterfaceByName(cfg.Interface)
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.Address, cfg.Port))
	if err != nil {
		return nil, err
	}

	s := &DiscoverServer{
		Context: context.Background(),
		DiscoverConfig: cfg,
		Interface: iface,
		UDPAddr: uaddr,
		chCaptured: make(chan Captured),
	}
	return s, nil
}

// ReadPacketData reads chCaptured channel and returns packet data and metadata.
// This method is from PacketDataSource interface.
func (s *DiscoverServer) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	captured := <-s.chCaptured
	data = captured.Data
	ci = captured.CaptureInfo
	return
}

func (s *DiscoverServer) Run() error {

	conn, err := net.ListenMulticastUDP("udp", s.Interface, s.UDPAddr)
	if err != nil {
		return err
	}

	defer conn.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 2048)

	// read packets from the chCaptured channel using the ReadPacketData method and parse them
	// into DeviceDescription struct
	go func() {
		source := gopacket.NewPacketSource(s, gopacketlayers.LayerTypeLinkLayerDiscovery)
		for packet := range source.Packets() {
			layer := packet.Layer(gopacketlayers.LayerTypeLinkLayerDiscoveryInfo)
			if layer != nil {
				layer, ok := layer.(*gopacketlayers.LinkLayerDiscoveryInfo)
				if !ok {
					log.Error("Error while asserting to LinkLayerDiscoveryInfo")
					continue
				}
				dd := &layers.DeviceDescription{}
				layers.DecodeOrgSpecific(layer.OrgTLVs, dd)
				peerUDPAddr, handleErr := GetAddrPort(packet)
				if handleErr != nil {
					// TODO
					continue
				}
				dd.SetPeer(peerUDPAddr)
				fmt.Print(dd.String())
			}
		}
	}()

	// capture discovery packets from the wire and put them into the chCaptured channel
	go func() {
		for {
			length, addr, captureErr := conn.ReadFrom(buffer)
			if captureErr != nil {
				errChan <- captureErr
				return
			}

			ci := gopacket.CaptureInfo{
				Length: length,
				CaptureLength: length,
				InterfaceIndex: s.Interface.Index,
				Timestamp: time.Now(),
				AncillaryData: []interface{}{addr},
			}

			s.chCaptured <- Captured{Data: buffer[:length], CaptureInfo: ci}
		}
	}()

	select {
	case <-s.Context.Done():
		return s.Context.Err()
	case err = <-errChan:
		return err
	}

}

