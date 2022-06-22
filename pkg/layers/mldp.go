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

package layers

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket/layers"
	"sigs.k8s.io/yaml"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	IEEEOUITIA layers.IEEEOUI = 0x0012bb
	IEEEOUIAFI layers.IEEEOUI = 0x02a6b8
)

const (
	LLDPTIASubtypeIgnore       uint8 = 1
	LLDPTIASubtypeHW           uint8 = 5
	LLDPTIASubtypeFW           uint8 = 6
	LLDPTIASubtypeSerial       uint8 = 8
	LLDPTIASubtypeManufacturer uint8 = 9
	LLDPTIASubtypeModel        uint8 = 10
)

const (
	LLDPAFISubtype1 uint8 = 1
	LLDPAFISubtype2 uint8 = 2
)

type Flags struct {
	MasterLocked  uint8 `json:"masterLocked"`
	MStreamLocked uint8 `json:"mstreamLocked"`
	Unused        uint8 `json:"unused"`
}

type Mac struct {
	net.HardwareAddr
}

func (m *Mac) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *Mac) UnmarshalJSON(bytes []byte) error {
	trimmed := strings.Trim(string(bytes), "\"")
	mac, err := net.ParseMAC(trimmed)
	if err != nil {
		return err
	}
	m.HardwareAddr = mac
	return nil
}

type DeviceID uint16

func (d *DeviceID) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("0x%02x", uint16(*d)))
}

func (d *DeviceID) UnmarshalJSON(bytes []byte) error {
	trimmed := strings.Trim(string(bytes), "\"")
	deviceID, err := strconv.ParseUint(trimmed, 0, 16)
	if err != nil {
		return err
	}
	*d = DeviceID(deviceID)
	return nil
}

type DeviceDescription struct {
	DeviceID       DeviceID `json:"deviceID,omitempty"`
	SerialID       uint64   `json:"serialID,omitempty"`
	ChassisSlot    uint16   `json:"chassisSlot,omitempty"`
	MasterMac      Mac      `json:"masterMac"`
	MasterIP       net.IP   `json:"masterIP"`
	MasterUDPPort  uint16   `json:"masterUDPPort"`
	MStreamMac     Mac      `json:"mstreamMac"`
	MStreamIP      net.IP   `json:"mstreamIP"`
	MStreamUDPPort uint16   `json:"mstreamUDPPort"`
	Flags          `json:"flags,omitempty"`

	HardwareRevision string `json:"hardwareRevision,omitempty"`
	FirmwareRevision string `json:"firmwareRevision,omitempty"`
	SerialNumber     string `json:"serialNumber,omitempty"`
	ManufacturerName string `json:"manufacturerName,omitempty"`
	ModelName        string `json:"modelName,omitempty"`
	Address          net.IP `json:"address"`
	Port             uint16 `json:"port"`
	Timestamp        uint64 `json:"timestamp,omitempty"`
}

func (dd *DeviceDescription) SetSource(udpAddr *net.UDPAddr) error {
	splitted := strings.Split(udpAddr.String(), ":")
	dd.Address = net.ParseIP(splitted[0])
	convertedPort, err := strconv.Atoi(splitted[1])
	if err != nil {
		return err
	}
	dd.Port = uint16(convertedPort)
	return nil
}

func (dd *DeviceDescription) SetTimestamp() error {
	dd.Timestamp = uint64(time.Now().UnixNano()) * uint64(time.Nanosecond) / uint64(time.Millisecond)
	return nil
}

func (dd *DeviceDescription) String() string {
	result, err := yaml.Marshal(dd)
	if err != nil {
		log.Error("Error occured while marshaling device description, %s", err)
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
		dd.DeviceID = DeviceID(binary.BigEndian.Uint16(tlv.Info[:2]))
		dd.SerialID = binary.BigEndian.Uint64(streatchByteSlice(tlv.Info[2:8], 8))
		if len(tlv.Info) >= 10 {
			dd.ChassisSlot = binary.BigEndian.Uint16(tlv.Info[8:10])
		}
	case LLDPAFISubtype2:
		if len(tlv.Info) < 25 {
			break
		}
		dd.MasterMac = Mac{HardwareAddr: tlv.Info[:6]}
		dd.MasterIP = net.IPv4(tlv.Info[6], tlv.Info[7], tlv.Info[8], tlv.Info[9])
		dd.MasterUDPPort = binary.BigEndian.Uint16(tlv.Info[10:12])
		dd.MStreamMac = Mac{HardwareAddr: tlv.Info[12:18]}
		dd.MStreamIP = net.IPv4(tlv.Info[18], tlv.Info[19], tlv.Info[20], tlv.Info[21])
		dd.MStreamUDPPort = binary.BigEndian.Uint16(tlv.Info[22:24])
		dd.Flags.MasterLocked = 0b00000001 & tlv.Info[24]
		dd.Flags.MStreamLocked = (0b00000010 & tlv.Info[24]) >> 1
		dd.Flags.Unused = (0b11111100 & tlv.Info[24]) >> 2
	}
}

func DecodeOrgSpecific(tlvs []layers.LLDPOrgSpecificTLV, dd *DeviceDescription) {
	for _, tlv := range tlvs {
		switch tlv.OUI {
		case IEEEOUITIA:
			decodeTia(tlv, dd)
		case IEEEOUIAFI:
			decodeAfi(tlv, dd)
		}
	}
}
