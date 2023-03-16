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
	"container/list"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	MaxFragmentsListLength          = 100
	FragmentBuilderFragmentedChSize = 100
)

/*
 The idea of how to handle fragmented flow is adopted from here
 https://github.com/google/gopacket/blob/master/ip4defrag/defrag.go and
 http://www.sans.org/reading-room/whitepapers/detection/ip-fragment-reassembly-scapy-33969
*/

// acceptedRange = 2*(hwBufSize-1) * FRAGMENTS_IN_PACKAGE_2_2;

// FragmentBuilder holds a linked list which is used to store MStream fragment parts.
// It stores internal counters/flags to track the state of the MStream flow.
type FragmentBuilder struct {
	FragmentID uint16
	DeviceID   uint8
	Flags      uint8
	layers.Subtype
	Free                 bool
	Parts                *list.List
	Highest              uint16
	TotalLength          uint16
	LastFragmentReceived bool
	Completed            bool
	FragmentedCh         chan *layers.MStreamFragment
	defragmentedCh       chan<- *layers.MStreamFragment
	mgr                  *DefragManager
}

func NewFragmentBuilder(mgr *DefragManager, fragmentId uint16, defragmentedCh chan<- *layers.MStreamFragment) *FragmentBuilder {
	fragmentedCh := make(chan *layers.MStreamFragment, FragmentBuilderFragmentedChSize)
	return &FragmentBuilder{
		mgr:                  mgr,
		FragmentID:           fragmentId,
		DeviceID:             0,
		Flags:                0,
		Subtype:              0,
		Free:                 true,
		Parts:                list.New(),
		Highest:              0,
		TotalLength:          0,
		LastFragmentReceived: false,
		Completed:            false,
		FragmentedCh:         fragmentedCh,
		defragmentedCh:       defragmentedCh,
	}
}

func (b *FragmentBuilder) Clear() {
	log.Debug("Clear fragment builder: %s %d", b.mgr.deviceName, b.FragmentID)
	b.Free = true
	b.DeviceID = 0
	b.Flags = 0
	b.Subtype = 0
	b.Parts = list.New()
	b.Highest = 0
	b.TotalLength = 0
	b.LastFragmentReceived = false
	b.Completed = false
}

func (b *FragmentBuilder) AssembleFragment() {
	log.Debug("Assemble fragment: %s %d", b.mgr.deviceName, b.FragmentID)
	defer b.Clear()

	var data []byte
	var currentOffset uint16

	for e := b.Parts.Front(); e != nil; e = e.Next() {
		// we don't check the error here since the list contains only MStream fragments
		f, _ := e.Value.(*layers.MStreamFragment)
		if f.FragmentOffset == currentOffset { // First fragment must have offset = 0
			log.Debug("AssembleFragment: %s fragment: %d offset: %d",
				b.mgr.deviceName, b.FragmentID, f.FragmentOffset)
			data = append(data, f.Data...)
			currentOffset += f.FragmentLength
		} else {
			log.Error("Overlapping fragment or hole found: %s fragment: %d",
				b.mgr.deviceName, b.FragmentID)
			return
		}
	}

	assembled := &layers.MStreamFragment{
		DeviceID:       b.DeviceID,
		Flags:          b.Flags,
		Subtype:        b.Subtype,
		FragmentLength: b.Highest,
		FragmentID:     b.FragmentID,
		FragmentOffset: 0,
		Data:           data,
	}
	assembled.SetLastFragment(true)
	err := assembled.DecodePayload()
	if err != nil {
		log.Error("Error while decoding fragment payload: "+
			"%s %d error: %s", b.mgr.deviceName, b.FragmentID, err)
		return
	}

	b.defragmentedCh <- assembled
}

func (b *FragmentBuilder) HandleFragmentPart(f *layers.MStreamFragment) {
	if b.Free {
		b.Free = false
		b.DeviceID = f.DeviceID
		b.Flags = f.Flags
		b.Subtype = f.Subtype
	}

	if f.FragmentOffset >= b.Highest {
		log.Debug("Fragment append: %s %04x %04x", b.mgr.deviceName, b.FragmentID, f.FragmentOffset)
		b.Parts.PushBack(f)
	} else {
		log.Debug("Fragment not in order: %s %04x %04x", b.mgr.deviceName, b.FragmentID, f.FragmentOffset)
		for e := b.Parts.Front(); e != nil; e = e.Next() {
			// we don't check the error here the list contains only MStream fragments
			frag, _ := e.Value.(*layers.MStreamFragment)

			if f.FragmentOffset == frag.FragmentOffset {
				log.Debug("Fragment duplication: %s %04x %04x",
					b.mgr.deviceName, b.FragmentID, f.FragmentOffset)
				return
			}

			if f.FragmentOffset < frag.FragmentOffset {
				b.Parts.InsertBefore(f, e)
				break
			}
		}
	}

	// After inserting the fragment, we update the fragment list state
	if b.Highest < f.FragmentOffset+f.FragmentLength {
		b.Highest = f.FragmentOffset + f.FragmentLength
	}
	b.TotalLength += f.FragmentLength

	log.Debug("Fragment builder: %s %04x state: count: %d highest: %d total: %d",
		b.mgr.deviceName, b.FragmentID, b.Parts.Len(), b.Highest, b.TotalLength)

	if f.LastFragment() {
		b.LastFragmentReceived = true
	}

	// Last fragment received and the total length of all fragments corresponds
	// to the end of the last fragment which means there are no missing fragments.
	if b.LastFragmentReceived && b.Highest == b.TotalLength {
		log.Debug("Fragment completed: %s %04x", b.mgr.deviceName, b.FragmentID)
		b.Completed = true
	}

	if b.Completed {
		b.AssembleFragment()
	}
}

func (b *FragmentBuilder) Run() error {
	log.Debug("Run fragment builder: device: %s fragment id: 0x%04x", b.mgr.deviceName, b.FragmentID)
	for {
		f := <-b.FragmentedCh
		b.HandleFragmentPart(f)
	}
	return nil
}

type DefragManager struct {
	deviceName       string
	fragmentBuilders []*FragmentBuilder
	FragmentedCh     <-chan *layers.MStreamFragment
	DefragmentedCh   chan<- *layers.MStreamFragment
}

func NewDefragManager(
	deviceName string,
	fragmentedCh <-chan *layers.MStreamFragment,
	defragmentedCh chan<- *layers.MStreamFragment,
) *DefragManager {
	return &DefragManager{
		deviceName:       deviceName,
		fragmentBuilders: make([]*FragmentBuilder, 65536), // fragment id is uint16 number, thus 65536
		FragmentedCh:     fragmentedCh,
		DefragmentedCh:   defragmentedCh,
	}
}

func (m *DefragManager) Run() error {
	log.Info("Run defrag manager: %s", m.deviceName)
	for i := 0; i < 65536; i++ { // fragment id is uint16 number
		m.fragmentBuilders[i] = NewFragmentBuilder(m, uint16(i), m.DefragmentedCh)
		go func(i int) {
			m.fragmentBuilders[i].Run()
		}(i)
	}
	log.Info("Fragment builders initialized: %s", m.deviceName)
	var f *layers.MStreamFragment
	for {
		f = <-m.FragmentedCh
		log.Debug("Send fragment part: %s %d offset: %d length: %d last: %t",
			m.deviceName, f.FragmentID, f.FragmentOffset, f.FragmentLength, f.LastFragment())
		m.fragmentBuilders[f.FragmentID].FragmentedCh <- f
	}
	return nil
}
