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
)

type EventBuilder struct {
	deviceName     string
	defragmentedCh <-chan *layers.MStreamFragment
	writerCh       chan<- []byte
	lastEventCh    chan []byte
}

// NewEventBuilder ...
func NewEventBuilder(deviceName string, defragmentedCh <-chan *layers.MStreamFragment, writerCh chan<- []byte, lastEventCh chan []byte) *EventBuilder {
	return &EventBuilder{
		deviceName:     deviceName,
		defragmentedCh: defragmentedCh,
		writerCh:       writerCh,
		lastEventCh:    lastEventCh,
	}
}

// SetFragment ...
// fragment payload must be decoded before calling this function
func (b *EventBuilder) HandleFragment(f *layers.MStreamFragment) {
	// We substruct 8 bytes from the fragment length because fragment payload
	// includes MStreamPayloadHeader which is not included in MPD data
	length := uint32(f.FragmentLength - 8)
	//  device header length = size of data + size of MpdMStreamHeader
	deviceHeaderLength := length + 4
	// event header length = device header length + size of MpdDeviceHeader
	eventHeaderLength := deviceHeaderLength + 8

	mpd := &layers.MpdLayer{
		MpdEventHeader: &layers.MpdEventHeader{
			Sync:     layers.MpdSyncMagic,
			EventNum: f.MStreamPayloadHeader.EventNum,
			Length:   eventHeaderLength,
		},
		MpdDeviceHeader: &layers.MpdDeviceHeader{
			DeviceSerial: f.MStreamPayloadHeader.DeviceSerial,
			DeviceID:     f.DeviceID,
			Length:       deviceHeaderLength,
		},
		Data: f.MStreamData,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, mpd)
	if err != nil {
		log.Error("Error while serializing Mpd layer: device: %08x, event: %s",
			f.MStreamPayloadHeader.DeviceSerial, f.MStreamPayloadHeader.EventNum)
		return
	}

	select {
	case <-b.lastEventCh:
		b.lastEventCh <- f.Data
	default:
		b.lastEventCh <- f.Data
	}

	b.writerCh <- buf.Bytes()
}

// Run ...
func (b *EventBuilder) Run() {
	log.Info("Run event builder: %s", b.deviceName)
	var f *layers.MStreamFragment
	for {
		f = <-b.defragmentedCh
		log.Info("Setting event fragment: device %s event: %d", b.deviceName, f.MStreamPayloadHeader.EventNum)
		b.HandleFragment(f)
	}
}
