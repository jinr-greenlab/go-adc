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

type Device struct {
	Name string `json:"name,omitempty"`
	IP *net.IP `json:"ip,omitempty"`
}

type Config struct {
	DiscoverIP *net.IP `json:"discoverIP,omitempty"`
	DiscoverIface string `json:"discoverIface,omitempty"`
	DiscoverDBPath string `json:"discoverDBPath"`
	IP *net.IP `json:"ip,omitempty"`
	Devices []*Device `json:"devices"`
	DBPath string `json:"dbpath,omitempty"`
	filepath string
}

// Persist serialized the config and saves it to the config file
func (c *Config) Persist(overwrite bool) error {
	if _, err := os.Stat(c.filepath); err == nil && !overwrite {
		return ErrConfigFileExists{Path: c.filepath}
	}

	data, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.filepath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(c.filepath, data, 0644)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Config is saved into file: %s\n", c.filepath)

	return nil
}

// Load reads config file and returns the unmarshalled config structure
func (c *Config) Load() error {
	data, err := ioutil.ReadFile(c.filepath)
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

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return filepath.Join(home, ConfigDir, ConfigFile)
}

func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return filepath.Join(home, ConfigDir, DBFile)
}

func NewDefaultConfig() *Config {
	discoverIP := net.ParseIP(DefaultDiscoverIP)
	ip := net.ParseIP(DefaultIP)

	return &Config{
		DiscoverIP: &discoverIP,
		DiscoverIface: DefaultDiscoverIface,
		IP: &ip,
		Devices: []*Device{},
		DBPath: DefaultDBPath(),
		filepath: DefaultConfigPath(),
	}
}


