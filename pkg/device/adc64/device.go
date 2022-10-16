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

package adc64

import (
	"net"

	"jinr.ru/greenlab/go-adc/pkg/config"
	deviceifc "jinr.ru/greenlab/go-adc/pkg/device/ifc"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/srv/control/ifc"
)

const (
	Nch                = 64
	FirRoundoffDefault = 1
	FirRoundoffMax     = 3
)

type FwVersion struct {
	Major    uint16
	Minor    uint16
	Revision uint16
}

type FirParams struct {
	PresetKey string
	Enabled   bool
	Roundoff  uint16 //int
	Coef      []uint16
}

func NewFirParams() *FirParams {
	f := &FirParams{
		Roundoff: FirRoundoffDefault,
		Coef:     make([]uint16, 16),
		Enabled:  false,
	}
	f.Coef[0] = 32767
	return f
}

func (f *FirParams) setRoundoff(value uint16) {
	if value < 0 {
		value = 0
	}
	if value > FirRoundoffMax {
		value = FirRoundoffMax
	}
	f.Roundoff = value
}

type DspParams struct {
	fir         *FirParams
	Enabled     bool
	BlcThr      int
	MafEnabled  bool
	MafTapSel   int
	TestEnabled bool
}

func NewDspParams() *DspParams {
	return &DspParams{
		Enabled:     false,
		BlcThr:      100,
		MafEnabled:  false,
		MafTapSel:   2,
		TestEnabled: false,
		fir:         NewFirParams(),
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
	Enabled          bool
	BaseLine         int
	TriggerEnabled   bool
	TriggerThreshold int
	ZeroThreshold    int
}

type Device struct {
	*config.Device
	fwVersion                      *FwVersion
	ChSettings                     [Nch]*ChannelSettings
	TriggerDelay                   int
	InvertInput                    bool
	ZeroSuppressionEnabled         bool
	InvertThresholdTrigger         bool
	InvertZeroSupperssionThreshold bool
	SoftwareZeroSuppression        bool
	dspParams                      *DspParams
	ctrl                           ifc.ControlServer
	state                          ifc.State
}

var _ deviceifc.Device = &Device{}

// NewDevice ...
func NewDevice(device *config.Device, ctrl ifc.ControlServer, state ifc.State) (*Device, error) {
	d := &Device{
		Device:                         device,
		fwVersion:                      nil,
		TriggerDelay:                   5,
		InvertInput:                    false,
		ZeroSuppressionEnabled:         false,
		InvertThresholdTrigger:         false,
		InvertZeroSupperssionThreshold: false,
		SoftwareZeroSuppression:        false,
		ctrl:                           ctrl,
		state:                          state,
	}
	for i := 0; i < Nch; i++ {
		d.ChSettings[i] = &ChannelSettings{
			Enabled:          true,
			BaseLine:         0,
			TriggerEnabled:   true,
			TriggerThreshold: 100,
			ZeroThreshold:    -0x8000,
		}
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

func (d *Device) ReadFirmware() error {
	ver, err := d.RegRead(RegMap[RegFwVer])
	if err != nil {
		return err
	}

	rev, err := d.RegRead(RegMap[RegFwRev])
	if err != nil {
		return err
	}

	d.fwVersion.Major = (ver.Value >> 8) & 0xFF
	d.fwVersion.Minor = ver.Value & 0xFF
	d.fwVersion.Revision = rev.Value

	return nil
}

func (d *Device) HasAdcRawDataSigned() bool {
	d.ReadFirmware()
	if d.fwVersion == nil {
		return true
	}

	if d.fwVersion.Major >= 1 || //fix formatting
		(d.fwVersion.Major == 1 && d.fwVersion.Minor >= 0) ||
		(d.fwVersion.Major == 1 && d.fwVersion.Minor == 0 && d.fwVersion.Revision >= 23232) {
		return true
	}

	return false
}

func (d *Device) TruncateValue(val int) int {
	if d.HasAdcRawDataSigned() {
		if val < -32768 {
			val = -32768
		}
		if val > 32767 {
			val = 32767
		}
	} else {
		val += 0x8000
		if val < 0 {
			val = 0
		}
		if val > 0xFFFF {
			val = 0xFFFF
		}
	}

	return val
}

func (d *Device) SetTriggerTimer(val bool) error {
	state, err := d.RegRead(RegMap[RegTrigCtrl])
	reg := state.Value
	if err != nil {
		return err
	}

	if val {
		reg |= RegTrigStatusBitTimer
	} else {
		reg &= ^RegTrigStatusBitTimer
	}

	ops := []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegTrigCtrl], Value: reg}},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

func (d *Device) SetTriggerThreshold(val bool) error {
	state, err := d.RegRead(RegMap[RegTrigCtrl])
	reg := state.Value
	if err != nil {
		return err
	}

	if val {
		reg |= RegTrigStatusBitThreshold
	} else {
		reg &= ^RegTrigStatusBitThreshold
	}

	ops := []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegTrigCtrl], Value: reg}},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

func (d *Device) SetTriggerLemo(val bool) error {
	state, err := d.RegRead(RegMap[RegTrigCtrl])
	reg := state.Value
	if err != nil {
		return err
	}

	if val {
		reg |= RegTrigStatusBitLemo
	} else {
		reg &= ^RegTrigStatusBitLemo
	}

	ops := []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegTrigCtrl], Value: reg}},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

func (d *Device) SetMafSelector(val int) error {
	d.dspParams.Enabled = true //need setters?
	d.dspParams.MafEnabled = true
	d.dspParams.MafTapSel = val

	for i := 0; i < Nch; i++ {
		d.WriteChReg(i, MemMap[MemChCtrl], uint32(d.encodeChCtrlRegValue(i)))
	}

	return nil
}

func (d *Device) SetMafBlcThresh(val int) error {
	d.dspParams.BlcThr = val //need setters?

	for i := 0; i < Nch; i++ {
		d.WriteChCtrl(i)
	}

	return nil
}

func (d *Device) SetInvert(val bool) error {
	d.InvertInput = val
	for i := 0; i < Nch; i++ {
		d.WriteChCtrl(i)
	}

	return nil
}

func (d *Device) SetRoundoff(val uint16) error {
	d.dspParams.fir.setRoundoff(val)
	for i := 0; i < Nch; i++ {
		d.WriteChCtrl(i)
	}

	return nil
}

func (d *Device) SetFirCoef(val []uint16) error {
	ops := []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegFirControl], Value: 1}}, //need to implement setting fir on/of
		{Reg: &layers.Reg{Addr: RegMap[RegFirRoundoff], Value: d.dspParams.fir.Roundoff}},
	}

	for i := 0; i < 16; i++ {
		ops = append(ops, &layers.RegOp{Reg: &layers.Reg{Addr: RegMap[RegFirCoefStart] + uint16(i), Value: val[i]}})
	}

	ops = append(ops, &layers.RegOp{Reg: &layers.Reg{Addr: RegMap[RegFirCoefCtrl], Value: 1}},
		&layers.RegOp{Reg: &layers.Reg{Addr: RegMap[RegFirCoefCtrl], Value: 0}})

	return d.ctrl.RegRequest(ops, d.IP)
}

func (d *Device) SetWindowSize(val uint16) error {
	ops := []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegMstreamDataSizeBytes], Value: val}},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

func (d *Device) SetLatency(val uint16) error {
	ops := []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceRlat], Value: val}},
	}
	return d.ctrl.RegRequest(ops, d.IP)
}

func (d *Device) SetChannels(val layers.ChannelsSetup) error {
	for i := 0; i < len(val.Channels); i++ {
		id := val.Channels[i].Id
		d.ChSettings[id].Enabled = val.Channels[id].En
		d.ChSettings[id].TriggerEnabled = val.Channels[id].TrigEn
		d.ChSettings[id].TriggerThreshold = val.Channels[id].TrigThr //to fix
		d.WriteChReg(id, MemMap[MemChCtrl], uint32(d.encodeChCtrlRegValue(i)))

		d.WriteChReg(id, MemMap[MemChBaseline], uint32(val.Channels[id].Baseline))

		thr := d.TruncateValue(val.Channels[id].ZsThr)
		d.WriteChReg(id, MemMap[MemChZsThr], uint32(thr))

		thr = d.TruncateValue(val.Channels[id].TrigThr)
		d.WriteChReg(id, MemMap[MemChThr], uint32(thr))
	}
	return nil
}

func (d *Device) SetZs(val bool) error {
	d.ZeroSuppressionEnabled = val

	return nil
}

// for details how to start and stop streaming data see DominoDevice::writeSettings()

// MStreamStart ...
func (d *Device) MStreamStart() error {
	var ercBit uint16 = 1
	if d.ZeroSuppressionEnabled {
		ercBit = 2
	}

	var ops []*layers.RegOp
	ops = []*layers.RegOp{
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceCtrl], Value: 0}},
		{Reg: &layers.Reg{Addr: RegMap[RegDeviceCtrl], Value: 0x8000}},
		{Reg: &layers.Reg{Addr: RegMap[RegMstreamRunCtrl], Value: ercBit}},
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
	return uint32(ch) << 14
}

// IsRunning checks if RegRunStatusBitRunning bit is set
func (d *Device) IsRunning() (bool, error) {
	status, err := d.RegRead(RegMap[RegRunStatus])
	if err != nil {
		return false, err
	}
	return 0 != status.Value&RegRunStatusBitRunning, nil
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
