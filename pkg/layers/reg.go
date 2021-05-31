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
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	// RegLayerNum identifies the layer
	RegLayerNum = 1997
)

type RegLayer struct {
	layers.BaseLayer
	Read bool
	RegNum uint16
	RegValue uint16 // if Read is true, RegValue is ignored
}

var RegLayerType = gopacket.RegisterLayerType(RegLayerNum,
	gopacket.LayerTypeMetadata{Name: "RegLayerType", Decoder: gopacket.DecodeFunc(DecodeRegLayer)})

// LayerType returns the type of the MStream layer in the layer catalog
func (reg *RegLayer) LayerType() gopacket.LayerType {
	return RegLayerType
}

// Serialize serializes the RegRWRequestLayer to a buffer.
// This is necessary because MLink CRC field depends on the contents of the MLink frame
// and sometimes we have to calculate it manually in upper layers instead of encapsulating
// it to MLinkLayer.SerializeTo method.
func (reg *RegLayer) Serialize(buf []byte) {
	if reg.Read {
		binary.LittleEndian.PutUint32(buf[0:4], 0x80000000 | ((uint32(reg.RegNum) & 0x7fff) << 16))
	} else {
		binary.LittleEndian.PutUint32(buf[0:4], 0x00000000 | ((uint32(reg.RegNum) & 0x7fff) << 16) | uint32(reg.RegValue))
	}
}

// SerializeTo serializes the register read/write request layer into bytes and writes the bytes to the SerializeBuffer
func (reg *RegLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	bytes, err := b.AppendBytes(4)
	if err != nil {
		return err
	}
	reg.Serialize(bytes)
	return nil
}

func (reg *RegLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	reg.BaseLayer = layers.BaseLayer{
		Contents: data[:],
		Payload: []byte{},
	}
	word := binary.LittleEndian.Uint32(data[0:4])
	if int8((word & 0x80000000) >> 31) == 1 {
		reg.Read = true
	} else {
		reg.Read = false
	}
	reg.RegNum = uint16((word & 0x7fff0000) >> 16)
	reg.RegValue = uint16(word & 0x0000ffff)

	return nil
}

func DecodeRegLayer(data []byte, p gopacket.PacketBuilder) error {
	req := &RegLayer{}
	err := req.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(req)
	return nil
}
