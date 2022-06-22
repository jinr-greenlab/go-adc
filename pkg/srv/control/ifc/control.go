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

package ifc

import (
	"net"

	deviceifc "jinr.ru/greenlab/go-adc/pkg/device/ifc"
	"jinr.ru/greenlab/go-adc/pkg/layers"
)

type ControlServer interface {
	Run() error

	// deviceName is used to get device IP from config
	RegRequestByDeviceName(ops []*layers.RegOp, deviceName string) error
	RegRequest(ops []*layers.RegOp, IP *net.IP) error
	// deviceName is used to get device IP from config
	MemRequestByDeviceName(op *layers.MemOp, deviceName string) error
	MemRequest(op *layers.MemOp, IP *net.IP) error

	GetDeviceByName(deviceName string) (deviceifc.Device, error)
	GetAllDevices() map[string]deviceifc.Device
}
