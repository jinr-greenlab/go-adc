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

package control

// Numbers of control registers are in mregdevice.h
// Memory addresses are in DominoDeviceRegisters.h

type RegAlias int

const (
	RegCtrl RegAlias = iota
	RegWin
	RegStatus
	RegTrigMult
	RegSerialIDHi
	RegLiveMagic
	RegChDpmKs
	RegTemperature
	RegFwVer
	RegFwRev
	RegSerialID
	RegMstreamCfg
	RegTsReadA
	RegTsSetReg64
	RegTsReadB
	RegTsReadReg64
	RegAliasLimit
)

var RegMap = map[RegAlias]uint16{
	RegCtrl: 0x40,
	RegWin: 0x41,
	RegStatus: 0x42,
	RegTrigMult: 0x43,
	RegSerialIDHi: 0x46,
	RegLiveMagic: 0x48,
	RegChDpmKs: 0x4A,
	RegTemperature: 0x4B,
	RegFwVer: 0x4C,
	RegFwRev: 0x4D,
	RegSerialID: 0x4E,
	RegMstreamCfg: 0x4F,
	RegTsReadA: 0x50,
	RegTsSetReg64: 0x54,
	RegTsReadB: 0x58,
	RegTsReadReg64: 0x5C,
}


type MemAlias int

const (
	MemChRWAddr MemAlias = iota
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
	MemChRWAddr: 0x0000,
	MemChCtrl: 0x0001,
	MemChThr: 0x0002,
	MemChZsThr: 0x0003,
	MemChBaseline: 0x0004,
	MemChAdcData: 0x0005,
	MemChAdcPattern: 0x0006,
	MemChAdcPatternMismatchCnt: 0x0007,
	MemChBlcThrHi: 0x0008,
	MemChBlcThrLo: 0x0009,
	MemChD2Hist: 0x0080,
	MemChD2HistCtrl: 0x00C0,
	MemChD2HistSt: 0x00C1,
	MemChD2HistTime: 0x00C2,

}
