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

	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
)

const (
	NumEventBuilders      = 10
	BuilderFragmentChSize = 8
)

type EventBuilder struct {
	Id              int
	ManagerId       string
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

	FragmentCh chan *layers.MStreamFragment
	mpdCh      chan<- []byte
	seq        <-chan uint32
}

// NewEvent ...
func NewEventBuilder(id int, managerId string, mpdCh chan<- []byte, seq <-chan uint32) *EventBuilder {
	return &EventBuilder{
		Id:              id,
		ManagerId:       managerId,
		Free:            true,
		DeviceSerial:    0,
		EventNum:        0,
		TriggerChannels: 0,
		DataChannels:    0,
		Trigger:         nil,
		Data:            make(map[layers.ChannelNum]*layers.MStreamData),
		DataSize:        0,
		Length:          0,
		FragmentCh:      make(chan *layers.MStreamFragment, BuilderFragmentChSize),
		mpdCh:           mpdCh,
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
	b.Free = true
	b.EventNum = <-b.seq
	b.TriggerChannels = 0
	b.DataChannels = 0
	b.Trigger = nil
	b.Data = make(map[layers.ChannelNum]*layers.MStreamData)
	b.DataSize = 0
	b.DeviceSerial = 0
	b.Length = 0
}

func (b *EventBuilder) CloseEvent() {
	defer b.Clear()

	if b.Trigger == nil {
		log.Error("Can not close event w/o trigger")
		return
	}
	log.Info("Close event: manager: %s builder: %d event: %d\n"+
		"Data    channels: %064b\n"+
		"Trigger channels: %064b", b.ManagerId, b.Id, b.EventNum, b.DataChannels, b.TriggerChannels)
	dataCount := countDataFragments(b.DataChannels)
	// Total data length is the total length of all data fragments + total length of all MpdMStreamHeader headers
	// data length + (num data fragments + one trigger fragment) * MStream header size
	deviceHeaderLength := b.Length + (dataCount+1)*4
	// + 8 bytes (which is the size of MpdDeviceHeader)
	eventHeaderLength := deviceHeaderLength + 8

	mpd := &layers.MpdLayer{
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
		log.Error("Error while serializing Mpd layer: device: %08x, event: %s", b.DeviceSerial, b.EventNum)
		return
	}

	b.mpdCh <- buf.Bytes()
}

// SetFragment ...
// fragment payload must be decoded before calling this function
func (b *EventBuilder) SetFragment(f *layers.MStreamFragment) {
	if b.Free {
		b.Free = false
		b.EventNum = f.MStreamPayloadHeader.EventNum
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
			b.CloseEvent()
		}
	} else if f.Subtype == layers.MStreamDataSubtype {
		b.DataChannels |= uint64(1) << f.MStreamPayloadHeader.ChannelNum
		b.Data[f.MStreamPayloadHeader.ChannelNum] = f.MStreamData
		if b.Trigger != nil && b.DataChannels == b.TriggerChannels {
			b.CloseEvent()
		}
	}
}

func (b *EventBuilder) Run() {
	b.EventNum = <-b.seq
	log.Info("Run EventBuilder: %s builder: %d event: %d", b.ManagerId, b.Id, b.EventNum)
	for {
		f := <-b.FragmentCh
		if f.MStreamPayloadHeader.EventNum >= b.EventNum+NumEventBuilders {
			log.Info("Force close event: "+
				"%s %d "+
				"builder event: %d "+
				"fragment event: %d",
				b.ManagerId, b.Id, b.EventNum, f.MStreamPayloadHeader.EventNum)
			b.CloseEvent()
			b.EventNum = <-b.seq
			log.Info("Change event number: manager: %s builder: %d event: %d", b.ManagerId, b.Id, b.EventNum)
		}
		if f.MStreamPayloadHeader.EventNum == b.EventNum {
			log.Info("Setting event fragment: %s %d event: %d", b.ManagerId, b.Id, f.MStreamPayloadHeader.EventNum)
			b.SetFragment(f)
		}
	}
}

type EventBuilderManager struct {
	deviceName    string
	eventBuilders []*EventBuilder
	writerCh      chan<- []byte
	fragmentCh    <-chan *layers.MStreamFragment
	seq           chan uint32
	exitCh        <-chan bool
}

func NewEventBuilderManager(
	deviceName string,
	fragmentCh <-chan *layers.MStreamFragment,
	writerCh chan<- []byte,
	exitCh <-chan bool) *EventBuilderManager {
	log.Info("Creating EventBuilderManager: %s", deviceName)
	return &EventBuilderManager{
		deviceName: deviceName,
		writerCh:   writerCh,
		fragmentCh: fragmentCh,
		exitCh:     exitCh,
	}
}

func (m *EventBuilderManager) Run() {
	m.seq = make(chan uint32)
	go func() {
		eventSeq := uint32(1)
		for {
			m.seq <- eventSeq
			eventSeq++
		}
	}()

	m.eventBuilders = []*EventBuilder{}
	for i := 0; i < NumEventBuilders; i++ {
		log.Info("Creating EventBuilder: %s %d", m.deviceName, i)
		b := NewEventBuilder(i, m.deviceName, m.writerCh, m.seq)
		m.eventBuilders = append(m.eventBuilders, b)
		go func() {
			b.Run()
		}()
	}

	for {
		f := <-m.fragmentCh
		for _, b := range m.eventBuilders {
			b.FragmentCh <- f
		}
	}
}
