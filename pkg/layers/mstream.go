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
	"encoding/hex"
	"errors"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	// MStreamLayerNum identifies the layer
	MStreamLayerNum = 1998
)

type Subtype uint8

const (
	MStreamDataSubtype Subtype = iota
)

// mstream-lib/types.h

// MStreamPayloadHeader ... // 7 bytes
type MStreamPayloadHeader struct {
	DeviceSerial uint32
	EventNum     uint32 // 24 bits
}

// MStreamData ...
type MStreamData struct {
	Bytes []byte
}

// MStreamFragment ...
type MStreamFragment struct {
	FragmentLength uint16 // length of fragment payload NOT including MStream header in bytes
	Subtype               // 2 bits
	Flags          uint8  // 6 bits
	// DeviceID is the ADC64 device model identifier
	// 0xd9 for ADC64VE-XGE
	// 0xdf for ADC64VE-V3-XG
	DeviceID       uint8
	FragmentID     uint16
	FragmentOffset uint16
	Data           []byte

	*MStreamPayloadHeader
	*MStreamData
}

// MStreamLayer ...
type MStreamLayer struct {
	layers.BaseLayer
	Fragments []*MStreamFragment
}

var MStreamLayerType = gopacket.RegisterLayerType(MStreamLayerNum,
	gopacket.LayerTypeMetadata{Name: "MStreamLayerType", Decoder: gopacket.DecodeFunc(DecodeMStreamLayer)})

// LayerType returns the type of the MStream layer in the layer catalog
func (ms *MStreamLayer) LayerType() gopacket.LayerType {
	return MStreamLayerType
}

// Serialize MStreamData ...
func (d *MStreamData) Serialize(buf []byte) error {
	copy(buf, d.Bytes)
	return nil
}

// SerializeTo serializes the MStream layer into bytes and writes the bytes to the SerializeBuffer
func (ms *MStreamLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	for _, fragment := range ms.Fragments {
		headerBytes, err := b.AppendBytes(8)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint16(headerBytes[0:2], fragment.FragmentLength)
		headerBytes[2] = (fragment.Flags << 2) | uint8(fragment.Subtype)
		headerBytes[3] = fragment.DeviceID
		binary.LittleEndian.PutUint32(headerBytes[4:8], (uint32(fragment.FragmentID)<<16)|uint32(fragment.FragmentOffset))

		payloadBytes, err := b.AppendBytes(int(fragment.FragmentLength))
		if err != nil {
			return err
		}
		copy(payloadBytes, fragment.Data)
	}

	return nil
}

// DecodeMStreamPayloadHeader ...
func DecodeMStreamPayloadHeader(fragmentPayload []byte) (*MStreamPayloadHeader, error) {
	if len(fragmentPayload) < 8 {
		return nil, errors.New("MStream data packet too short. Must at least have data header.")
	}
	// TODO: perhaps it could be more pretty
	eventNumBytes := make([]byte, 4)
	copy(eventNumBytes, fragmentPayload[4:7])

	log.Debug("DecodeMStreamPayloadHeader: DeviceSerial: %08x", binary.LittleEndian.Uint32(fragmentPayload[0:4]))
	log.Debug("DecodeMStreamPayloadHeader: EventNum: %d", binary.LittleEndian.Uint32(eventNumBytes))

	return &MStreamPayloadHeader{
		DeviceSerial: binary.LittleEndian.Uint32(fragmentPayload[0:4]),
		EventNum:     binary.LittleEndian.Uint32(eventNumBytes),
	}, nil
}

// DecodeMStreamData ...
func DecodeMStreamData(fragmentPayload []byte) (*MStreamData, error) {
	log.Debug("DecodeMStreamData: Bytes:\n%s", hex.Dump(fragmentPayload[8:]))
	return &MStreamData{
		Bytes: fragmentPayload[8:],
	}, nil
}

// DecodeFragment ...
// offset is the beginning of MStream fragment inside MStream packet
// data is the whole MStream packet
func (ms *MStreamLayer) DecodeFragment(offset int, data []byte) (int, error) {
	log.Debug("DecodeFragment: ~~~~~~~")
	log.Debug("DecodeFragment: offset: %d", offset)

	// Decoding fragment header
	fragmentLength := binary.LittleEndian.Uint16(data[offset : offset+2])
	if fragmentLength == 0 {
		return offset, errors.New("Invalid MStream fragment: FragmentLength = 0")
	}
	// end of fragment is current offset + size of fragment header + fragment length
	newOffset := offset + 8 + int(fragmentLength)
	log.Debug("DecodeFragment: newOffset: %d", newOffset)
	log.Debug("DecodeFragment: fragment data: \n%s", hex.Dump(data[offset:newOffset]))

	subtype := data[offset+2] & 0x3       // Subtype is two least significant bits
	flags := (data[offset+2] >> 2) & 0x3f // Flags is six high bits
	deviceID := data[offset+3]
	fragmentOffsetID := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
	fragmentID := uint16(fragmentOffsetID >> 16)        // FragmentID takes 2 bytes for MStream 2.x
	fragmentOffset := uint16(fragmentOffsetID & 0xffff) // FragmentOffset takes 2 bytes for MStream 2.x

	log.Debug("DecodeFragment: FragmentLength: %d", fragmentLength)
	log.Debug("DecodeFragment: Subtype: %d", subtype)
	log.Debug("DecodeFragment: Flags: %d", flags)
	log.Debug("DecodeFragment: DeviceID: 0x%02x", deviceID)
	log.Debug("DecodeFragment: FragmentID: 0x%04x", fragmentID)
	log.Debug("DecodeFragment: FragmentOffset: 0x%04x", fragmentOffset)

	fragment := &MStreamFragment{
		FragmentLength: fragmentLength,
		Subtype:        Subtype(subtype),
		Flags:          flags,
		DeviceID:       deviceID,
		FragmentID:     fragmentID,
		FragmentOffset: fragmentOffset,
		Data:           data[offset+8 : newOffset],
	}

	ms.Fragments = append(ms.Fragments, fragment)

	return newOffset, nil
}

func (ms *MStreamLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	log.Debug("DecodeFromBytes: start")
	defer log.Debug("DecodeFromBytes: stop")
	log.Debug("DecodeFromBytes: data length: %d", len(data))
	log.Debug("DecodeFromBytes: data: \n%s", hex.Dump(data))

	// At least one fragment must be in the packet and fragment header length is 8
	if len(data) < 8 {
		df.SetTruncated()
		// TODO return custom error
		return errors.New("MStream packet too short")
	}

	// MStream layer consists of fragments without common layer header
	ms.BaseLayer = layers.BaseLayer{
		Contents: []byte{},
		Payload:  data,
	}

	var err error
	offset := 0
	for offset < len(data) {
		offset, err = ms.DecodeFragment(offset, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *MStreamFragment) LastFragment() bool {
	return ((f.Flags >> 5) & 0b00000001) == 1
}

func (f *MStreamFragment) SetLastFragment(last bool) {
	if last {
		f.Flags |= 0b00100000
	} else {
		f.Flags &= 0b11011111
	}
}

func (f *MStreamFragment) Ack() bool {
	return ((f.Flags >> 4) & 0b00000001) == 1
}

func (f *MStreamFragment) SetAck(ack bool) {
	if ack {
		f.Flags |= 0b00010000
	} else {
		f.Flags &= 0b11101111
	}
}

// DecodePayload decodes payload of MStream fragment
// It is assumed to be one of MStreamTrigger or MStreamData
// This method must be called only for defragmented (assembled) frames
func (f *MStreamFragment) DecodePayload() error {
	log.Debug("DecodePayload: length: %d", len(f.Data))
	log.Debug("DecodePayload: data: \n%s\n", hex.Dump(f.Data))
	payloadHeader, err := DecodeMStreamPayloadHeader(f.Data)
	if err != nil {
		return errors.New("Error while decoding payload header of MStream fragment")
	}
	f.MStreamPayloadHeader = payloadHeader
	// for tqdc the subtype is always 0
	if f.Subtype != 0 {
		return errors.New("Unknown fragment subtype. It must be 0.")
	}
	data, err := DecodeMStreamData(f.Data)
	if err != nil {
		return errors.New("Error while decoding payload of MStream data fragment")
	}
	f.MStreamData = data
	return nil
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
