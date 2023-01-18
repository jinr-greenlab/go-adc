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

package ifc

import (
	"net"

	"jinr.ru/greenlab/go-adc/pkg/layers"
)

type Device interface {
	MStreamStart() error
	MStreamStop() error

	SetTrigger(bitmask uint16, val bool) error

	SetMafSelector(val int) error
	SetMafBlcThresh(val int) error

	SetInvert(val bool) error

	SetRoundoff(val uint16) error
	SetFirCoef(val []uint16) error

	SetWindowSize(val uint16) error
	SetLatency(val uint16) error

	SetChannels(val layers.ChannelsSetup) error

	SetZs(val bool) error

	RegRead(addr uint16) (*layers.Reg, error)
	RegReadAll() ([]*layers.Reg, error)
	RegWrite(reg *layers.Reg) error
	IsRunning() (bool, error)

	UpdateReg(reg *layers.Reg) error

	GetName() string
	GetIP() *net.IP
}
