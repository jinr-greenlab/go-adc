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
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
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

// fragmentList holds a list used to contain MStream framgments.
// It stores internal counters/flags to track the state of the MStream flow.
type fragmentList struct {
	List                 list.List
	Highest              uint16
	TotalLength          uint16
	LastFragmentReceived bool
}

// insert inserts MStream fragment into the fragment list
func (fl *fragmentList) insert(ms *MStreamLayer) (*MStreamLayer, error) {

	if ms.FragmentOffset >= fl.Highest {
		fl.List.PushBack(ms)
	} else {
		for e := fl.List.Front(); e != nil; e = e.Next() {
			// we don't check the error here the list contains only MStream fragments
			frag, _ := e.Value.(*MStreamLayer)

			if ms.FragmentOffset == frag.FragmentOffset {
				log.Debug("defrag: insert: ignoring fragment %d as we already have it (duplicate?)",
					ms.FragmentOffset)
				return nil, nil
			}

			if ms.FragmentOffset < frag.FragmentOffset {
				log.Debug("defrag: insert: inserting fragment %d before existing fragment %d",
					ms.FragmentOffset, frag.FragmentOffset)
				fl.List.InsertBefore(ms, e)
				break
			}
		}
	}

	// After inserting the fragment, we update the fragment list state
	if fl.Highest < ms.FragmentOffset + ms.FragmentLength {
		fl.Highest = ms.FragmentOffset + ms.FragmentLength
	}
	fl.TotalLength += ms.FragmentLength

	log.Debug("defrag: insert: fragment list state: fragments count: %d highest length: %d total length: %d",
		fl.List.Len(), fl.Highest, fl.TotalLength)

	if ms.LastFragment() {
		fl.LastFragmentReceived = true
	}

	// Last fragment received and the total length of all fragments corresponds
	// to the end of the last fragment which means there are no missing fragments.
	if fl.LastFragmentReceived && fl.Highest == fl.TotalLength {
		return fl.assemble(ms)
	}
	return nil, nil
}

// assemble builds MStream frame from its fragments placed in fragment list
func (fl *fragmentList) assemble(ms *MStreamLayer) (*MStreamLayer, error) {
	var result []byte
	var currentOffset uint16

	log.Debug("defrag: assemble: assembling the MStream frame from fragments")

	for e := fl.List.Front(); e != nil; e = e.Next() {
		// we don't check the error here since the list contains only MStream fragments
		frag, _ := e.Value.(*MStreamLayer)
		if frag.FragmentOffset == currentOffset { // First fragment must have offset = 0
			log.Debug("defrag: assemble: add fragment id: %d offset: %d", frag.FragmentID, frag.FragmentOffset)
			result = append(result, frag.Payload...)
			currentOffset += frag.FragmentLength
		} else {
			// Houston, we have a problem.
			return nil, ErrMStreamAssemble{ What: "overlapping fragment or hole found while assembling" }
		}
		log.Debug("defrag: assemble: next id: %d offset: %d", ms.FragmentID, currentOffset)
	}

	out := &MStreamLayer{
		MStreamHeader: MStreamHeader{
			DeviceID: ms.DeviceID,
			Flags:    ms.Flags,
			Subtype:  ms.Subtype,
			FragmentLength: fl.Highest,
			FragmentID: ms.FragmentID,
			FragmentOffset: 0,
		},
		BaseLayer: layers.BaseLayer{
			Payload: result,
		},
	}
	out.SetLastFragment(true)

	return out, nil
}


// fragmentListKey is used as a map key. It fully identifies the fragmented
// MStream frame since it contains two MLink endpoints (see gopacket.Flow and gopacket.Endpoint)
// plus FragmentID which is the same for all fragments within a MStream frame
// and different for different MStream frames.
type fragmentListKey struct {
	*gopacket.Flow
	FragmentID uint16
}

func newFragmentListKey(ms *MStreamLayer, flow *gopacket.Flow) fragmentListKey {
	return fragmentListKey{
		Flow: flow,
		FragmentID: ms.FragmentID,
	}
}

// MStreamDefragmenter is a struct which embeds a map of fragment lists.
type MStreamDefragmenter struct {
	sync.RWMutex
	// Fragments field is an in memory buffer which is used to store
	// MStream frame fragments as they are received and until we are
	// able to assemble a whole MStream frame.
	fragments map[fragmentListKey]*fragmentList
}

func (md *MStreamDefragmenter) Defrag(ms *MStreamLayer, flow *gopacket.Flow) (*MStreamLayer, error) {
	log.Debug("defrag: got a new fragment FragmentID: %d FragmentOffset: %d LastFragment: %t",
		ms.FragmentID, ms.FragmentOffset, ms.LastFragment())

	key := newFragmentListKey(ms, flow)
	var fl *fragmentList
	md.Lock()
	fl, ok := md.fragments[key]
	if !ok {
		log.Debug("defrag: unknown flow, creating a new one")
		fl = new(fragmentList)
		md.fragments[key] = fl
	}
	md.Unlock()

	out, err := fl.insert(ms)

	// drop fragment list if maximum fragment list length is achieved
	if out == nil && fl.List.Len() + 1 > MaxFragmentsListLength {
		md.flush(key)
		return nil, ErrMStreamTooManyFragments{Number: MaxFragmentsListLength}
	}

	// packet is defragmented
	if out != nil {
		md.flush(key)
		return out, nil
	}

	return nil, err
}

// flush the fragment list for a particular key
// Reasons might be different, e.g. maximum number of fragments is achieved
// or defragmentation is done or timed out
func (md *MStreamDefragmenter) flush(key fragmentListKey) {
	md.Lock()
	delete(md.fragments, key)
	md.Unlock()
}

// NewMStreamDefragmenter returns a new MStreamDefragmenter with the initialized map of fragment lists.
func NewMStreamDefragmenter() *MStreamDefragmenter {
	return &MStreamDefragmenter{
		fragments: make(map[fragmentListKey]*fragmentList),
	}
}

