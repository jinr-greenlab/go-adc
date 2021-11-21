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

package srv

import (
	"bufio"
	"fmt"
	"github.com/google/gopacket"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"os"
)


// Multiple EventBuilders are needed (one per device)
// OR it must be able to distinguish events from different devices

type EventBuilder struct {
	DeviceSerial uint32
	EventNum uint32
	Free bool
	TriggerChannels uint64
	DataChannels uint64
	DataSize uint32
	DeviceID uint8

	Trigger *layers.MStreamTrigger
	Data map[layers.ChannelNum]layers.MStreamData
	Length uint32

	writer *bufio.Writer
	file *os.File
}

func NewEventBuilder(deviceSerial uint32, fileSuffix string) (*EventBuilder, error) {

	filename := fmt.Sprintf("%04x_%s.data", deviceSerial, fileSuffix)
	file, err := os.Create(filename)
	if err != nil {
		log.Error("Error while creating file: %s", filename)
		return nil, err
	}
	writer := bufio.NewWriter(file)

	return &EventBuilder{
		Free: true,
		Data: make(map[layers.ChannelNum]layers.MStreamData),
		file: file,
		writer: writer,
	}, nil
}

func (eb *EventBuilder) Close() {
	eb.writer.Flush()
	eb.file.Close()
}

func (eb *EventBuilder) Clear() {
	eb.Free = true
	eb.EventNum = 0
	eb.TriggerChannels = 0
	eb.DataChannels = 0
	eb.Trigger = nil
	eb.Data = make(map[layers.ChannelNum]layers.MStreamData)
	eb.DataSize = 0
	eb.DeviceSerial = 0
	eb.Length = 0
}

func countDataFragments(channels uint64) (count uint32){
	// we use here Brian Kernighan’s algorithm
	for channels > 0 {
		channels &= channels - 1
		count += 1
	}
	return
}

func (eb *EventBuilder) CloseEvent() error {
	defer eb.Clear()

	if eb.Trigger == nil {
		log.Error("Can not close event w/o trigger frame")
		return nil
	}

	// TODO: Add logic to check if there are missing events
	// EventNum - ExpectedEventNum ... (see MStreamDump::closeAdc())

	dataCount := countDataFragments(eb.DataChannels)
	// Total data length is the total length of all data fragments + total length of all MpdMStreamHeader headers
	// data length + (num data fragments + one trigger fragment) * MStream header size
	deviceHeaderLength := eb.Length + (dataCount + 1) * 4
	// + 8 bytes (which is the size of MpdDeviceHeader)
	eventHeaderLength := deviceHeaderLength + 8

	mpd := &layers.MpdLayer{
		MpdTimestampHeader: &layers.MpdTimestampHeader{
			Sync: layers.MpdTimestampMagic,
			Length: 8,
			Timestamp: Now(),
		},
		MpdEventHeader: &layers.MpdEventHeader{
			Sync: layers.MpdSyncMagic,
			EventNum: eb.EventNum,
			Length: eventHeaderLength,
		},
		MpdDeviceHeader: &layers.MpdDeviceHeader{
			DeviceSerial: eb.DeviceSerial,
			DeviceID: eb.DeviceID,
			Length: eb.Length + (dataCount + 1) * 4,
		},
		Trigger: eb.Trigger,
		Data: eb.Data,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, mpd)
	if err != nil {
		log.Error("Error while serializing Mpd layer")
		return err
	}

	_, err = eb.writer.Write(buf.Bytes())
	if err != nil {
		log.Error("Error while closing event: device: %04x event: %d", eb.DeviceSerial, eb.EventNum)
		return err
	}
	return nil
}

// SetFragment ...
// fragment payload must be decoded before calling this function
func (eb *EventBuilder) SetFragment(f *layers.MStreamFragment) {
	if !eb.Free && f.MStreamPayloadHeader.EventNum != eb.EventNum {
		// Close current event if a new event comes in even if we don't
		// have all necessary fragments
		eb.CloseEvent()
	}

	if eb.Free {
		eb.Free = false
		eb.EventNum = f.MStreamPayloadHeader.EventNum
		eb.DeviceSerial = f.MStreamPayloadHeader.DeviceSerial
		eb.Length += uint32(f.FragmentLength)
	}

	if f.Subtype == layers.MStreamTriggerSubtype {
		eb.DeviceID = f.DeviceID
		eb.TriggerChannels = uint64(f.MStreamTrigger.HiCh) << 32 | uint64(f.MStreamTrigger.LowCh)
		eb.Trigger = f.MStreamTrigger
		if eb.DataChannels == eb.TriggerChannels {
			eb.CloseEvent()
		}
	} else if f.Subtype == layers.MStreamDataSubtype {
		eb.DataChannels |= uint64(1) << f.MStreamPayloadHeader.ChannelNum
		eb.Data[f.MStreamPayloadHeader.ChannelNum] = f.MStreamData
		if eb.Trigger != nil && eb.DataChannels == eb.TriggerChannels {
			eb.CloseEvent()
		}
	}
}

type EventHandler struct {
	eventBuilders map[uint32]*EventBuilder
	fileSuffix string
}

func NewEventHandler(fileSuffix string) *EventHandler {
	return &EventHandler{
		eventBuilders: make(map[uint32]*EventBuilder),
		fileSuffix: fileSuffix,
	}
}

func (eh *EventHandler) Close() {
	for _, eb := range eh.eventBuilders {
		eb.Close()
	}
}

func (eh *EventHandler) SetFragment(f *layers.MStreamFragment) {
	deviceSerial := f.MStreamPayloadHeader.DeviceSerial
	eventBuilder, ok := eh.eventBuilders[deviceSerial]
	if !ok {
		eventBuilder, err := NewEventBuilder(deviceSerial, eh.fileSuffix)
		if err != nil {
			log.Error("Error while creating event builder: device: %04x", deviceSerial)
		}
		eh.eventBuilders[deviceSerial] = eventBuilder
	}
	eventBuilder.SetFragment(f)
}
