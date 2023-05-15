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
	"github.com/google/gopacket"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
)

const (
	NumEventBuildersPerManager = 2
)

type EventBuilder struct {
	id              int
	cfg             *config.Config
	device          *config.Device
	Free            bool
	DeviceSerial    uint32
	EventNum        uint32
	TriggerChannels uint64
	DataChannels    uint64
	DataSize        uint32
	DeviceID        uint8

	Trigger *layers.MStreamTrigger
	Data    map[layers.ChannelNum]*layers.MStreamData
	Length  uint32

	DefragmentedCh chan *layers.MStreamFragment
	writerCh       chan<- []byte
	seq            <-chan uint32
}

// NewEvent ...
func NewEventBuilder(id int, cfg *config.Config, device *config.Device, writerCh chan<- []byte, seq <-chan uint32) *EventBuilder {
	return &EventBuilder{
		id:              id,
		cfg:             cfg,
		device:          device,
		Free:            true,
		DeviceSerial:    0,
		EventNum:        0,
		TriggerChannels: 0,
		DataChannels:    0,
		Trigger:         nil,
		Data:            make(map[layers.ChannelNum]*layers.MStreamData),
		DataSize:        0,
		Length:          0,
		DefragmentedCh:  make(chan *layers.MStreamFragment),
		writerCh:        writerCh,
		seq:             seq,
	}
}

func countDataFragments(channels uint64) (count uint32) {
	// we use here Brian Kernighan’s algorithm
	for channels > 0 {
		channels &= channels - 1
		count += 1
	}
	return
}

func (b *EventBuilder) Clear() {
	//log.Info("Clear event builder: device: %s event: %d", b.deviceName, b.EventNum)
	b.Free = true
	//b.EventNum = 0
	b.TriggerChannels = 0
	b.DataChannels = 0
	b.Trigger = nil
	b.Data = make(map[layers.ChannelNum]*layers.MStreamData)
	b.DataSize = 0
	b.DeviceSerial = 0
	b.Length = 0
}

func (b *EventBuilder) CloseEvent(persist bool) {
	defer b.Clear()

	if b.Trigger == nil {
		log.Error("Can not close event w/o trigger: %s event: %d", b.device.Name, b.EventNum)
		return
	}
	//log.Info("Close event: %s event: %d\n"+
	//	"Data    channels: %064b\n"+
	//	"Trigger channels: %064b", b.deviceName, b.EventNum, b.DataChannels, b.TriggerChannels)

	if !persist {
		return
	}

	dataCount := countDataFragments(b.DataChannels)
	// Total data length is the total length of all data fragments + total length of all MpdMStreamHeader headers
	// data length + (num data fragments + one trigger fragment) * MStream header size
	deviceHeaderLength := b.Length + (dataCount+1)*4
	// + 8 bytes MpdDeviceHeader
	eventHeaderLength := deviceHeaderLength + 8
	// + 12 bytes MpdEventHeader
	// + 16 bytes MpdTimestampHeader
	// + 16 bytes MpdInventoryHeader
	inventoryHeaderLength := eventHeaderLength + 12 + 16 + 16
	if inventoryHeaderLength%64 != 0 {
		panic("Inventory header error: Data length is not multiple of 64")
	}
	if inventoryHeaderLength/64 > 0xffff {
		panic("Inventory header error: Data length is more than 2^16 * 64")
	}

	var mpdInventoryHeader *layers.MpdInventoryHeader = nil
	if b.cfg.Inventory != nil && b.device.DeviceInventory != nil {
		mpdInventoryHeader = &layers.MpdInventoryHeader{
			Version:    b.cfg.Inventory.Version,
			DetectorID: b.cfg.Inventory.DetectorID,
			CrateID:    b.device.DeviceInventory.CrateID,
			SlotID:     b.device.DeviceInventory.SlotID,
			StreamID:   0,
			Reserved:   0,
			// sequenceID takes 12 bits in the inventory header
			SequenceID: uint16(b.EventNum % 0xfff),
			Length:     uint16(inventoryHeaderLength / 64),
			Timestamp:  uint64(b.Trigger.TaiSec<<30) | uint64(b.Trigger.TaiNSec&0x3fffffff),
		}
	}

	mpd := &layers.MpdLayer{
		MpdInventoryHeader: mpdInventoryHeader,
		MpdTimestampHeader: &layers.MpdTimestampHeader{
			Sync:      layers.MpdTimestampMagic,
			Length:    8,
			Timestamp: srv.Now(),
		},
		MpdEventHeader: &layers.MpdEventHeader{
			Sync:     layers.MpdSyncMagic,
			EventNum: b.EventNum,
			Length:   eventHeaderLength,
		},
		MpdDeviceHeader: &layers.MpdDeviceHeader{
			DeviceSerial: b.DeviceSerial,
			DeviceID:     b.DeviceID,
			Length:       deviceHeaderLength,
		},
		Trigger: b.Trigger,
		Data:    b.Data,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, mpd)
	if err != nil {
		log.Error("Error while serializing Mpd layer: %s, event: %d", b.device.Name, b.EventNum)
		return
	}

	b.writerCh <- buf.Bytes()
}

// SetFragment ...
// fragment payload must be decoded before calling this function
func (b *EventBuilder) SetFragment(f *layers.MStreamFragment) {
	if b.Free {
		b.Free = false
		//b.EventNum = f.MStreamPayloadHeader.EventNum
		b.DeviceSerial = f.MStreamPayloadHeader.DeviceSerial
	}

	// We substruct 8 bytes from the fragment length because fragment payload has
	// its own header MStreamPayloadHeader which is not included when we serialize
	// trigger and data when writing to MPD file.
	b.Length += uint32(f.FragmentLength - 8)

	if f.Subtype == layers.MStreamTriggerSubtype {
		b.DeviceID = f.DeviceID
		b.TriggerChannels = uint64(f.MStreamTrigger.HiCh)<<32 | uint64(f.MStreamTrigger.LowCh)
		b.Trigger = f.MStreamTrigger
		if b.DataChannels == b.TriggerChannels {
			b.CloseEvent(true)
		}
	} else if f.Subtype == layers.MStreamDataSubtype {
		b.DataChannels |= uint64(1) << f.MStreamPayloadHeader.ChannelNum
		b.Data[f.MStreamPayloadHeader.ChannelNum] = f.MStreamData
		if b.Trigger != nil && b.DataChannels == b.TriggerChannels {
			b.CloseEvent(true)
		}
	}
}

func (b *EventBuilder) Run() {
	b.EventNum = <-b.seq
	log.Info("Run EventBuilder: %s id: %d", b.device.Name, b.id)
	for {
		f := <-b.DefragmentedCh
		if f.MStreamPayloadHeader.EventNum >= b.EventNum+NumEventBuildersPerManager {
			if !b.Free {
				//log.Info("Force close event: %s id: %d builder event: %d fragment event: %d",
				//	b.deviceName, b.id, b.EventNum, f.MStreamPayloadHeader.EventNum)
				b.CloseEvent(false)
			}
			b.EventNum = <-b.seq
		}
		if f.MStreamPayloadHeader.EventNum == b.EventNum {
			//log.Info("Handle event fragment: %s id: %d event: %d fragment: %04x",
			//	b.deviceName, b.id, f.MStreamPayloadHeader.EventNum, f.FragmentID)
			b.SetFragment(f)
		}
	}
}

type EventBuilderManager struct {
	cfg            *config.Config
	device         *config.Device
	eventBuilders  []*EventBuilder
	writerCh       chan<- []byte
	defragmentedCh <-chan *layers.MStreamFragment
	seq            chan uint32
}

func NewEventBuilderManager(cfg *config.Config, device *config.Device, defragmentedCh <-chan *layers.MStreamFragment, writerCh chan<- []byte) *EventBuilderManager {
	//log.Info("Creating EventBuilderManager: %s", deviceName)
	return &EventBuilderManager{
		cfg:            cfg,
		device:         device,
		writerCh:       writerCh,
		defragmentedCh: defragmentedCh,
	}
}

func (m *EventBuilderManager) Run() {
	log.Info("Run EventBuilderManger: %s", m.device.Name)
	m.seq = make(chan uint32)

	go func(seq chan uint32) {
		eventSeq := uint32(1)
		for {
			seq <- eventSeq
			eventSeq++
		}
	}(m.seq)

	m.eventBuilders = []*EventBuilder{}
	for i := 0; i < NumEventBuildersPerManager; i++ {
		//log.Info("Creating EventBuilder: %s id: %d", m.deviceName, i)
		b := NewEventBuilder(i, m.cfg, m.device, m.writerCh, m.seq)
		m.eventBuilders = append(m.eventBuilders, b)
		go func(eventBuilder *EventBuilder) {
			eventBuilder.Run()
		}(b)
	}

	for {
		f := <-m.defragmentedCh
		//log.Info("Handling event fragment: device %s event: %d", m.deviceName, f.MStreamPayloadHeader.EventNum)
		for _, b := range m.eventBuilders {
			b.DefragmentedCh <- f
		}
	}

}
