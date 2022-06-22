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
	"fmt"
	"hash/crc32"
	"strconv"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	// RegLayerNum identifies the layer
	RegLayerNum = 1997
)

type Reg struct {
	Addr  uint16
	Value uint16
}

func (r *Reg) String() string {
	hexAddr, hexValue := r.Hex()
	return fmt.Sprintf("addr: %s value: %s", hexAddr, hexValue)
}

func (reg *Reg) Hex() (string, string) {
	return fmt.Sprintf("0x%04x", reg.Addr), fmt.Sprintf("0x%04x", reg.Value)
}

func NewRegFromHex(hexAddr, hexValue string) (*Reg, error) {
	addr, err := strconv.ParseUint(hexAddr, 0, 16)
	if err != nil {
		return nil, err
	}
	value, err := strconv.ParseUint(hexValue, 0, 16)
	if err != nil {
		return nil, err
	}
	return &Reg{
		Addr:  uint16(addr),
		Value: uint16(value),
	}, nil
}

type RegOp struct {
	// if Read is true, Reg.Value is ignored
	Read bool
	*Reg
}

func (ro *RegOp) String() string {
	return fmt.Sprintf("read: %t %s", ro.Read, ro.Reg)
}

type RegLayer struct {
	layers.BaseLayer
	RegOps []*RegOp
}

var RegLayerType = gopacket.RegisterLayerType(RegLayerNum,
	gopacket.LayerTypeMetadata{Name: "RegLayerType", Decoder: gopacket.DecodeFunc(DecodeRegLayer)})

// LayerType returns the type of the Reg layer in the layer catalog
func (reg *RegLayer) LayerType() gopacket.LayerType {
	return RegLayerType
}

// Serialize serializes the RegRWRequestLayer to a buffer.
// This is necessary because MLink CRC field depends on the contents of the MLink frame
// and sometimes we have to calculate it manually in upper layers instead of encapsulating
// it to MLinkLayer.SerializeTo method.
func (reg *RegLayer) Serialize(buf []byte) {
	for i, op := range reg.RegOps {
		log.Debug("Serializing RegOp: %s", op)
		offset := i * 4
		if op.Read {
			binary.LittleEndian.PutUint32(buf[offset:offset+4], 0x80000000|((uint32(op.Addr)&0x7fff)<<16))
		} else {
			binary.LittleEndian.PutUint32(buf[offset:offset+4], 0x00000000|((uint32(op.Addr)&0x7fff)<<16)|uint32(op.Value))
		}
		log.Debug("Serialized RegOp: 0x%08x", buf[offset:offset+4])
	}
}

// SerializeTo serializes the register read/write request layer into bytes and writes the bytes to the SerializeBuffer
func (reg *RegLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	bytes, err := b.AppendBytes(len(reg.RegOps) * 4)
	if err != nil {
		return err
	}
	reg.Serialize(bytes)
	return nil
}

func (reg *RegLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	reg.BaseLayer = layers.BaseLayer{
		Contents: data,
		Payload:  []byte{},
	}
	for i := 0; i < len(data)/4; i++ {
		offset := i * 4
		word := binary.LittleEndian.Uint32(data[offset+0 : offset+4])
		regOp := &RegOp{Reg: &Reg{}}
		if ((word & 0x80000000) >> 31) == 1 {
			regOp.Read = true
		} else {
			regOp.Read = false
		}
		regOp.Addr = uint16((word & 0x7fff0000) >> 16)
		regOp.Value = uint16(word & 0x0000ffff)
		reg.RegOps = append(reg.RegOps, regOp)
	}
	return nil
}

func DecodeRegLayer(data []byte, p gopacket.PacketBuilder) error {
	log.Debug("Trying to decode RegLayer, data len: %d", len(data))
	reg := &RegLayer{}
	err := reg.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(reg)
	return nil
}

// RegOpToBytes ...
func RegOpsToBytes(ops []*RegOp, seq uint16) ([]byte, error) {
	ml := &MLinkLayer{}
	ml.Type = MLinkTypeRegRequest
	ml.Sync = MLinkSync
	// 3 words for MLink header + 1 word CRC + N words for request
	ml.Len = uint16(4 + len(ops))
	ml.Seq = seq
	ml.Src = MLinkHostAddr
	ml.Dst = MLinkDeviceAddr

	// Calculate crc32 checksum
	mlHeaderBytes := make([]byte, 12)
	ml.SerializeHeader(mlHeaderBytes)

	reg := &RegLayer{}
	reg.RegOps = ops
	regBytes := make([]byte, len(ops)*4)
	reg.Serialize(regBytes)

	ml.Crc = crc32.ChecksumIEEE(append(mlHeaderBytes, regBytes...))

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, ml, reg)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
