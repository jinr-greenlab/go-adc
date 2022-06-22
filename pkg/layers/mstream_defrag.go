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
	"container/list"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"sync"
)

const (
	// TODO figure out whether there is fragmentation limit
	MaxFragmentsListLength = 100
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
	Subtype
	Free                 bool
	Parts                *list.List
	Highest              uint16
	TotalLength          uint16
	LastFragmentReceived bool
	Completed            bool
	closeCh              chan<- *MStreamFragment
	mgr                  *FragmentBuilderManager
}

func NewFragmentBuilder(mgr *FragmentBuilderManager, fragmentId uint16, closeCh chan<- *MStreamFragment) *FragmentBuilder {
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
		closeCh:              closeCh,
	}
}

func (b *FragmentBuilder) Clear() {
	//log.Info("Clear fragment builder: %s %d", b.mgr.deviceName, b.FragmentID)
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

func (b *FragmentBuilder) CloseFragment() {
	//log.Info("Close fragment: %s %d", b.mgr.deviceName, b.FragmentID)
	defer b.Clear()

	var data []byte
	var currentOffset uint16

	for e := b.Parts.Front(); e != nil; e = e.Next() {
		// we don't check the error here since the list contains only MStream fragments
		f, _ := e.Value.(*MStreamFragment)
		if f.FragmentOffset == currentOffset { // First fragment must have offset = 0
			log.Debug("CloseFragment: %s fragment: %d offset: %d",
				b.mgr.deviceName, b.FragmentID, f.FragmentOffset)
			data = append(data, f.Data...)
			currentOffset += f.FragmentLength
		} else {
			log.Error("Overlapping fragment or hole found: %s fragment: %d",
				b.mgr.deviceName, b.FragmentID)
			return
		}
	}

	assembled := &MStreamFragment{
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

	b.closeCh <- assembled
	b.mgr.SetLastClosedFragment(b.FragmentID)
}

func (b *FragmentBuilder) SetFragment(f *MStreamFragment) {
	if b.Free {
		b.Free = false
		b.DeviceID = f.DeviceID
		b.Flags = f.Flags
		b.Subtype = f.Subtype
	}

	if f.FragmentOffset >= b.Highest {
		b.Parts.PushBack(f)
	} else {
		for e := b.Parts.Front(); e != nil; e = e.Next() {
			// we don't check the error here the list contains only MStream fragments
			frag, _ := e.Value.(*MStreamFragment)

			if f.FragmentOffset == frag.FragmentOffset {
				log.Debug("Fragment duplication: %s %d",
					b.mgr.deviceName, b.FragmentID)
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

	//log.Debug("Fragment builder: %s %d state: count: %d highest: %d total: %d",
	//	b.mgr.deviceName, b.FragmentID, b.Parts.Len(), b.Highest, b.TotalLength)

	if f.LastFragment() {
		b.LastFragmentReceived = true
	}

	// Last fragment received and the total length of all fragments corresponds
	// to the end of the last fragment which means there are no missing fragments.
	if b.LastFragmentReceived && b.Highest == b.TotalLength {
		//log.Info("Fragment completed: %s %d", b.mgr.deviceName, b.FragmentID)
		b.Completed = true
	}

	if b.Completed && (b.mgr.GetLastClosedFragment()+1) == b.FragmentID {
		b.CloseFragment()
	}
}

type FragmentBuilderManager struct {
	mu         sync.RWMutex
	deviceName string
	// fragmentBuilders field is an in memory buffer which is used to store
	// MStream fragment parts as they are received and until we are
	// able to assemble them
	fragmentBuilders   []*FragmentBuilder
	closeCh            chan<- *MStreamFragment
	lastClosedFragment uint16
}

func NewFragmentBuilderManager(deviceName string, closeCh chan<- *MStreamFragment) *FragmentBuilderManager {
	log.Info("Creating FragmentBuilderManager: %s", deviceName)
	return &FragmentBuilderManager{
		deviceName:       deviceName,
		fragmentBuilders: make([]*FragmentBuilder, 65536),
		closeCh:          closeCh,
	}
}

func (m *FragmentBuilderManager) Init() {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.Info("Initializing fragment builder manager: %s", m.deviceName)
	for i := 0; i < 65536; i++ {
		m.fragmentBuilders[i] = NewFragmentBuilder(m, uint16(i), m.closeCh)
	}
	log.Info("Init last closed fragment to 65535")
	m.SetLastClosedFragment(65535)
	log.Info("Done initializing fragment builder manager: %s", m.deviceName)
}

func (m *FragmentBuilderManager) GetLastClosedFragment() uint16 {
	return m.lastClosedFragment
}

func (m *FragmentBuilderManager) SetLastClosedFragment(fragmentID uint16) {
	//log.Info("Set last closed fragment: %s %d", m.deviceName, fragmentID)
	m.lastClosedFragment = fragmentID
}

func (m *FragmentBuilderManager) SetFragment(f *MStreamFragment) {
	//log.Info("Setting fragment part: %s %d offset: %d length: %d last: %t",
	//	m.deviceName, f.FragmentID, f.FragmentOffset, f.FragmentLength, f.LastFragment())

	m.fragmentBuilders[f.FragmentID].SetFragment(f)
}
