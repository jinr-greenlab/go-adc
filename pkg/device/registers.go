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

type RegAlias int

const (
	RegDeviceCtrl RegAlias = iota
	Reg32TrigEventNumLoad
	RegTrigCsr
	Reg32TdcGenCtrl
	RegAliasLimit
)

// even more register addresses in mregdevice.h qtmregdevice.cpp
var RegMap = map[RegAlias]uint16{
	RegDeviceCtrl:         0x40,
	Reg32TrigEventNumLoad: 0x104,
	RegTrigCsr:            0x100,
	Reg32TdcGenCtrl:       0x220,
}

const (
	RegRunStatusBitRunning uint16 = 0x0010
)

const (
	RegTrigStatusBitTimer     uint16 = 0x001
	RegTrigStatusBitThreshold uint16 = 0x002
	RegTrigStatusBitLemo      uint16 = 0x004
)

type MemAlias int

const (
	MemChWrAddr MemAlias = iota
	MemChCtrl
	MemChThr
	MemChZsThr
	MemChBaseline
	MemChAdcData
	MemChAdcPattern
	MemChAdcPatternMismatchCnt
	MemChBlcThrHi
	MemChBlcThrLo
	MemChD2Hist
	MemChD2HistCtrl
	MemChD2HistSt
	MemChD2HistTime
	MemAliasLimit
)

var MemMap = map[MemAlias]uint32{
	MemChWrAddr:                0x0000,
	MemChCtrl:                  0x0001,
	MemChThr:                   0x0002,
	MemChZsThr:                 0x0003,
	MemChBaseline:              0x0004,
	MemChAdcData:               0x0005,
	MemChAdcPattern:            0x0006,
	MemChAdcPatternMismatchCnt: 0x0007,
	MemChBlcThrHi:              0x0008,
	MemChBlcThrLo:              0x0009,
	MemChD2Hist:                0x0080,
	MemChD2HistCtrl:            0x00C0,
	MemChD2HistSt:              0x00C1,
	MemChD2HistTime:            0x00C2,
}

const (
	MemBitSelectCtrl = 1 << 13 // bit13==1 (bus 15:0) - register operation
)
