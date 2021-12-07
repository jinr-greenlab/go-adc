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
	"jinr.ru/greenlab/go-adc/pkg/config"
	"net"
	"time"

	"github.com/google/gopacket"
	gopacketlayers "github.com/google/gopacket/layers"

	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
)

const (
	DiscoverPort = 33303
)

/*
 As defined by the IANA, the leftmost 24 bits of an IPv4 multicast
 MAC address are 0x01005E, the 25th bit is 0, and the rightmost 23
 bits are mapped to the rightmost 23 bits of a multicast IPv4 address.
 For example, if the IPv4 multicast address of a group is 224.0.1.1,
 the IPv4 multicast MAC address of this group is 01-00-5E-00-01-01.`
 */

type DiscoverServer struct {
	srv.Server
	*net.Interface
	state *State
	api *ApiServer
}

func NewDiscoverServer(ctx context.Context, cfg *config.Config) (*DiscoverServer, error) {
	log.Info("Initializing discover server with address: %s port: %d iface: %s",
		cfg.DiscoverIP, DiscoverPort, cfg.DiscoverIface)

	iface, err := net.InterfaceByName(cfg.DiscoverIface)
	if err != nil {
		return nil, err
	}

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.DiscoverIP, DiscoverPort))
	if err != nil {
		return nil, err
	}

	state, err := NewState(ctx, cfg)
	if err != nil {
		return nil, err
	}

	s := &DiscoverServer{
		Server: srv.Server{
			Context: context.Background(),
			UDPAddr: uaddr,
			ChIn: make(chan srv.InPacket),
			Config: cfg,
		},
		Interface: iface,
		state: state,
	}

	apiServer, err := NewApiServer(ctx, cfg, s)
	if err != nil {
		return nil, err
	}
	s.api = apiServer

	return s, nil
}

func (s *DiscoverServer) Run() error {

	conn, err := net.ListenMulticastUDP("udp", s.Interface, s.UDPAddr)
	if err != nil {
		return err
	}

	defer conn.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 2048)

	// Read UDP packets from wire and put them to input queue
	go func() {
		for {
			length, addr, captureErr := conn.ReadFrom(buffer)
			if captureErr != nil {
				errChan <- captureErr
				return
			}

			udpAddr, readErr := net.ResolveUDPAddr("udp", addr.String())
			if readErr != nil {
				errChan <- readErr
				return
			}

			captureInfo := gopacket.CaptureInfo{
				Length: length,
				CaptureLength: length,
				InterfaceIndex: s.Interface.Index,
				Timestamp: time.Now(),
				AncillaryData: []interface{}{udpAddr},
			}
			packet := srv.InPacket{CaptureInfo: captureInfo, Data: make([]byte, length)}
			copy(packet.Data, buffer[:length])
			s.ChIn <- packet
		}
	}()

	// Read captured packets from input queue, parse them and update the discover database
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
				udpAddr, handleErr := srv.GetAddrPort(packet)
				if handleErr != nil {
					// TODO
					continue
				}
				dd.SetSource(udpAddr)
				dd.SetTimestamp()

				if err := s.state.CreateBucket(BucketName(dd.SerialNumber)); err != nil {
					log.Error("Error while creating bucket: device: %s", dd.SerialNumber)
					continue
				}
				if err := s.state.SetDeviceDescription(dd); err != nil {
					log.Error("Error while updating device description: device: %s error: %s", dd.SerialNumber, err)
					continue
				}
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

