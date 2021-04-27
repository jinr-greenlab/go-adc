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
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)


func init() {
	initUnknownMLinkTypes()
	initActualMLinkTypes()
}

const (
	// MLinkLayerNum identifies the layer number
	MLinkLayerNum = 1999
	// MLinkEndPointNum
	MLinkEndpointNum = 1000
	// MLinkSync is a magic number that appears in the beginning of each MLink frame
	MLinkSync = 0x2A50
	// MLink is the last word of each MLink frame, they call it MLINK_DATA_PADDING_MAGIC or CRC
	MLinkCRC = 0x12206249
)

type MLinkType uint16

const (
	// TODO add other MLink types once they are implemented
	MLinkTypeMStream MLinkType = 0x5354
)

type errorDecoderForMLinkType int

func (e *errorDecoderForMLinkType) Decode(data []byte, p gopacket.PacketBuilder) error {
	return e
}

func (e *errorDecoderForMLinkType) Error() string {
	return fmt.Sprintf("Unable to decode MLink type %d", int(*e))
}

var errorDecodersForMLinkType [65536]errorDecoderForMLinkType
var MLinkMetadata [65536]layers.EnumMetadata

func initUnknownMLinkTypes() {
	for i := 0; i < 65536; i++ {
		errorDecodersForMLinkType[i] = errorDecoderForMLinkType(i)
		MLinkMetadata[i] = layers.EnumMetadata{
			DecodeWith: &errorDecodersForMLinkType[i],
			Name:       "UnknownMLinkType",
		}
	}
}

func initActualMLinkTypes() {
	// TODO init other MLink types once they are implemented
	MLinkMetadata[MLinkTypeMStream] = layers.EnumMetadata{DecodeWith: gopacket.DecodeFunc(decodeMStreamLayer), Name: "MStream", LayerType: MStreamLayerType}
}

// LayerType returns MLinkMetadata.LayerType
func (t MLinkType) LayerType() gopacket.LayerType {
	return MLinkMetadata[t].LayerType
}

// Decode calls MLinkMetadata.DecodeWith's decoder
func (t MLinkType) Decode(data []byte, p gopacket.PacketBuilder) error {
	return MLinkMetadata[t].DecodeWith.Decode(data, p)
}

// String returns MLinkMetadata.Name
func (t MLinkType) String() string {
	return MLinkMetadata[t].Name
}

type MLinkHeader struct {
	Sync uint16
	Type MLinkType
	Seq uint16
	Len uint16
	Src uint16
	Dst uint16
}

type MLinkLayer struct {
	layers.BaseLayer
	MLinkHeader
}

var MLinkLayerType = gopacket.RegisterLayerType(MLinkLayerNum,
	gopacket.LayerTypeMetadata{Name: "MLinkLayerType", Decoder: gopacket.DecodeFunc(decodeMLinkLayer)})

func (ml *MLinkLayer) LayerType() gopacket.LayerType {
	return MLinkLayerType
}

// SerializeTo serializes the layer into bytes and writes the bytes to the SerializeBuffer
func (ml *MLinkLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	headerBytes, err := b.PrependBytes(12)
	if err != nil {
		return err
	}

	binary.BigEndian.PutUint16(headerBytes[0:2], ml.Sync)
	binary.BigEndian.PutUint16(headerBytes[2:4], uint16(ml.Type))
	binary.BigEndian.PutUint16(headerBytes[4:6], ml.Seq)
	binary.BigEndian.PutUint16(headerBytes[6:8], ml.Len)
	binary.BigEndian.PutUint16(headerBytes[8:10], ml.Src)
	binary.BigEndian.PutUint16(headerBytes[10:12], ml.Dst)

	tailBytes, err := b.AppendBytes(4)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(tailBytes[0:4], MLinkCRC)
	return nil
}

// DecodeFromBytes attempts to decode the byte slice as a MLink frame
func (ml *MLinkLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	if len(data) < 16 {
		df.SetTruncated()
		return errors.New("MLink packet too short")
	}

	if binary.BigEndian.Uint16(data[0:2]) != MLinkSync {
		return errors.New(fmt.Sprintf("Wrong MLink sync. Must be %d", MLinkSync))
	}

	if binary.BigEndian.Uint32(data[len(data)-4:]) != MLinkCRC {
		return errors.New(fmt.Sprintf("Wrong MLink tail. Must be %d", MLinkCRC))
	}

	ml.BaseLayer = layers.BaseLayer{
		Contents: data[0:12], // MLink header 12 bytes
		Payload: data[12:len(data)-4], // data without MLink header and without CRC in the end of each MLink frame
	}

	ml.Sync = binary.BigEndian.Uint16(data[0:2])
	ml.Type = MLinkType(binary.BigEndian.Uint16(data[2:4]))
	ml.Seq = binary.BigEndian.Uint16(data[4:6])
	ml.Len = binary.BigEndian.Uint16(data[6:8])
	ml.Src = binary.BigEndian.Uint16(data[8:10])
	ml.Dst = binary.BigEndian.Uint16(data[10:12])

	return nil
}

func (ml *MLinkLayer) NextLayerType() gopacket.LayerType {
	return ml.Type.LayerType()
}

func decodeMLinkLayer(data []byte, p gopacket.PacketBuilder) error {
	ml := &MLinkLayer{}
	err := ml.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(ml)
	return p.NextDecoder(ml.NextLayerType())
}

func (ml *MLinkLayer) Flow() gopacket.Flow {
	src := make([]byte, 2)
	dst := make([]byte, 2)
	binary.BigEndian.PutUint16(src, ml.Src)
	binary.BigEndian.PutUint16(dst, ml.Dst)
	return gopacket.NewFlow(EndpointMLink, src, dst)
}

var (
	EndpointMLink = gopacket.RegisterEndpointType(MLinkEndpointNum, gopacket.EndpointTypeMetadata{Name: "MLink", Formatter: func(b []byte) string {
		return string(b)
	}})
)