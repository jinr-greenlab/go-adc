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
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"io"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
	"os"
	"path"
	"sync"
	"time"
)

const (
	MaxEventDiff = 10
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

type Event struct {
	DeviceSerial uint32
	EventNum uint32
	TriggerChannels uint64
	DataChannels uint64
	DataSize uint32
	DeviceID uint8

	Trigger *layers.MStreamTrigger
	Data map[layers.ChannelNum]*layers.MStreamData
	Length uint32

	writer *Writer
}

// NewEvent ...
func NewEvent(deviceSerial, eventNum uint32) *Event {
	return &Event{
		DeviceSerial: deviceSerial,
		EventNum: eventNum,
		TriggerChannels: 0,
		DataChannels: 0,
		Trigger: nil,
		Data: make(map[layers.ChannelNum]*layers.MStreamData),
		DataSize: 0,
		Length: 0,
	}

}

func countDataFragments(channels uint64) (count uint32){
	// we use here Brian Kernighan’s algorithm
	for channels > 0 {
		channels &= channels - 1
		count += 1
	}
	return
}

func (e *Event) Close(writer io.Writer) error {
	if e.Trigger == nil {
		return errors.New("Can not close event w/o trigger")
	}

	log.Info("Data    channels: %064b", e.DataChannels)
	log.Info("Trigger channels: %064b", e.TriggerChannels)
	dataCount := countDataFragments(e.DataChannels)
	// Total data length is the total length of all data fragments + total length of all MpdMStreamHeader headers
	// data length + (num data fragments + one trigger fragment) * MStream header size
	deviceHeaderLength := e.Length + (dataCount + 1) * 4
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
			EventNum: e.EventNum,
			Length: eventHeaderLength,
		},
		MpdDeviceHeader: &layers.MpdDeviceHeader{
			DeviceSerial: e.DeviceSerial,
			DeviceID: e.DeviceID,
			Length: deviceHeaderLength,
		},
		Trigger: e.Trigger,
		Data: e.Data,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, mpd)
	if err != nil {
		log.Error("Error while serializing Mpd layer")
		return err
	}

	_, err = writer.Write(buf.Bytes())
	if err != nil {
		log.Error("Error while writing event: device: %04x event: %d", e.DeviceSerial, e.EventNum)
		return err
	}
	return nil
}

// SetFragment ...
// fragment payload must be decoded before calling this function
func (e *Event) SetFragment(f *layers.MStreamFragment) (bool, error) {
	if f.MStreamPayloadHeader.DeviceSerial != e.DeviceSerial {
		return false, errors.New(fmt.Sprintf("Wrong device serial number. Must be: %08x", e.DeviceSerial))
	}

	if f.Subtype == layers.MStreamTriggerSubtype {
		if f.MStreamPayloadHeader.EventNum != e.EventNum {
			return false, errors.New(fmt.Sprintf("Wrong event number. Must be: %d", e.EventNum))
		}
	}
	// We substruct 8 bytes from the fragment length because fragment payload has
	// its own header MStreamPayloadHeader which is not included when we serialize
	// trigger and data when writing to MPD file.
	e.Length += uint32(f.FragmentLength - 8)

	if f.Subtype == layers.MStreamTriggerSubtype {
		e.DeviceID = f.DeviceID
		e.TriggerChannels = uint64(f.MStreamTrigger.HiCh) << 32 | uint64(f.MStreamTrigger.LowCh)
		e.Trigger = f.MStreamTrigger
		if e.DataChannels == e.TriggerChannels {
			return true, nil
		}
	} else if f.Subtype == layers.MStreamDataSubtype {
		e.DataChannels |= uint64(1) << f.MStreamPayloadHeader.ChannelNum
		e.Data[f.MStreamPayloadHeader.ChannelNum] = f.MStreamData
		if e.Trigger != nil && e.DataChannels == e.TriggerChannels {
			return true, nil
		}
	}
	return false, nil
}


type eventKey struct {
	DeviceSerial uint32
	EventNum uint32
}

func NewEventKey(deviceSerial, eventNum uint32) eventKey {
	return eventKey{
		DeviceSerial: deviceSerial,
		EventNum: eventNum,
	}
}


type EventHandler struct {
	sync.RWMutex
	writers map[uint32]*Writer
	events map[eventKey]*Event
	persist bool
	persistDir string
	persistFilePrefix string
	persistTimestamp string

	maxClosedEvent uint32
	//nextCloseLog uint32
	nextFlushOld uint32
}

func NewEventHandler() *EventHandler {
	return &EventHandler{
		writers: make(map[uint32]*Writer),
		events: make(map[eventKey]*Event),
		persist: false,
		maxClosedEvent: 0,
		nextFlushOld: MaxEventDiff,
	}
}

func (eh *EventHandler) Flush() {
	log.Info("Flush data files")
	eh.persist = false
	eh.Lock()
	defer eh.Unlock()
	for _, w := range eh.writers {
		w.Flush()
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
	eh.Lock()
	defer eh.Unlock()
	eh.persist = true
	eh.persistDir = dir
	eh.persistFilePrefix = filePrefix
	eh.persistTimestamp = time.Now().UTC().Format("20060102_150405")
	log.Info("Persist data to files: dir: %s prefix: %s timestamp: %s", dir, filePrefix, eh.persistTimestamp)
	for deviceSerial, w := range eh.writers {
		filename := eh.persistFilename(deviceSerial)
		log.Info("Persist writer: device: %08x file: %s", deviceSerial, filename)
		err := w.Persist(filename)
		if err != nil {
			return err
		}
	}
	return nil
}

func (eh *EventHandler) SetFragment(f *layers.MStreamFragment) error {
	eh.Lock()
	defer eh.Unlock()
	eventKey := NewEventKey(f.MStreamPayloadHeader.DeviceSerial, f.MStreamPayloadHeader.EventNum)

	// Add writer for a device if it does not exist
	_, ok := eh.writers[eventKey.DeviceSerial]
	if !ok {
		log.Info("Create writer: device: %08x", eventKey.DeviceSerial)
		writer := NewWriter()
		if eh.persist {
			filename := eh.persistFilename(eventKey.DeviceSerial)
			log.Info("Persist writer: device: %08x file: %s", eventKey.DeviceSerial, filename)
			writer.Persist(filename)
		}
		eh.writers[eventKey.DeviceSerial] = writer
	}

	// Add event for a pair of (device, event number) if it does not exist
	_, ok = eh.events[eventKey]
	if !ok {
		event := NewEvent(eventKey.DeviceSerial, eventKey.EventNum)
		eh.events[eventKey] = event
	}

	full, err := eh.events[eventKey].SetFragment(f)
	if err != nil {
		return err
	}

	if full {
		log.Info("Close event: device %08x event: %d", eventKey.DeviceSerial, eventKey.EventNum)
		err := eh.events[eventKey].Close(eh.writers[eventKey.DeviceSerial])
		if err != nil {
			return err
		}
		delete(eh.events, eventKey)

		if eventKey.EventNum > eh.maxClosedEvent {
			eh.maxClosedEvent = eventKey.EventNum
		}

		if len(eh.events) > 2 * MaxEventDiff {
			for key, event := range eh.events {
				if eh.maxClosedEvent - key.EventNum > MaxEventDiff {
					log.Info("Flush event: %d", key.EventNum)
					if event.Trigger != nil {
						log.Info("Force close event: %d", key.EventNum)
						event.Close(eh.writers[key.DeviceSerial])
					}
					delete(eh.events, key)
				}
			}
			eh.nextFlushOld += MaxEventDiff
		}
	}
	return nil
}
