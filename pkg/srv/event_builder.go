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
	"jinr.ru/greenlab/go-adc/pkg/layers"
)

type EventBuilder struct {
	EventNum uint32
	Free bool
	TriggerChannels uint64
	DataChannels uint64

	Trigger *layers.MStreamTrigger
	Data map[layers.ChannelNum]*layers.MStreamData
}

func NewEventBuilder() *EventBuilder {
	return &EventBuilder{
		Free: true,
	}
}

func (eb *EventBuilder) CloseEvent() {
	// write event on disk and clear state
}

// SetFragment ...
// fragment payload must be decoded before calling this function
func (eb *EventBuilder) SetFragment(f *layers.MStreamFragment) {
	if !eb.Free && f.MStreamPayloadHeader.EventNum != eb.EventNum {
		eb.CloseEvent()
	}

	if eb.Free {
		eb.Free = false
		eb.EventNum = f.MStreamPayloadHeader.EventNum
	}

	if f.Subtype == layers.MStreamTriggerSubtype {
		eb.TriggerChannels = uint64(f.MStreamTrigger.HiCh) << 32 | uint64(f.MStreamTrigger.LowCh)
		eb.Trigger = f.MStreamTrigger
		if eb.DataChannels == eb.TriggerChannels {
			eb.CloseEvent()
		}
	} else if f.Subtype == layers.MStreamDataSubtype {
		eb.DataChannels |= uint64(1) << f.MStreamData.ChannelNum
		eb.Data[f.MStreamData.ChannelNum] = f.MStreamData
		if eb.Trigger != nil && eb.DataChannels == eb.TriggerChannels {
			eb.CloseEvent()
		}
	}
}


