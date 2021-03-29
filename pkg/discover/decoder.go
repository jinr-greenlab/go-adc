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

package discover

import (
	"fmt"
	"encoding/binary"
	"github.com/google/gopacket/layers"
	"sigs.k8s.io/yaml"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	IEEEOUITIA layers.IEEEOUI = 0x0012bb
	IEEEOUIAFI layers.IEEEOUI = 0x02a6b8
)

const (
	LLDPTIASubtypeIgnore uint8 = 1
	LLDPTIASubtypeHW uint8 = 5
	LLDPTIASubtypeFW uint8 = 6
	LLDPTIASubtypeSerial uint8 = 8
	LLDPTIASubtypeManufacturer uint8 = 9
	LLDPTIASubtypeModel uint8 = 10
)

const (
	LLDPAFISubtype1 uint8 = 1
	LLDPAFISubtype2 uint8 = 2
)

type Flags struct {
	MasterLocked uint8 `json:"MasterLocked,omitempty"`
	MStreamLocked uint8 `json:"MStreamLocked,omitempty"`
	Unused uint8 `json:"Unused,omitempty"`
}

type DeviceDescription struct {
	DeviceID uint16 `json:"DeviceID,omitempty"`
	SerialID uint64 `json:"SerialID,omitempty"`
	ChassisSlot uint16 `json:"ChassisSlot,omitempty"`
	MasterMac uint64 `json:"MasterMac,omitempty"`
	MasterIP uint32 `json:"MasterIP,omitempty"`
	MasterUDPPort uint16 `json:"MasterUDPPort,omitempty"`
	MStreamMac uint64 `json:"MStreamMac,omitempty"`
	MStreamIP uint32 `json:"MStreamIP,omitempty"`
	MStreamUDPPort uint16 `json:"MStreamUDPPort,omitempty"`
	Flags `json:"Flags,omitempty"`

	HardwareRevision string `json:"HardwareRevision,omitempty"`
	FirmwareRevision string `json:"FirmwareRevision,omitempty"`
	SerialNumber string `json:"SerialNumber,omitempty"`
	ManufacturerName string `json:"ManufacturerName,omitempty"`
	ModelName string `json:"ModelName,omitempty"`
}

func (dd *DeviceDescription) String() string {
	result, err := yaml.Marshal(dd)
	if err != nil {
		log.Info("Error occured while marshaling device description, %s", err)
		return ""
	}
	return fmt.Sprintf("---\n%s", string(result))
}


func streatchByteSlice(orig []byte, upToLength int) []byte {
	result := make([]byte, upToLength)
	copy(result, orig)
	return result
}

func decodeTia(tlv layers.LLDPOrgSpecificTLV, dd *DeviceDescription) {
	switch tlv.SubType {
	case LLDPTIASubtypeIgnore:
		break
	case LLDPTIASubtypeHW:
		dd.HardwareRevision = string(tlv.Info)
	case LLDPTIASubtypeFW:
		dd.FirmwareRevision = string(tlv.Info)
	case LLDPTIASubtypeSerial:
		dd.SerialNumber = string(tlv.Info)
	case LLDPTIASubtypeManufacturer:
		dd.ManufacturerName = string(tlv.Info)
	case LLDPTIASubtypeModel:
		dd.ModelName = string(tlv.Info)
	}
}

func decodeAfi(tlv layers.LLDPOrgSpecificTLV, dd *DeviceDescription) {
	switch tlv.SubType {
	case LLDPAFISubtype1:
		if len(tlv.Info) < 8 {
			break
		}
		dd.DeviceID = binary.BigEndian.Uint16(tlv.Info[:2])
		dd.SerialID = binary.BigEndian.Uint64(streatchByteSlice(tlv.Info[2:8], 8))
		if len(tlv.Info) >= 10 {
			dd.ChassisSlot = binary.BigEndian.Uint16(tlv.Info[8:10])
		}
	case LLDPAFISubtype2:
		if len(tlv.Info) < 25 {
			break
		}
		dd.MasterMac = binary.BigEndian.Uint64(streatchByteSlice(tlv.Info[:6], 8))
		dd.MasterIP = binary.BigEndian.Uint32(tlv.Info[6:10])
		dd.MasterUDPPort = binary.BigEndian.Uint16(tlv.Info[10:12])
		dd.MStreamMac = binary.BigEndian.Uint64(streatchByteSlice(tlv.Info[12:18], 8))
		dd.MStreamIP = binary.BigEndian.Uint32(tlv.Info[18:22])
		dd.MStreamUDPPort = binary.BigEndian.Uint16(tlv.Info[22:24])
		dd.Flags.MasterLocked = 0b10000000 & tlv.Info[24]
		dd.Flags.MStreamLocked = 0b01000000 & tlv.Info[24]
		dd.Flags.Unused = 0b00111111 & tlv.Info[24]
	}
}

func decodeOrgSpecific(tlvs []layers.LLDPOrgSpecificTLV, dd *DeviceDescription) {
	for _, tlv := range tlvs {
		switch tlv.OUI {
		case IEEEOUITIA:
			decodeTia(tlv, dd)
		case IEEEOUIAFI:
			decodeAfi(tlv, dd)
		}
	}
}

