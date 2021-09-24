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

package device

import (
	"jinr.ru/greenlab/go-adc/pkg/config"
	deviceifc "jinr.ru/greenlab/go-adc/pkg/device/ifc"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/srv/control/ifc"
)

const (
	Nch = 64
	FirRoundoffDefault = 1
	FirRoundoffMax = 3
)

type FirParams struct {
	PresetKey string
	Enabled bool
	Roundoff int
	Coef []uint16
}

func NewFirParams() *FirParams {
	f := &FirParams{
		Roundoff: FirRoundoffDefault,
		Coef: make([]uint16, 16),
		Enabled: false,
	}
	f.Coef[0] = 32767
	return f
}

func (f *FirParams) setRoundoff(value int) {
	if value < 0 {
		value = 0
	}
	if value > FirRoundoffMax {
		value = FirRoundoffMax
	}
    f.Roundoff = value
}

type DspParams struct {
	fir *FirParams
	Enabled bool
	BlcThr int
	MafEnabled bool
	MafTapSel int
	TestEnabled bool
}

func NewDspParams() *DspParams {
	return &DspParams{
		Enabled: false,
		BlcThr: 100,
		MafEnabled: false,
		MafTapSel: 2,
		TestEnabled: false,
		fir: NewFirParams(),
	}
}

func (dsp *DspParams) getMafType() int {
	result := 0
	if dsp.MafEnabled {
		result += 1
	}
	if dsp.TestEnabled {
		result += 2
	}
	return result
}

func (dsp *DspParams) getMafEnabled() bool {
	return dsp.Enabled && dsp.MafEnabled
}

func (dsp *DspParams) getTestEnabled() bool {
	return dsp.Enabled && dsp.TestEnabled
}

type ChannelSettings struct {
	Enabled bool
	BaseLine int
	TriggerEnabled bool
	TriggerThreshold int
	ZeroThreshold int
}

type Device struct {
	*config.Device
	ChSettings [Nch]*ChannelSettings
	TriggerDelay int
	InvertInput bool
	ZeroSuppressionEnabled bool
	InvertThresholdTrigger bool
	InvertZeroSupperssionThreshold bool
	SoftwareZeroSuppression bool
	MStreamEnabled bool
	dspParams *DspParams
	Run bool
	ctrl ifc.ControlServer
	state ifc.State
}

var _ deviceifc.Device = &Device{}

// NewDevice ...
func NewDevice(device *config.Device, ctrl ifc.ControlServer, state ifc.State) (*Device, error) {

	d := &Device{
		Device: device,
		TriggerDelay: 5,
		InvertInput: false,
		ZeroSuppressionEnabled: false,
		InvertThresholdTrigger: false,
		InvertZeroSupperssionThreshold: false,
		SoftwareZeroSuppression: false,
		ctrl: ctrl,
		state: state,
	}
	for i := 0; i < Nch; i++ {
		d.ChSettings[i] = &ChannelSettings{
			Enabled: true,
			BaseLine: 0,
			TriggerEnabled: true,
			TriggerThreshold: 100,
			ZeroThreshold: -0x8000,
		}
	}
	return d, nil
}

// TODO: Read from state
// RegRead ...
func (d *Device) RegRead(addr uint16) (*layers.Reg, error) {
	return d.state.GetReg(addr, d.Name)
}

// RegRead ...
func (d *Device) RegReadAll() ([]*layers.Reg, error) {
	return d.state.GetRegAll(d.Name)
}

// TODO: Write to state
// RegWrite ...
func (d *Device) RegWrite(reg *layers.Reg) error {
	ops := []*layers.RegOp{
		{
			Read: false,
			Reg: reg,
		},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

// Update ...
func (d *Device) UpdateReg(reg *layers.Reg) error {
	return d.state.SetReg(reg, d.Name)
}

// for details how to start and stop streaming data see DominoDevice::writeSettings()

// MStreamStart ...
func (d *Device) MStreamStart() error {
	var ops []*layers.RegOp
	ops = []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceCtrl], Value: 0}},
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceCtrl],	Value: 0x8000}},
		{Reg: &layers.Reg{Addr: RegMap[RegMstreamRunCtrl],	Value: 1}},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

// MStreamStop ...
func (d *Device) MStreamStop() error {
	ops := []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceCtrl], Value: 1}},
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceCtrl], Value: 0}},
		{Reg: &layers.Reg{Addr: RegMap[RegMstreamRunCtrl], Value: 0}},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

// MemWrite ...
func (d *Device) MemWrite(addr uint32, data []uint32) error {
	op := &layers.MemOp{
		Addr: addr,
		Size: uint32(len(data)),
		Data: data,
	}
	return d.ctrl.MemRequest(op, d.IP)
}

func ChBaseMemAddr(ch int) uint32 {
	return uint32(ch) << 14;
}

// IsRunning checks if RegRunStatusBitRunning bit is set
func (d *Device) IsRunning() (bool, error) {
	status, err := d.RegRead(RegMap[RegRunStatus])
	if err != nil {
		return false, err
	}
	return 0 != status.Value & RegRunStatusBitRunning, nil
}

func (d *Device) encodeChCtrlRegValue(ch int) uint16 {
	var result uint16 = 0
	if d.ChSettings[ch].Enabled {
		result |= 0x8000
	}
	if d.InvertInput {
		result |= 0x4000
	}
	if d.InvertThresholdTrigger {
		result |= 0x2000
	}
	if d.InvertZeroSupperssionThreshold {
		result |= 0x1000
	}
	if d.ChSettings[ch].TriggerEnabled {
		result |= 0x0800
	}
	result |= 0x0600 // unused 1
	if d.dspParams.getMafEnabled() {
		result |= 0x0080
	}
	if d.dspParams.getTestEnabled() {
		result |= 0x0040
	}
	result |= ((0x0003 & uint16(d.dspParams.MafTapSel)) << 4)
	return result
}

// WriteChReg ...
func (d *Device) WriteChReg(ch int, addr, data uint32) error {
	var writeAddr uint32 = MemBitSelectCtrl
	writeAddr |= addr
	writeAddr |= ChBaseMemAddr(ch)
	d.MemWrite(writeAddr, []uint32{data})
	return nil
}

func (d *Device) WriteChCtrl(ch int) error {
	d.WriteChReg(ch, MemMap[MemChCtrl], uint32(d.encodeChCtrlRegValue(ch)))
	d.WriteChReg(ch, MemMap[MemChBlcThrHi], uint32(d.dspParams.BlcThr))
	d.WriteChReg(ch, MemMap[MemChBlcThrLo], uint32(-d.dspParams.BlcThr))
	return nil
}


//func (d *Device) WriteSettings() error {
//	if d.MStreamEnabled && ! d.Run {
//		d.MStreamStop()
//	}
//
//	// if there is difference between Device settings cache and real settings
//	for i := 0; i < Nch; i++ {
//		d.WriteChCtrl(i)
//	}
//	return nil
//}
