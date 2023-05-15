package mstream

import (
	"encoding/binary"
	"encoding/json"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	wordSize = 4
)

type MstreamHeader struct {
	EventTimestamp    uint32
	Subtype           uint8
	Length            uint16
	Tainsec           uint32
	Taiflags          uint32
	MStreamDataHeader []MstreamDataHeader
}

type MstreamDataHeader struct {
	DataType uint8
	Channel  uint8
	Spec     uint8
	Length   uint16
	TdcData  []interface{}
	ADCData  AdcData
}

type TdcData2 struct {
	Timestamp uint16
	Evnum     uint16
	Id        uint8
	N         uint8
}

type TdcData3 struct {
	Wcount uint16
	Evnum  uint16
	Id     uint8
	N      uint8
}

type TdcData4 struct {
	Rcdata uint32
	Ledge  uint16
	Ch     uint8
	N      uint8
}

type TdcData5 struct {
	Rcdata uint32
	Tedge  uint16
	Ch     uint8
	N      uint8
}

type TdcData6 struct {
	Err uint16
	Id  uint16
	N   uint8
}

type AdcData struct {
	Timestamp uint16
	Length    uint16
	Voltage   []uint16
}

type RepeatToApi struct {
	EventNumber    uint32
	Timestamp      uint32
	ChannelVoltage []ChannelVoltage
}

type ChannelVoltage struct {
	Channel uint8
	Voltage []uint16
}

// Encode []byte to struct Mstrean signal doc from TQDC2_Data_Format_rev0.pdf and tqdc16vs.py
func NewMstreamHeader(d []byte) MstreamHeader {
	e := MstreamHeader{}
	buffer := make([]uint32, len(d)/4, len(d)/4)
	for i := 0; i < len(d)/4; i++ {
		buffer[i] = binary.LittleEndian.Uint32(d[i*4 : i*4+4])
	}
	e.Length = uint16(len(d))
	e.EventTimestamp = buffer[0]
	e.Tainsec = (buffer[1] & 0xFFFFFFFC) >> 2
	e.Taiflags = buffer[1] & 0x3
	e.Subtype = 0 // only 0 subtype is present for TQDC (from TQDC2_Data_Format_rev0.pdf)

	mstreamDataSize := int(e.Length / wordSize)
	if e.Subtype == 0 {
		doffset := 2
		for doffset != mstreamDataSize {
			d1 := buffer[doffset]
			mstreamDataHeader := MstreamDataHeader{}
			mstreamDataHeader.DataType = uint8((d1 & 0xF0000000) >> 28)
			mstreamDataHeader.Channel = uint8((d1 & 0xF000000) >> 24)
			mstreamDataHeader.Spec = uint8((d1 & 0x70000) >> 16)
			mstreamDataHeader.Length = uint16(d1 & 0xFFFF)
			doffset++
			if mstreamDataHeader.DataType == 0 { //TDC DATA
				dataSize := mstreamDataHeader.Length / 4
				for dataSize != 0 {
					d1 = buffer[doffset]
					tdcDataType := (d1 & 0xF0000000) >> 28

					switch tdcDataType {
					case 2:
						tdcData := TdcData2{
							uint16(d1 & 0xFFF),
							uint16((d1 & 0xFFF000) >> 12),
							uint8((d1 & 0xF000000) >> 24),
							uint8((d1 & 0xF0000000) >> 28),
						}
						mstreamDataHeader.TdcData = append(mstreamDataHeader.TdcData, tdcData)
						doffset++ //important: +1 only cases 2,3,4,5,6
					case 3:
						tdcData := TdcData3{
							uint16(d1 & 0xFFF),
							uint16((d1 & 0xFFF000) >> 12),
							uint8((d1 & 0xF000000) >> 24),
							uint8((d1 & 0xF0000000) >> 28),
						}
						mstreamDataHeader.TdcData = append(mstreamDataHeader.TdcData, tdcData)
						doffset++
					case 4:
						tdcData := TdcData4{
							d1 & 0x3,
							uint16((d1 & 0x1FFFFC) >> 2),
							uint8((d1 & 0x1E00000) >> 21),
							uint8((d1 & 0xF0000000) >> 28),
						}
						mstreamDataHeader.TdcData = append(mstreamDataHeader.TdcData, tdcData)
						doffset++
					case 5:
						tdcData := TdcData5{
							d1 & 0x3,
							uint16((d1 & 0x1FFFFC) >> 2),
							uint8((d1 & 0x1E00000) >> 21),
							uint8((d1 & 0xF0000000) >> 28),
						}
						mstreamDataHeader.TdcData = append(mstreamDataHeader.TdcData, tdcData)
						doffset++
					case 6:
						tdcData := TdcData6{
							uint16(d1 & 0x7FFF),
							uint16((d1 & 0xF000000) >> 24),
							uint8((d1 & 0xF0000000) >> 28),
						}
						mstreamDataHeader.TdcData = append(mstreamDataHeader.TdcData, tdcData)
						doffset++
					}
					dataSize--
				}
			} else if mstreamDataHeader.DataType == 1 { //ADC Data
				adcData := AdcData{}
				adcData.Timestamp = uint16(buffer[doffset] & 0xFFFF)
				adcData.Length = uint16((buffer[doffset] & 0xFFFF0000) >> 16)
				sn := int((adcData.Length / 4) * 2)
				doffset++
				voltage := make([]uint16, sn, sn)
				for s := 0; s < sn/2; s++ {
					ind := doffset + s
					d1 = buffer[ind]
					voltage[s*2] = uint16(d1 & 0xFFFF)
					voltage[s*2+1] = uint16((d1 & 0xFFFF0000) >> 16)
				}
				adcData.Voltage = voltage
				mstreamDataHeader.ADCData = adcData
				doffset += sn / 2
			}
			e.MStreamDataHeader = append(e.MStreamDataHeader, mstreamDataHeader)
		}
	}

	return e
}

func BuildStructToApi(Ev MstreamHeader, EventNumber uint32) RepeatToApi {
	RepeatToApi := RepeatToApi{}
	RepeatToApi.EventNumber = EventNumber
	RepeatToApi.Timestamp = Ev.EventTimestamp
	for _, c := range Ev.MStreamDataHeader {
		ChannelVoltage := ChannelVoltage{}
		ChannelVoltage.Channel = c.Channel
		ChannelVoltage.Voltage = c.ADCData.Voltage
		RepeatToApi.ChannelVoltage = append(RepeatToApi.ChannelVoltage, ChannelVoltage)
	}

	return RepeatToApi
}

func MstreamHeaderJson(d LastEvent) []byte {
	e := NewMstreamHeader(d.Data)
	n := BuildStructToApi(e, d.EventNumber)
	eJson, err := json.Marshal(n)
	if err != nil {
		log.Error("Error Marshal data %s", d) //what is better to do?
		eJson = []byte{}
	}
	return eJson
}
