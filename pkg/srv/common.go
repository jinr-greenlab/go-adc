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
	"net"
	"time"

	"github.com/google/gopacket"

	"jinr.ru/greenlab/go-adc/pkg/config"
)

type InPacket struct {
	Data []byte
	gopacket.CaptureInfo
}

type OutPacket struct {
	Data []byte
	*net.UDPAddr
}

// GetAddrPort returns the UDPAddr of the device that sent the packet
func GetAddrPort(packet gopacket.Packet) (*net.UDPAddr, error) {
	meta := packet.Metadata()
	if len(meta.CaptureInfo.AncillaryData) >= 1 {
		ancillary := meta.CaptureInfo.AncillaryData[0]
		udpAddr, ok := ancillary.(*net.UDPAddr)
		if !ok {
			return nil, ErrGetAddr{}
		}
		return udpAddr, nil
	}
	return nil, ErrGetAddr{}
}

// GetDeviceName returns the UDPAddr of the device that sent the packet
func GetDeviceName(packet gopacket.Packet) (string, error) {
	meta := packet.Metadata()
	if len(meta.CaptureInfo.AncillaryData) >= 2 {
		ancillary := meta.CaptureInfo.AncillaryData[1]
		deviceName, ok := ancillary.(string)
		if !ok {
			return "", ErrGetDeviceName{What: "can not cast ancillary data to string"}
		}
		return deviceName, nil
	}
	return "", ErrGetDeviceName{What: "not enough ancillary data"}
}

type Server struct {
	context.Context
	*config.Config
	*net.UDPAddr
	ChIn  chan InPacket
	ChOut chan OutPacket
}

// ReadPacketData reads chCaptured channel and returns packet data and metadata.
// This method is from PacketDataSource interface.
func (s *Server) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	p := <-s.ChIn
	return p.Data, p.CaptureInfo, nil
}

func Now() uint64 {
	return uint64(time.Now().UnixNano()) * uint64(time.Nanosecond) / uint64(time.Millisecond)
}
