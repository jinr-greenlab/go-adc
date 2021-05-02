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

package layers

import (
	"encoding/binary"
	"errors"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	// MStreamLayerNum identifies the layer number
	MStreamLayerNum = 1998
)

type MStreamHeader struct {
	DeviceID       uint8 // it is always 0xd9 which corresponds to adc64ve-xge
	Flags          uint8
	Subtype        uint8
	FragmentLength uint16 // length of fragment payload NOT including MStream header
	FragmentID     uint16
	FragmentOffset uint16
}

type MStreamLayer struct {
	layers.BaseLayer
	MStreamHeader
}

var MStreamLayerType = gopacket.RegisterLayerType(MStreamLayerNum,
	gopacket.LayerTypeMetadata{Name: "MStreamLayerType", Decoder: gopacket.DecodeFunc(DecodeMStreamLayer)})

// LayerType returns the type of the MStream layer in the layer catalog
func (ms *MStreamLayer) LayerType() gopacket.LayerType {
	return MStreamLayerType
}

// SerializeTo serializes the MStream layer into bytes and writes the bytes to the SerializeBuffer
func (ms *MStreamLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	headerBytes, err := b.PrependBytes(8)
	if err != nil {
		return err
	}

	//binary.BigEndian.PutUint16(headerBytes[0:2], ms.FragmentLength)
	binary.LittleEndian.PutUint16(headerBytes[0:2], ms.FragmentLength)

	headerBytes[2] = (ms.Flags << 2) | ms.Subtype
	headerBytes[3] = ms.DeviceID
	binary.LittleEndian.PutUint16(headerBytes[4:6], ms.FragmentID)
	binary.LittleEndian.PutUint16(headerBytes[6:8], ms.FragmentOffset)
	return nil
}

func (ms *MStreamLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	if len(data) < 16 {
		df.SetTruncated()
		// TODO return custom error
		return errors.New("MStream packet too short")
	}

	ms.BaseLayer = layers.BaseLayer{
		Contents: data[0:16], // MStream header 16 bytes
		Payload: data[16:], // data without MStream header
	}

	ms.FragmentLength = binary.BigEndian.Uint16(data[0:2])
	if ms.FragmentLength == 0 {
		return errors.New("Invalid MStream header: FragmentLength = 0")
	}

	ms.Subtype = data[2] & 0b00000011 // Subtype is two least significant bits
	ms.Flags = (data[2] >> 2) & 0b00111111 // Flags is six high bits
	ms.DeviceID = data[3]
	ms.FragmentID = binary.BigEndian.Uint16(data[4:6]) // FragmentID takes 2 bytes for MStream 2.x
	ms.FragmentOffset = binary.BigEndian.Uint16(data[6:8]) // FragmentOffset takes 2 bytes for MStream 2.x

	return nil
}

func (ms *MStreamLayer) LastFragment() bool {
	return ((ms.Flags >> 5) & 0b00000001) == 1
}

func (ms *MStreamLayer) SetLastFragment(last bool) {
	if last {
		ms.Flags |= 0b00100000
	} else {
		ms.Flags &= 0b11011111
	}
}

func (ms *MStreamLayer) Ack() bool {
	return ((ms.Flags >> 4) & 0b00000001) == 1
}

func (ms *MStreamLayer) SetAck(ack bool) {
	if ack {
		ms.Flags |= 0b00010000
	} else {
		ms.Flags &= 0b11101111
	}
}

func DecodeMStreamLayer(data []byte, p gopacket.PacketBuilder) error {
	ms := &MStreamLayer{}
	err := ms.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(ms)
	return nil
}
