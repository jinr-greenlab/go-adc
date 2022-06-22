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
	MStreamTriggerSubtype Subtype = iota
	MStreamDataSubtype
)

// mstream-lib/types.h

type ChannelNum uint8

// MStreamPayloadHeader ... // 7 bytes
type MStreamPayloadHeader struct {
	DeviceSerial uint32
	EventNum     uint32 // 24 bits
	ChannelNum          // for trigger it is always 0
}

// MStreamTrigger ... // 16 bytes
type MStreamTrigger struct {
	TaiSec  uint32
	Flags   uint8  // 2 bits
	TaiNSec uint32 // 30 bits
	// TODO: Put these two fields into one uint64
	LowCh uint32
	HiCh  uint32
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
	// Fragment contains either MStreamTrigger or MStreamData, not both of them at the same time
	*MStreamTrigger
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

// Serialize MStreamPayloadHeader ...
func (h *MStreamPayloadHeader) Serialize(buf []byte) error {
	binary.LittleEndian.PutUint32(buf[0:4], h.DeviceSerial)
	buf[4] = uint8(h.EventNum & 0xff)
	binary.LittleEndian.PutUint16(buf[5:7], uint16((h.EventNum&0xffff00)>>8))
	buf[7] = uint8(h.ChannelNum)
	return nil
}

// Serialize MStreamTrigger ...
func (t *MStreamTrigger) Serialize(buf []byte) error {
	log.Debug("MStreamTrigger.Serialize: TaiSec: %d", t.TaiSec)
	log.Debug("MStreamTrigger.Serialize: hex TaiSec: %08x", t.TaiSec)
	log.Debug("MStreamTrigger.Serialize: TaiNSec: %d", t.TaiNSec)
	log.Debug("MStreamTrigger.Serialize: Flags: %d", t.Flags)
	log.Debug("MStreamTrigger.Serialize: hex TaiNSec/Flags: %08x", (t.TaiNSec<<2 | uint32(t.Flags)))
	log.Debug("MStreamTrigger.Serialize: LowCh: %d", t.LowCh)
	log.Debug("MStreamTrigger.Serialize: HiCh: %d", t.HiCh)

	binary.LittleEndian.PutUint32(buf[0:4], t.TaiSec)
	binary.LittleEndian.PutUint32(buf[4:8], (t.TaiNSec<<2 | uint32(t.Flags)))
	binary.LittleEndian.PutUint32(buf[8:12], t.LowCh)
	binary.LittleEndian.PutUint32(buf[12:16], t.HiCh)
	return nil
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
	log.Debug("DecodeMStreamPayloadHeader: ChannelNum: %d", fragmentPayload[7])

	return &MStreamPayloadHeader{
		DeviceSerial: binary.LittleEndian.Uint32(fragmentPayload[0:4]),
		EventNum:     binary.LittleEndian.Uint32(eventNumBytes),
		ChannelNum:   ChannelNum(fragmentPayload[7]),
	}, nil
}

// DecodeMStreamTrigger ...
func DecodeMStreamTrigger(fragmentPayload []byte) (*MStreamTrigger, error) {
	if len(fragmentPayload) < 24 {
		return nil, errors.New("MStream trigger packet too short. Must be at least 24 bytes.")
	}

	taiNSecFlags := binary.LittleEndian.Uint32(fragmentPayload[12:16])

	log.Debug("DecodeMStreamTrigger: TaiSec: %d", binary.LittleEndian.Uint32(fragmentPayload[8:12]))
	log.Debug("DecodeMStreamTrigger: Flags: %d", uint8(taiNSecFlags&0x3))
	log.Debug("DecodeMStreamTrigger: TaiNSec: %d", taiNSecFlags>>2)
	log.Debug("DecodeMStreamTrigger: LowCh: %d", binary.LittleEndian.Uint32(fragmentPayload[16:20]))
	log.Debug("DecodeMStreamTrigger: HiCh: %d", binary.LittleEndian.Uint32(fragmentPayload[20:24]))

	return &MStreamTrigger{
		TaiSec:  binary.LittleEndian.Uint32(fragmentPayload[8:12]),
		Flags:   uint8(taiNSecFlags & 0x3),
		TaiNSec: taiNSecFlags >> 2,
		LowCh:   binary.LittleEndian.Uint32(fragmentPayload[16:20]),
		HiCh:    binary.LittleEndian.Uint32(fragmentPayload[20:24]),
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
	log.Debug("DecodeFragment: DeviceID: %d", deviceID)
	log.Debug("DecodeFragment: FragmentID: %d", fragmentID)
	log.Debug("DecodeFragment: FragmentOffset: %d", fragmentOffset)

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
	log.Debug("DecodeFromBytes: decoding MStream layer")
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
	switch f.Subtype {
	case MStreamTriggerSubtype:
		trigger, err := DecodeMStreamTrigger(f.Data)
		if err != nil {
			return errors.New("Error while decoding payload of MStream trigger fragment")
		}
		f.MStreamTrigger = trigger
		return nil
	case MStreamDataSubtype:
		data, err := DecodeMStreamData(f.Data)
		if err != nil {
			return errors.New("Error while decoding payload of MStream data fragment")
		}
		f.MStreamData = data
		return nil
	default:
		return errors.New("Unknown fragment subtype")
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
