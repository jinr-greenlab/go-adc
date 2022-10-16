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

package tqdc

import (
	"net"

	"jinr.ru/greenlab/go-adc/pkg/config"
	deviceifc "jinr.ru/greenlab/go-adc/pkg/device/ifc"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/srv/control/ifc"
)

const (
	Nch = 64
)

type Device struct {
	*config.Device
	ctrl      ifc.ControlServer
	state     ifc.State
	isRunning bool
}

var _ deviceifc.Device = &Device{}

// NewDevice ...
func NewDevice(device *config.Device, ctrl ifc.ControlServer, state ifc.State) (*Device, error) {
	d := &Device{
		Device:    device,
		ctrl:      ctrl,
		state:     state,
		isRunning: false,
	}
	return d, nil
}

// GetName ...
func (d *Device) GetName() string {
	return d.Name
}

// GetIP ...
func (d *Device) GetIP() *net.IP {
	return d.IP
}

// GetType ...
func (d *Device) GetType() string {
	return d.Type
}

// TODO: Read from state
// RegRead ...
func (d *Device) RegRead(addr uint16) (*layers.Reg, error) {
	return d.state.GetReg(addr, d.Name)
}

// RegReadAll ...
func (d *Device) RegReadAll() ([]*layers.Reg, error) {
	regs := []uint16{}
	for _, addr := range RegMap {
		regs = append(regs, addr)
	}
	return d.state.GetRegs(d.Name, regs)
}

// TODO: Write to state
// RegWrite ...
func (d *Device) RegWrite(reg *layers.Reg) error {
	ops := []*layers.RegOp{
		{
			Read: false,
			Reg:  reg,
		},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

// Update ...
func (d *Device) UpdateReg(reg *layers.Reg) error {
	return d.state.SetReg(reg, d.Name)
}

// SetTriggerTimer not implemented
func (d *Device) SetTriggerTimer(val bool) error {
	return nil
}

// SetTriggerThreshold not implemented
func (d *Device) SetTriggerThreshold(val bool) error {
	return nil
}

// SetTriggerLemo not implemented
func (d *Device) SetTriggerLemo(val bool) error {
	return nil
}

// SetMafSelector not implemented
func (d *Device) SetMafSelector(val int) error {
	return nil
}

// SetMafBlcThresh not implemented
func (d *Device) SetMafBlcThresh(val int) error {
	return nil
}

// SetInvert not implemented
func (d *Device) SetInvert(val bool) error {
	return nil
}

// SetRoundoff not implemented
func (d *Device) SetRoundoff(val uint16) error {
	return nil
}

// SetFirCoef not implemented
func (d *Device) SetFirCoef(val []uint16) error {
	return nil
}

// SetWindowSize not implemented
func (d *Device) SetWindowSize(val uint16) error {
	return nil
}

// SetLatency not implemented
func (d *Device) SetLatency(val uint16) error {
	return nil
}

// SetChannels not implemented
func (d *Device) SetChannels(val layers.ChannelsSetup) error {
	return nil
}

// SetZs not implemented
func (d *Device) SetZs(val bool) error {
	return nil
}

// for details how to start and stop streaming data see TQDCDevice::writeSetup()

// MStreamStart ...
func (d *Device) MStreamStart() error {
	var ops []*layers.RegOp
	ops = []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceCtrl], Value: 0x8000}},
	}
	if err := d.ctrl.RegRequest(ops, d.IP); err != nil {
		return err
	}
	d.isRunning = true
	return nil
}

// MStreamStop ...
func (d *Device) MStreamStop() error {
	ops := []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceCtrl], Value: 0}},
	}
	if err := d.ctrl.RegRequest(ops, d.IP); err != nil {
		return err
	}
	d.isRunning = false
	return nil
}

// MemWrite ...
func (d *Device) MemWrite(addr uint32, data []uint32) error {
	return nil
}

// IsRunning checks if isRunning flag is set
func (d *Device) IsRunning() (bool, error) {
	return d.isRunning, nil
}

// WriteChReg ...
func (d *Device) WriteChReg(ch int, addr, data uint32) error {
	return nil
}

func (d *Device) WriteChCtrl(ch int) error {
	return nil
}
