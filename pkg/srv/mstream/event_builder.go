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
	"bufio"
	"fmt"
	"io"
	"path"
	"sync"
	"github.com/google/gopacket"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
	"os"
	"time"
)

type Writer struct {
	*bufio.Writer
	file *os.File
	discard bool
}

func NewWriter() *Writer {
	return &Writer{
		discard: true,
	}
}

func (w *Writer) Write(buf []byte) (int, error) {
	if w.discard {
		return io.Discard.Write(buf)
	}
	return w.Writer.Write(buf)
}

func (w *Writer) Flush() {
	discard := w.discard
	w.discard = true
	if !discard {
		w.Flush()
		w.file.Close()
	}
}

func (w *Writer) Persist(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		log.Error("Error while creating file: %s", filename)
		return err
	}
	w.Writer = bufio.NewWriter(file)
	w.discard = false
	return nil
}

// This is trivial implementation of the event builder which assumes that
// all fragments come from the same ADC board. This means we need to create
// multiple event builders, one per ADC board.
type EventBuilder struct {
	DeviceSerial uint32
	EventNum uint32
	Free bool
	TriggerChannels uint64
	DataChannels uint64
	DataSize uint32
	DeviceID uint8

	Trigger *layers.MStreamTrigger
	Data map[layers.ChannelNum]*layers.MStreamData
	Length uint32

	writer *Writer
}

// NewEventBuilder ...
func NewEventBuilder(writer *Writer) (*EventBuilder, error) {
	builder := &EventBuilder{
		writer: writer,
	}
	builder.Clear()
	return builder, nil
}

func (eb *EventBuilder) Clear() {
	eb.Free = true
	eb.EventNum = 0
	eb.TriggerChannels = 0
	eb.DataChannels = 0
	eb.Trigger = nil
	eb.Data = make(map[layers.ChannelNum]*layers.MStreamData)
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
			Timestamp: srv.Now(),
		},
		MpdEventHeader: &layers.MpdEventHeader{
			Sync: layers.MpdSyncMagic,
			EventNum: eb.EventNum,
			Length: eventHeaderLength,
		},
		MpdDeviceHeader: &layers.MpdDeviceHeader{
			DeviceSerial: eb.DeviceSerial,
			DeviceID: eb.DeviceID,
			Length: deviceHeaderLength,
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
	}

	// We substruct 8 bytes from the fragment length because fragment payload has
	// its own header MStreamPayloadHeader which is not included when we serialize
	// trigger and data when writing to MPD file.
	eb.Length += uint32(f.FragmentLength - 8)

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
	sync.RWMutex
	eventBuilders map[uint32]*EventBuilder
	persistTimestamp string
	persistDir string
	persistFilePrefix string
	persist bool
}

func NewEventHandler() *EventHandler {
	return &EventHandler{
		eventBuilders: make(map[uint32]*EventBuilder),
		persist: false,
	}
}

func (eh *EventHandler) Flush() {
	log.Info("Flush data files")
	eh.persist = false
	eh.Lock()
	defer eh.Unlock()
	for _, eb := range eh.eventBuilders {
		eb.writer.Flush()
		eb.Clear()
	}
}

func (eh *EventHandler) persistFilename(deviceSerial uint32) string {
	filename := fmt.Sprintf("%08x_%s.data", deviceSerial, eh.persistTimestamp)
	if eh.persistFilePrefix != "" {
		filename = fmt.Sprintf("%s_%s", eh.persistFilePrefix, filename)
	}
	return path.Join(eh.persistDir, filename)
}

func (eh *EventHandler) Persist(dir, filePrefix string) error {
	eh.persist = true
	eh.persistTimestamp = time.Now().UTC().Format("20060102_150405")
	eh.persistDir = dir
	eh.persistFilePrefix = filePrefix
	log.Info("Persist data to files: timestamp: %s dir: %s prefix: %s", eh.persistTimestamp, dir, filePrefix)
	eh.Lock()
	defer eh.Unlock()
	for deviceSerial, eb := range eh.eventBuilders {
		filename := eh.persistFilename(deviceSerial)
		log.Info("Persist: device: %08x file: %s", deviceSerial, filename)
		err := eb.writer.Persist(filename)
		if err != nil {
			return err
		}
	}
	return nil
}

func (eh *EventHandler) SetFragment(f *layers.MStreamFragment) error {
	eh.Lock()
	defer eh.Unlock()
	deviceSerial := f.MStreamPayloadHeader.DeviceSerial
	_, ok := eh.eventBuilders[deviceSerial]
	if !ok {
		writer := NewWriter()
		if eh.persist {
			filename := eh.persistFilename(deviceSerial)
			log.Info("Persist: device: %08x file: %s", deviceSerial, filename)
			writer.Persist(filename)
		}
		eventBuilder, err := NewEventBuilder(writer)
		if err != nil {
			log.Error("Error while creating event builder: device: %04x", deviceSerial)
			return err
		}
		eh.eventBuilders[deviceSerial] = eventBuilder
	}
	eh.eventBuilders[deviceSerial].SetFragment(f)
	return nil
}
