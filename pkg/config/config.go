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

package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

type TrigSetup struct {
	Timer     bool `json:"Timer"`
	Threshold bool `json:"Threshold"`
	Lemo      bool `json:"Lemo"`
}

type MAFSetup struct {
	Selector int
	BLC      int
}

type InvertSetup struct {
	Invert bool
}

type ReadoutWindowSetup struct {
	Size    uint16
	Latency uint16
}

type ZsSetup struct {
	Zs bool
}

type Inventory struct {
	Version    uint8 `json:"version"`
	DetectorID uint8 `json:"detectorID"` // 33 for NDLAr
}

type DeviceInventory struct {
	CrateID uint16 `json:"crateID"`
	SlotID  uint8  `json:"slotID"`
}

type Device struct {
	Name                string  `json:"name,omitempty"`
	IP                  *net.IP `json:"ip,omitempty"`
	*TrigSetup          `json:"TriggerSetup,omitempty"`
	*MAFSetup           `json:"MafSetup,omitempty"`
	*InvertSetup        `json:"InvertSetup,omitempty"`
	*ReadoutWindowSetup `json:"ReadoutWindowSetup,omitempty"`
	*ZsSetup            `json:"ZsSetup,omitempty"`
	*DeviceInventory    `json:"inventory,omitempty"`
}

type Config struct {
	LogLevel      string     `json:"logLevel,omitempty"`
	DiscoverIP    *net.IP    `json:"discoverIP,omitempty"`
	DiscoverIface string     `json:"discoverIface,omitempty"`
	IP            *net.IP    `json:"ip,omitempty"`
	Devices       []*Device  `json:"devices"`
	Inventory     *Inventory `json:"inventory,omitempty"`
	dirpath       string
}

// Persist serialized the config and saves it to the config file
func (c *Config) Persist(overwrite bool) error {
	if _, err := os.Stat(c.ConfigPath()); err == nil && !overwrite {
		return ErrConfigFileExists{Path: c.ConfigPath()}
	}

	data, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}

	err = os.MkdirAll(c.dirpath, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(c.ConfigPath(), data, 0600)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Config is saved into file: %s\n", c.ConfigPath())

	return nil
}

// Load reads config file and returns the unmarshalled config structure
func (c *Config) Load() error {
	data, err := ioutil.ReadFile(c.ConfigPath())
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

// GetDeviceByName ...
func (c *Config) GetDeviceByName(name string) (*Device, error) {
	for i, device := range c.Devices {
		if device.Name == name {
			return c.Devices[i], nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Device not found: %s", name))
}

// GetDeviceByIP ...
func (c *Config) GetDeviceByIP(ip net.IP) (*Device, error) {
	for i, device := range c.Devices {
		if device.IP.String() == ip.String() {
			return c.Devices[i], nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Device not found: %s", ip.String()))
}

func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return filepath.Join(home, ConfigDir)
}

func (c *Config) ConfigPath() string {
	return filepath.Join(c.dirpath, ConfigFile)
}

func (c *Config) DBPath() string {
	return filepath.Join(c.dirpath, DBFile)
}

func (c *Config) DiscoverDBPath() string {
	return filepath.Join(c.dirpath, DiscoverDBFile)
}

func NewDefaultConfig() *Config {
	discoverIP := net.ParseIP(DefaultDiscoverIP)
	ip := net.ParseIP(DefaultIP)

	deviceIP := net.ParseIP(DefaultDeviceIP)
	device := Device{
		Name: DefaultDeviceName,
		IP:   &deviceIP,
		DeviceInventory: &DeviceInventory{
			CrateID: DefaultDeviceInventoryCrateID,
			SlotID:  DefaultDeviceInventorySlotID,
		},
	}

	return &Config{
		LogLevel:      DefaultLogLevel,
		DiscoverIP:    &discoverIP,
		DiscoverIface: DefaultDiscoverIface,
		IP:            &ip,
		Devices: []*Device{
			&device,
		},
		Inventory: &Inventory{
			Version:    DefaultInventoryVersion,
			DetectorID: DefaultInventoryDetectorID,
		},
		dirpath: DefaultConfigDir(),
	}
}
