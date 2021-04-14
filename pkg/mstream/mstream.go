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
	"encoding/binary"
	"errors"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	// MStreamLayerNum identifies the layer number
	MStreamLayerNum = 1998
	// MLinkType appears in the Type field of the MLink frame header and means the payload type
	// which is aslways the same since we always deal with MStream data
	// MLinkType = 0x5354

)

type MStreamHeader struct {
	FragmentLen uint16
	Subtype uint8
	Flags uint8
	DeviceID uint8 // it is always 0xd9 which corresponds to adc64ve-xge
	FragmentID uint8
	FragmentOffset uint32
	DeviceSerial uint32
	UserDefBytes uint8
	EventNum uint32
}
// TODO implement two additional methods for MStreamHeader GetFragmentID and GetFragmentOffset

type MStreamLayer struct {
	layers.BaseLayer
	MStreamHeader
}

var MStreamLayerType = gopacket.RegisterLayerType(MStreamLayerNum,
	gopacket.LayerTypeMetadata{Name: "MStreamLayerType", Decoder: gopacket.DecodeFunc(decodeMStreamLayer)})

// LayerType returns the type of the MStream layer in the layer catalog
func (ms *MStreamLayer) LayerType() gopacket.LayerType {
	return MStreamLayerType
}

// SerializeTo serializes the layer into bytes and writes the bytes to the SerializeBuffer
func (ms *MStreamLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	return nil
}

func (ms *MStreamLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	if len(data) < 16 {
		df.SetTruncated()
		return errors.New("MStream packet too short")
	}

	ms.BaseLayer = layers.BaseLayer{
		Contents: data[0:16], // MStream header 16 bytes
		Payload: data[16:], // data without MStream header
	}

	ms.FragmentLen = binary.BigEndian.Uint16(data[0:2])
	if ms.FragmentLen == 0 {
		return errors.New("Invalid MStream header: FragmentLen = 0")
	}

	ms.Subtype = data[2] & 0b00000011 // Subtype is two least significant bits
	ms.Flags = (data[2] >> 2) & 0b00111111 // Flags is six high bits
	ms.DeviceID = data[3]
	ms.FragmentID = data[4]
	ms.FragmentOffset = binary.BigEndian.Uint32(data[5:8])
	ms.DeviceSerial = binary.BigEndian.Uint32(data[8:12])
	ms.UserDefBytes = data[12]
	ms.EventNum = binary.BigEndian.Uint32(data[13:16])

	return nil
}

func decodeMStreamLayer(data []byte, p gopacket.PacketBuilder) error {
	ms := &MStreamLayer{}
	err := ms.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(ms)
	return nil
}
