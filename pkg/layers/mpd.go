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
	"sort"

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
	MpdTimestampMagic = 0x3f60b8a8
)

// MpdLayer ...
type MpdLayer struct {
	layers.BaseLayer
	*MpdTimestampHeader
	*MpdEventHeader
	*MpdDeviceHeader
	Trigger *MStreamTrigger
	Data    map[ChannelNum]*MStreamData
}

var MpdLayerType = gopacket.RegisterLayerType(MpdLayerNum,
	gopacket.LayerTypeMetadata{Name: "MpdLayerType"})

// LayerType returns the type of the Mpd layer in the layer catalog
func (ms *MpdLayer) LayerType() gopacket.LayerType {
	return MpdLayerType
}

// lib-common/MpdRawTypes.h

// MpdTimestampHeader ... // 16 bytes
type MpdTimestampHeader struct {
	Sync      uint32
	Length    uint32
	Timestamp uint64
}

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

// MpdMStreamHeader ... 4 bytes
type MpdMStreamHeader struct {
	Subtype           // 2 bits 0-1
	Length     uint32 // 22 bits 2-23 // payload length in 32-bit words
	ChannelNum        // 8 bits 24-31
}

// Serialize MpdEventHeader
func (h *MpdTimestampHeader) Serialize(buf []byte) error {
	log.Debug("MpdTimestampHeader.Serialize: Sync: %08x", h.Sync)
	log.Debug("MpdTimestampHeader.Serialize: Length: %d", h.Length)
	log.Debug("MpdTimestampHeader.Serialize: Timestamp: %d", h.Timestamp)
	binary.LittleEndian.PutUint32(buf[0:4], h.Sync)
	binary.LittleEndian.PutUint32(buf[4:8], h.Length)
	binary.LittleEndian.PutUint64(buf[8:16], h.Timestamp)
	return nil
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
	log.Debug("MpdMStreamHeader.Serialize: ChannelNum: %d", h.ChannelNum)
	buf[0] = uint8(h.Length<<2) | uint8(h.Subtype&0x3)
	binary.LittleEndian.PutUint16(buf[1:3], uint16(((h.Length<<2)&0xffff00)>>8))
	buf[3] = uint8(h.ChannelNum)
	return nil
}

// SerializeTo serializes the Mpd layer into bytes and writes the bytes to the SerializeBuffer
func (mpd *MpdLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	timestampHeaderBytes, err := b.AppendBytes(16)
	if err != nil {
		return err
	}
	mpd.MpdTimestampHeader.Serialize(timestampHeaderBytes)
	log.Debug("MPD SerializeTo: MpdTimestampHeader:\n%s", hex.Dump(timestampHeaderBytes))

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

	triggerHeaderBytes, err := b.AppendBytes(4)
	if err != nil {
		return err
	}
	triggerHeader := &MpdMStreamHeader{
		Subtype:    MStreamTriggerSubtype,
		Length:     4,
		ChannelNum: 0,
	}
	triggerHeader.Serialize(triggerHeaderBytes)
	log.Debug("MPD SerializeTo: MpdMStreamHeader: trigger:\n%s", hex.Dump(triggerHeaderBytes))

	triggerBytes, err := b.AppendBytes(16)
	if err != nil {
		return err
	}
	mpd.Trigger.Serialize(triggerBytes)
	log.Debug("MPD SerializeTo: trigger:\n%s", hex.Dump(triggerBytes))

	channels := []ChannelNum{}
	for c := range mpd.Data {
		channels = append(channels, c)
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i] < channels[j] })
	for _, c := range channels {
		headerBytes, err := b.AppendBytes(4)
		if err != nil {
			return err
		}
		header := &MpdMStreamHeader{
			Subtype:    MStreamDataSubtype,
			Length:     uint32(len(mpd.Data[c].Bytes) / 4),
			ChannelNum: c,
		}
		header.Serialize(headerBytes)
		log.Debug("MPD SerializeTo: MpdMStreamHeader: data:\n%s", hex.Dump(headerBytes))
		dataBytes, _ := b.AppendBytes(len(mpd.Data[c].Bytes))

		mpd.Data[c].Serialize(dataBytes)
		log.Debug("MPD SerializeTo: data:\n%s", hex.Dump(dataBytes))
	}

	return nil
}
