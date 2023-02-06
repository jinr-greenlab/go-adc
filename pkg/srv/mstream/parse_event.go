package mstream

import (
	"encoding/binary"
	"encoding/json"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const word_size = 4
const mstream_time_header_size = 8

type Mstream_Header struct {
	Event_timestamp     uint32
	Subtype             uint8
	Length              uint16
	Tainsec             uint32
	Taiflags            uint32
	MStream_Data_Header []Mstream_Data_Header
}

type Mstream_Data_Header struct {
	Data_type uint8
	Channel   uint8
	Spec      uint8
	Length    uint16
	TDC_Data  []interface{}
	ADC_Data  ADC_Data
}

type TDC_Data2 struct {
	timestamp uint16
	evnum     uint16
	id        uint8
	n         uint8
}

type TDC_Data3 struct {
	wcount uint16
	evnum  uint16
	id     uint8
	n      uint8
}

type TDC_Data4 struct {
	rcdata uint32
	ledge  uint16
	ch     uint8
	n      uint8
}

type TDC_Data5 struct {
	rcdata uint32
	tedge  uint16
	ch     uint8
	n      uint8
}

type TDC_Data6 struct {
	err uint16
	id  uint16
	n   uint8
}

type ADC_Data struct {
	Timestamp uint16
	Length    uint16
	Voltage   []uint16
}

func Mstream_Header_Constructor(d []byte) Mstream_Header {
	e := Mstream_Header{}
	buffer := make([]uint32, len(d)/4, len(d)/4)
	for i := 0; i < len(d)/4; i++ {
		buffer[i] = binary.LittleEndian.Uint32(d[i*4 : i*4+4])
	}
	e.Length = uint16(len(d))
	e.Event_timestamp = buffer[0]
	e.Tainsec = (buffer[1] & 0xFFFFFFFC) >> 2
	e.Taiflags = buffer[1] & 0x3
	e.Subtype = 0 // only 0 subtype is present for TQDC (from TQDC2_Data_Format_rev0.pdf)

	if e.Subtype == 0 {
		doffset := 2
		mstream_data_size := int((e.Length - mstream_time_header_size) / word_size)
		for doffset != mstream_data_size {
			d1 := buffer[doffset]
			Mstream_Data_Header := Mstream_Data_Header{}
			Mstream_Data_Header.Data_type = uint8((d1 & 0xF0000000) >> 28)
			Mstream_Data_Header.Channel = uint8((d1 & 0xF000000) >> 24)
			Mstream_Data_Header.Spec = uint8((d1 & 0x70000) >> 16)
			Mstream_Data_Header.Length = uint16(d1 & 0xFFFF)
			doffset++

			if Mstream_Data_Header.Data_type == 0 { //TDC DATA
				data_size := Mstream_Data_Header.Length / 4
				for data_size != 0 {
					d1 := buffer[doffset]
					tdc_data_type := (d1 & 0xF0000000) >> 28

					if tdc_data_type == 2 {
						TDC_Data := TDC_Data2{
							uint16(d1 & 0xFFF),
							uint16((d1 & 0xFFF000) >> 12),
							uint8((d1 & 0xF000000) >> 24),
							uint8((d1 & 0xF0000000) >> 28),
						}
						Mstream_Data_Header.TDC_Data = append(Mstream_Data_Header.TDC_Data, TDC_Data)
						doffset++
					} else if tdc_data_type == 3 {
						TDC_Data := TDC_Data3{
							uint16(d1 & 0xFFF),
							uint16((d1 & 0xFFF000) >> 12),
							uint8((d1 & 0xF000000) >> 24),
							uint8((d1 & 0xF0000000) >> 28),
						}
						Mstream_Data_Header.TDC_Data = append(Mstream_Data_Header.TDC_Data, TDC_Data)
						doffset++
					} else if tdc_data_type == 4 {
						TDC_Data := TDC_Data4{
							uint32(d1 & 0x3),
							uint16((d1 & 0x1FFFFC) >> 2),
							uint8((d1 & 0x1E00000) >> 21),
							uint8((d1 & 0xF0000000) >> 28),
						}
						Mstream_Data_Header.TDC_Data = append(Mstream_Data_Header.TDC_Data, TDC_Data)
						doffset++
					} else if tdc_data_type == 5 {
						TDC_Data := TDC_Data5{
							uint32(d1 & 0x3),
							uint16((d1 & 0x1FFFFC) >> 2),
							uint8((d1 & 0x1E00000) >> 21),
							uint8((d1 & 0xF0000000) >> 28),
						}
						Mstream_Data_Header.TDC_Data = append(Mstream_Data_Header.TDC_Data, TDC_Data)
						doffset++
					} else if tdc_data_type == 6 {
						TDC_Data := TDC_Data6{
							uint16(d1 & 0x7FFF),
							uint16((d1 & 0xF000000) >> 24),
							uint8((d1 & 0xF0000000) >> 28),
						}
						Mstream_Data_Header.TDC_Data = append(Mstream_Data_Header.TDC_Data, TDC_Data)
						doffset++
					}
					data_size--
				}
			} else if Mstream_Data_Header.Data_type == 1 { //ADC Data
				ADC_Data := ADC_Data{}
				ADC_Data.Timestamp = uint16(buffer[doffset] & 0xFFFF)
				ADC_Data.Length = uint16((buffer[doffset] & 0xFFFF0000) >> 16)
				sn := int((ADC_Data.Length / 4) * 2)
				doffset++
				Voltage := make([]uint16, sn, sn)
				for s := 0; s < sn/2; s++ {
					ind := doffset + s
					d1 = buffer[ind]
					Voltage[s*2] = uint16(d1 & 0xFFFF)
					Voltage[s*2+1] = uint16((d1 & 0xFFFF0000) >> 16)
				}
				ADC_Data.Voltage = Voltage
				doffset += sn / 2
			}
			e.MStream_Data_Header = append(e.MStream_Data_Header, Mstream_Data_Header)
		}
	}

	return e
}

func Mstream_Header_JSON(d []byte) []byte {
	e := Mstream_Header_Constructor(d)
	e_json, err := json.Marshal(e)
	if err != nil {
		log.Error("Error Marshal data %s", d) //what is better to do?
		e_json = []byte{}
	}
	return e_json
}

// Waiting for a signal (chan_reqCh) to receive (chan_byteCh), parse, and send parsed data (chan_jsonCh)
func ParseEventsForApi(chan_byteCh <-chan []byte, chan_jsonCh chan<- []byte, chan_reqCh <-chan bool) {
	for {
		select {
		case <-chan_reqCh:
			chan_jsonCh <- Mstream_Header_JSON(<-chan_byteCh)
		}
		<-chan_byteCh
	}

}
