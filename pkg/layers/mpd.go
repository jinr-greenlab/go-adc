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
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	// MpdLayerNum identifies the layer
	MpdLayerNum = 1995
)

const (
	MpdSyncMagic      = 0x2A502A50
	MpdStartRunMagic  = 0x72617453
	MpdStopRunMagic   = 0x706F7453
	MpdRunNumberMagic = 0x236E7552
	MpdRunIndexMagic  = 0x78646E49
)

// MpdLayer ...
type MpdLayer struct {
	layers.BaseLayer
	*MpdEventHeader
	*MpdDeviceHeader
	Data *MStreamData
}

var MpdLayerType = gopacket.RegisterLayerType(MpdLayerNum,
	gopacket.LayerTypeMetadata{Name: "MpdLayerType"})

// LayerType returns the type of the Mpd layer in the layer catalog
func (ms *MpdLayer) LayerType() gopacket.LayerType {
	return MpdLayerType
}

// lib-common/MpdRawTypes.h

// MpdEventHeader ... 12 bytes
type MpdEventHeader struct {
	Sync     uint32
	EventNum uint32
	Length   uint32 // total length in bytes of all device event blocks
}

// MpdDeviceHeader ... 8 bytes
type MpdDeviceHeader struct {
	DeviceSerial uint32
	DeviceID     uint8
	Length       uint32 // 24 bits // total length in bytes of all mstream blocks
}

// MpdMStreamHeader ... 4 bytes (1 byte is unused)
type MpdMStreamHeader struct {
	Subtype        // 2 bits 0-1
	Length  uint32 // 22 bits 2-23 // payload length in 32-bit words
}

// Serialize MpdEventHeader
func (h *MpdEventHeader) Serialize(buf []byte) error {
	log.Debug("MpdEventHeader.Serialize: Sync: %d", h.Sync)
	log.Debug("MpdEventHeader.Serialize: EventNum: %d", h.EventNum)
	log.Debug("MpdEventHeader.Serialize: Length: %d", h.Length)
	binary.LittleEndian.PutUint32(buf[0:4], h.Sync)
	binary.LittleEndian.PutUint32(buf[4:8], h.Length)
	binary.LittleEndian.PutUint32(buf[8:12], h.EventNum)
	return nil
}

// Serialize MpdDeviceHeader
func (h *MpdDeviceHeader) Serialize(buf []byte) error {
	log.Debug("MpdDeviceHeader.Serialize: DeviceSerial: %08x", h.DeviceSerial)
	log.Debug("MpdDeviceHeader.Serialize: DeviceID: %d", h.DeviceID)
	log.Debug("MpdDeviceHeader.Serialize: Length: %d", h.Length)
	binary.LittleEndian.PutUint32(buf[0:4], h.DeviceSerial)
	binary.LittleEndian.PutUint16(buf[4:6], uint16(h.Length&0xffff))
	buf[6] = uint8((h.Length & 0xff0000) >> 4)
	buf[7] = h.DeviceID
	return nil
}

// Serialize MpdMStreamHeader
func (h *MpdMStreamHeader) Serialize(buf []byte) error {
	log.Debug("MpdMStreamHeader.Serialize: Subtype: %d", h.Subtype)
	log.Debug("MpdMStreamHeader.Serialize: Length: %d", h.Length)
	buf[0] = uint8(h.Length<<2) | uint8(h.Subtype&0x3)
	binary.LittleEndian.PutUint16(buf[1:3], uint16(((h.Length<<2)&0xffff00)>>8))
	buf[3] = uint8(0) // in ADC64 this is the channel number, in TQDC it is not used
	return nil
}

// SerializeTo serializes the Mpd layer into bytes and writes the bytes to the SerializeBuffer
func (mpd *MpdLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	eventHeaderBytes, err := b.AppendBytes(12)
	if err != nil {
		return err
	}
	mpd.MpdEventHeader.Serialize(eventHeaderBytes)
	log.Debug("MPD SerializeTo: MpdEventHeader:\n%s", hex.Dump(eventHeaderBytes))

	deviceHeaderBytes, err := b.AppendBytes(8)
	if err != nil {
		return err
	}
	mpd.MpdDeviceHeader.Serialize(deviceHeaderBytes)
	log.Debug("MPD SerializeTo: MpdDeviceHeader:\n%s", hex.Dump(deviceHeaderBytes))

	headerBytes, err := b.AppendBytes(4)
	if err != nil {
		return err
	}
	header := &MpdMStreamHeader{
		Subtype: MStreamDataSubtype,
		Length:  uint32(len(mpd.Data.Bytes) / 4),
	}
	header.Serialize(headerBytes)
	log.Debug("MPD SerializeTo: MpdMStreamHeader: data:\n%s", hex.Dump(headerBytes))
	dataBytes, _ := b.AppendBytes(len(mpd.Data.Bytes))

	mpd.Data.Serialize(dataBytes)
	log.Debug("MPD SerializeTo: data:\n%s", hex.Dump(dataBytes))

	return nil
}
