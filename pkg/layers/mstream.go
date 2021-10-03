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
	// MStreamLayerNum identifies the layer
	MStreamLayerNum = 1998
	MStreamTriggerSubtype uint8 = 0
	MStreamDataSubtype uint8 = 1
)

// MStreamTrigger ...
type MStreamTrigger struct {
	DeviceSerial uint32
	EventNum uint32 // 24 bits
	TaiSec uint32
	Flags uint8 // 2 bits
	TaiNSec uint32 // 30 bits
	LowCh uint32
	HiCh uint32
}

// MStreamData ...
type MStreamData struct {
	DeviceSerial uint32
	EventNum uint32 // 24 bits
	ChannelNum uint8
	Data []byte
}

// MStreamFragment ...
type MStreamFragment struct {
	FragmentLength uint16 // length of fragment payload NOT including MStream header in bytes
	Subtype        uint8 // 2 bits
	Flags          uint8 // 6 bits
	// 0xd9 for ADC64VE-XGE
	// 0xdf for ADC64VE-V3-XG
	DeviceID       uint8
	FragmentID     uint16
	FragmentOffset uint16
	Data []byte
	// Fragment contains either MStreamTrigger or MStreamData, not both of them at the same time
	//*MStreamTrigger
	//*MStreamData
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


// SerializeMStreamData ...
func (mst *MStreamTrigger) Serialize(buf []byte) error {
	binary.LittleEndian.PutUint32(buf[0:4], mst.DeviceSerial)
	buf[4] = uint8(mst.EventNum & 0xff)
	binary.LittleEndian.PutUint16(buf[5:7], uint16((mst.EventNum & 0xffff00) >> 8))
	binary.LittleEndian.PutUint32(buf[8:12], mst.TaiSec)
	binary.LittleEndian.PutUint32(buf[12:16], (mst.TaiNSec << 2 | uint32(mst.Flags)))
	binary.LittleEndian.PutUint32(buf[16:20], mst.LowCh)
	binary.LittleEndian.PutUint32(buf[20:24], mst.HiCh)
	return nil
}

// SerializeMStreamData ...
func (msd *MStreamData) Serialize(buf []byte) error {
	binary.LittleEndian.PutUint32(buf[0:4], msd.DeviceSerial)
	buf[4] = uint8(msd.EventNum & 0xff)
	binary.LittleEndian.PutUint16(buf[5:7], uint16((msd.EventNum & 0xffff00) >> 8))
	buf[7] = msd.ChannelNum
	copy(buf[8:], msd.Data)
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
		headerBytes[2] = (fragment.Flags << 2) | fragment.Subtype
		headerBytes[3] = fragment.DeviceID
		binary.LittleEndian.PutUint32(headerBytes[4:8], (uint32(fragment.FragmentID) << 16) | uint32(fragment.FragmentOffset))

		payloadBytes, err := b.AppendBytes(int(fragment.FragmentLength))
		if err != nil {
			return err
		}
		copy(payloadBytes, fragment.Data)
	}

	return nil
}

// DecodeMStreamData ...
func DecodeMStreamData(fragmentPayload []byte) (*MStreamData, error) {
	if len(fragmentPayload) < 8 {
		return nil, errors.New("MStream data packet too short. Must at least have data header.")
	}
	return &MStreamData{
		DeviceSerial: binary.LittleEndian.Uint32(fragmentPayload[0:4]),
		EventNum: binary.LittleEndian.Uint32(fragmentPayload[4:7]),
		ChannelNum: fragmentPayload[7],
		Data: fragmentPayload[8:],
	}, nil
}

// DecodeMStreamTrigger ...
func DecodeMStreamTrigger(fragmentPayload []byte) (*MStreamTrigger, error) {
	if len(fragmentPayload) < 24 {
		return nil, errors.New("MStream trigger packet too short. Must be at least 24 bytes.")
	}

	taiNSecFlags := binary.LittleEndian.Uint32(fragmentPayload[12:16])

	return &MStreamTrigger{
		DeviceSerial: binary.LittleEndian.Uint32(fragmentPayload[0:4]),
		EventNum: binary.LittleEndian.Uint32(fragmentPayload[4:7]),
		TaiSec: binary.LittleEndian.Uint32(fragmentPayload[8:12]),
		Flags: uint8(taiNSecFlags & 0x3),
		TaiNSec: taiNSecFlags >> 2,
		LowCh: binary.LittleEndian.Uint32(fragmentPayload[16:20]),
		HiCh: binary.LittleEndian.Uint32(fragmentPayload[20:24]),
	}, nil
}

// DecodeFragment ...
// offset is the beginning of MStream fragment inside MStream packet
// data is the whole MStream packet
func (ms *MStreamLayer) DecodeFragment(offset int, data []byte) (int, error) {
	// Decoding fragment header
	fragmentLength := binary.LittleEndian.Uint16(data[offset:offset + 2])
	if fragmentLength == 0 {
		return offset, errors.New("Invalid MStream fragment: FragmentLength = 0")
	}
	// end of fragment is current offset + size of fragment header + fragment length
	newOffset := offset + 8 + int(fragmentLength)

	subtype := data[offset + 2] & 0x3 // Subtype is two least significant bits
	flags := (data[offset + 2] >> 2) & 0x3f // Flags is six high bits
	deviceID := data[offset + 3]
	fragmentOffsetID := binary.LittleEndian.Uint32(data[offset + 4:offset + 8])
	fragmentID := uint16(fragmentOffsetID >> 16) // FragmentID takes 2 bytes for MStream 2.x
	fragmentOffset := uint16(fragmentOffsetID & 0xffff) // FragmentOffset takes 2 bytes for MStream 2.x

	fragment := &MStreamFragment{
		FragmentLength: fragmentLength,
		Subtype: subtype,
		Flags: flags,
		DeviceID: deviceID,
		FragmentID: fragmentID,
		FragmentOffset: fragmentOffset,
		Data: data[offset + 8:newOffset],
	}

	//// Decoding fragment payload which is one of MStreamTrigger or MStreamData
	//// We call fragment payload decoders not with the whole MStream packet but with
	//// the actual payload of a fragment excluding fragment header
	//mstreamPayloadDecoders := map[uint8]func([]byte, *MStreamFragment) error{
	//	MStreamTriggerSubtype: DecodeMStreamTrigger,
	//	MStreamDataSubtype: DecodeMStreamData,
	//}
	//
	//err := mstreamPayloadDecoders[subtype](data[offset + 8:newOffset], fragment)
	//if err != nil {
	//	return newOffset, err
	//}

	ms.Fragments = append(ms.Fragments, fragment)

	return newOffset, nil
}

func (ms *MStreamLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	// At least one fragment must be in the packet and fragment header length is 8
	if len(data) < 8 {
		df.SetTruncated()
		// TODO return custom error
		return errors.New("MStream packet too short")
	}

	// MStream layer consists of fragments without common layer header
	ms.BaseLayer = layers.BaseLayer{
		Contents: []byte{},
		Payload: data,
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

func DecodeMStreamLayer(data []byte, p gopacket.PacketBuilder) error {
	ms := &MStreamLayer{}
	err := ms.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(ms)
	return nil
}
