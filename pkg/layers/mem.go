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
	// MemLayerNum identifies the layer
	MemLayerNum = 1996
)

type MemOp struct {
	Read bool
	Addr uint32
	Size uint32
	Data []uint32 // if Read is true, Value is ignored
}

type MemLayer struct {
	layers.BaseLayer
	*MemOp
}

var MemLayerType = gopacket.RegisterLayerType(MemLayerNum,
	gopacket.LayerTypeMetadata{Name: "MemLayerType", Decoder: gopacket.DecodeFunc(DecodeMemLayer)})

// LayerType returns the type of the MStream layer in the layer catalog
func (reg *MemLayer) LayerType() gopacket.LayerType {
	return MemLayerType
}

// Serialize serializes the MemRWRequestLayer to a buffer.
// This is necessary because MLink CRC field depends on the contents of the MLink frame
// and sometimes we have to calculate it manually in upper layers instead of encapsulating
// it to MLinkLayer.SerializeTo method.
func (mem *MemLayer) Serialize(buf []byte) {
	if mem.MemOp.Read {
		binary.LittleEndian.PutUint32(buf[0:4], 0x80000000|((uint32(mem.Size)&0x1ff)<<22)|(uint32(mem.Addr)&0x3fffff))
	} else {
		binary.LittleEndian.PutUint32(buf[0:4], 0x00000000|((uint32(mem.Size)&0x1ff)<<22)|(uint32(mem.Addr)&0x3fffff))
		for i, word := range(mem.Data)  {
			offset := (i + 1) * 4
			binary.LittleEndian.PutUint32(buf[offset:offset+4], word)
		}
	}
}

// SerializeTo serializes the register read/write request layer into bytes and writes the bytes to the SerializeBuffer
func (mem *MemLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	bytes, err := b.AppendBytes(4)
	if err != nil {
		return err
	}
	mem.Serialize(bytes)
	return nil
}

func (mem *MemLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	mem.BaseLayer = layers.BaseLayer{
		Contents: data[:],
		Payload: []byte{},
	}
	hdr := binary.LittleEndian.Uint32(data[0:4])
	if int8((hdr & 0x80000000) >> 31) == 1 {
		mem.Read = true
	} else {
		mem.Read = false
	}
	mem.Addr = hdr & 0x3fffff
	mem.Size = (hdr >> 22) & 0x1ff
	for i := uint32(0); i < mem.Size; i++ {
		mem.Data = append(mem.Data, binary.LittleEndian.Uint32(data[i+4:i+8]))
	}
	return nil
}

func DecodeMemLayer(data []byte, p gopacket.PacketBuilder) error {
	req := &MemLayer{}
	err := req.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(req)
	return nil
}
