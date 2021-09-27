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
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	MLinkHostAddr = 1
	MLinkDeviceAddr = 0xfefe
)

func init() {
	initUnknownMLinkTypes()
	initActualMLinkTypes()
}

const (
	// MLinkLayerNum identifies the layer
	MLinkLayerNum = 1999
	// MLinkEndPointNum
	MLinkEndpointNum = 1000
	// MLinkSync is a magic number that appears in the beginning of each MLink frame
	MLinkSync = 0x2A50
	// MLink is the last word of each MLink frame, they call it MLINK_DATA_PADDING_MAGIC or CRC
	// For MStream ACK frames it is 0x00000000
	// For MStream frames sent from a device to host it is 0x12206249
	// For register r/w request it is crc32 sum
	// For register r/w response it is 0x00000000
	MLinkMStreamCRC = 0x12206249
	// MLinkMaxFrameSize is the max size of MLink frame including MLink header and CRC
	MLinkMaxFrameSize = 1400
	// MLinkMaxPayloadSize is the max size of Mlink frame payload
	// MLink header 12 bytes
	// MLink CRC 4 bytes
	MLinkMaxPayloadSize = MLinkMaxFrameSize - 16
)

type MLinkType uint16

const (
	// TODO add other MLink types once they are implemented
	MLinkTypeMStream MLinkType = 0x5354
	MLinkTypeRegRequest MLinkType = 0x0101
	MLinkTypeRegResponse MLinkType = 0x0102
	MLinkTypeMemRequest MLinkType = 0x0105
	MLinkTypeMemResponse MLinkType = 0x0106
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
	MLinkMetadata[MLinkTypeMStream] = layers.EnumMetadata{DecodeWith: gopacket.DecodeFunc(DecodeMStreamLayer), Name: "MStream", LayerType: MStreamLayerType}
	MLinkMetadata[MLinkTypeRegResponse] = layers.EnumMetadata{DecodeWith: gopacket.DecodeFunc(DecodeRegLayer), Name: "Reg", LayerType: RegLayerType}
	MLinkMetadata[MLinkTypeMemResponse] = layers.EnumMetadata{DecodeWith: gopacket.DecodeFunc(DecodeMemLayer), Name: "Mem", LayerType: MemLayerType}
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
	Type MLinkType
	Sync uint16
	Seq uint16
	Len uint16 // length of MLink frame including header, payload and CRC in 4-byte words NOT in bytes
	Src uint16
	Dst uint16
}

type MLinkLayer struct {
	layers.BaseLayer
	MLinkHeader
	Crc uint32
}

var MLinkLayerType = gopacket.RegisterLayerType(MLinkLayerNum,
	gopacket.LayerTypeMetadata{Name: "MLinkLayerType", Decoder: gopacket.DecodeFunc(decodeMLinkLayer)})

func (ml *MLinkLayer) LayerType() gopacket.LayerType {
	return MLinkLayerType
}

// SerializeHeader serializes only MLink header (not tail) to a buffer
// This is necessary because CRC field depends on the contents of the MLink frame
// and we calculate it in upper layers manually using serialized MLink header.
// Otherwise CRC calculates could encapsulated into SerializeTo method.
func (ml *MLinkLayer) SerializeHeader(buf []byte) {
	binary.LittleEndian.PutUint16(buf[0:2], uint16(ml.Type))
	binary.LittleEndian.PutUint16(buf[2:4], ml.Sync)
	binary.LittleEndian.PutUint16(buf[4:6], ml.Seq)
	binary.LittleEndian.PutUint16(buf[6:8], ml.Len)
	binary.LittleEndian.PutUint16(buf[8:10], ml.Src)
	binary.LittleEndian.PutUint16(buf[10:12], ml.Dst)
}

// SerializeTo serializes the layer into bytes and writes the bytes to the SerializeBuffer
func (ml *MLinkLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	headerBytes, err := b.PrependBytes(12)
	if err != nil {
		return err
	}
	ml.SerializeHeader(headerBytes)

	tailBytes, err := b.AppendBytes(4)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(tailBytes[0:4], ml.Crc)
	return nil
}

// DecodeFromBytes attempts to decode the byte slice as a MLink frame
func (ml *MLinkLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	if len(data) < 16 {
		df.SetTruncated()
		return errors.New("MLink packet too short")
	}

	if binary.LittleEndian.Uint16(data[2:4]) != MLinkSync {
		log.Debug("Mlink sync is invalid")
		return errors.New(fmt.Sprintf("Wrong MLink sync. Must be %d", MLinkSync))
	}

	ml.BaseLayer = layers.BaseLayer{
		Contents: data[0:12], // MLink header 12 bytes
		Payload: data[12:len(data)-4], // data without MLink header and without CRC in the end of each MLink frame
	}

	ml.Type = MLinkType(binary.LittleEndian.Uint16(data[0:2]))
	ml.Sync = binary.LittleEndian.Uint16(data[2:4])
	ml.Seq = binary.LittleEndian.Uint16(data[4:6])
	ml.Len = binary.LittleEndian.Uint16(data[6:8])
	ml.Src = binary.LittleEndian.Uint16(data[8:10])
	ml.Dst = binary.LittleEndian.Uint16(data[10:12])
	ml.Crc = binary.LittleEndian.Uint32(data[len(data)-4:])

	// TODO Discuss with AFI and unificate CRC to be crc32 sum
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	// This check is only valid for MStream
	if ml.Type == MLinkTypeMStream && ml.Crc != MLinkMStreamCRC {
		return errors.New(fmt.Sprintf("Wrong MLink tail for MStream frame. Must be 0x%08x", MLinkMStreamCRC))
	}

	return nil
}

func (ml *MLinkLayer) NextLayerType() gopacket.LayerType {
	return ml.Type.LayerType()
}

func decodeMLinkLayer(data []byte, p gopacket.PacketBuilder) error {
	ml := &MLinkLayer{}
	err := ml.DecodeFromBytes(data, p)
	if err != nil {
		log.Error("Error while decoding mlink layer: %s", err)
		return err
	}
	p.AddLayer(ml)
	return p.NextDecoder(ml.NextLayerType())
}
