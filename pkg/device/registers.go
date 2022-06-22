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
	RegDeviceRlat
	RegRunStatus
	RegDeviceId
	RegTrigCtrl
	RegAdcInfo
	RegChDpmKs
	RegTemperature
	RegFwVer
	RegFwRev
	RegSerialNum
	RegDacMax5501
	RegDacMax5502
	RegPca12
	RegAdcSpi
	RegAdc12SpiRead
	RegAdc34SpiRead
	RegAdc56SpiRead
	RegAdc78SpiRead
	RegZsEvents
	//RegLtm9011Spi1Addr
	//RegLtm9011Spi1WrData
	//RegLtm9011Spi2Addr
	//RegLtm9011Spi2WrData
	//RegAd9252SpiAddr
	//RegAd9252SpiWrData
	//RegAd9252SpiReadData
	//RegAd9252Csr
	//RegAd9249SpiAddr
	//RegAd9249SpiWrData
	//RegAd9249SpiReadData
	//RegAd9249Csr
	//RegAd9249ActiveAdcSelect
	//RegAds52j90SpiAddr
	//RegAds52j90SpiWrData
	//RegAds52j90SpiReadData
	//RegAds52j90Csr
	//RegAds52j90ActiveAdcSelect
	//RegAmpSet
	//RegAd5622Command
	//RegAd5622Baseline
	//RegAd5622Level
	RegMstreamRunCtrl
	RegMstreamDataSizeBytes
	RegMstreamReadoutChannelEn
	RegMstreamSparseCtrl
	RegMstreamSparseOffset
	RegMstreamSparsePeriod
	RegMstreamMtuSize
	RegDesCtrl
	RegDesStatus
	RegDesIdelayTapVal
	RegDesIdelayLoadMask
	//RegHmcad1101SpiBase
	//RegHmcad1101SpiAddrReg
	RegFirControl
	RegFirCoefCtrl
	RegFirRoundoff
	RegFirCoefStart
	RegTrigCsrTrigTs
	RegTrigCsrEvNum
	RegTrigCsrTrigInDelay
	RegTrigCsrTrigCode
	RegStatisticControl
	RegAdcStatus
	RegRunEventNumber
	RegWrSyncLostCounter
	RegWrLinkErrorCounter
	RegAdcStatusMask
	RegTrigOnXoffErrorCounter
	RegRunEventNumber64
	RegAdcTimeSec
	RegAliasLimit
)

const (
	RegFirBase     uint16 = 0x200
	RegTrigCsrBase uint16 = 0x240
)

// even more register addresses in mregdevice.h qtmregdevice.cpp
var RegMap = map[RegAlias]uint16{
	RegDeviceCtrl:   0x40,
	RegDeviceRlat:   0x41,
	RegRunStatus:    0x42,
	RegDeviceId:     0x42,
	RegTrigCtrl:     0x43,
	RegAdcInfo:      0x44,
	RegChDpmKs:      0x4A,
	RegTemperature:  0x4B,
	RegFwVer:        0x4C,
	RegFwRev:        0x4D,
	RegSerialNum:    0x4E,
	RegDacMax5501:   0x100,
	RegDacMax5502:   0x101,
	RegPca12:        0x102,
	RegAdcSpi:       0x160,
	RegAdc12SpiRead: 0x161,
	RegAdc34SpiRead: 0x162,
	RegAdc56SpiRead: 0x163,
	RegAdc78SpiRead: 0x164,
	RegZsEvents:     0x110,
	//RegLtm9011Spi1Addr: 0x160,
	//RegLtm9011Spi1WrData: 0x161,
	//RegLtm9011Spi2Addr:  0x168,
	//RegLtm9011Spi2WrData: 0x169,
	//RegAd9252SpiAddr: 0x120,
	//RegAd9252SpiWrData: 0x121,
	//RegAd9252SpiReadData: 0x122,
	//RegAd9252Csr:  0x123,
	//RegAd9249SpiAddr: 0x160,
	//RegAd9249SpiWrData: 0x161,
	//RegAd9249SpiReadData: 0x162,
	//RegAd9249Csr: 0x163,
	//RegAd9249ActiveAdcSelect:  0x164,
	//RegAds52j90SpiAddr: 0x160,
	//RegAds52j90SpiWrData: 0x161,
	//RegAds52j90SpiReadData: 0x162,
	//RegAds52j90Csr: 0x163,
	//RegAds52j90ActiveAdcSelect:  0x164,
	//RegAmpSet: 0x130,
	//RegAd5622Command: 0x131,
	//RegAd5622Baseline: 0x132,
	//RegAd5622Level: 0x133,
	RegMstreamRunCtrl:          0x140,
	RegMstreamDataSizeBytes:    0x141,
	RegMstreamReadoutChannelEn: 0x142,
	RegMstreamSparseCtrl:       0x148,
	RegMstreamSparseOffset:     0x149,
	RegMstreamSparsePeriod:     0x14A,
	RegMstreamMtuSize:          0x14C,
	RegDesCtrl:                 0x150,
	RegDesStatus:               0x151,
	RegDesIdelayTapVal:         0x153,
	RegDesIdelayLoadMask:       0x154,
	//RegHmcad1101SpiBase: 0x160,
	//RegHmcad1101SpiAddrReg: 0x168,
	RegFirControl:             RegFirBase + 0,
	RegFirCoefCtrl:            RegFirBase + 1,
	RegFirRoundoff:            RegFirBase + 2,
	RegFirCoefStart:           RegFirBase + 0x10,
	RegTrigCsrTrigTs:          RegTrigCsrBase + 0,
	RegTrigCsrEvNum:           RegTrigCsrBase + 4,
	RegTrigCsrTrigInDelay:     RegTrigCsrBase + 8,
	RegTrigCsrTrigCode:        RegTrigCsrBase + 9,
	RegStatisticControl:       0x300,
	RegAdcStatus:              0x301,
	RegRunEventNumber:         0x302,
	RegWrSyncLostCounter:      0x304,
	RegWrLinkErrorCounter:     0x306,
	RegAdcStatusMask:          0x308,
	RegTrigOnXoffErrorCounter: 0x30A,
	RegRunEventNumber64:       0x30C,
	RegAdcTimeSec:             0x1000,
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
