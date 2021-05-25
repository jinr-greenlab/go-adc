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
	IP *net.IP `json:"ip,omitempty"`
	Devices []*Device `json:"devices"`
	filepath string
}

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

	return nil
}

func (c *Config) Load() error {
	data, err := ioutil.ReadFile(c.filepath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return filepath.Join(home, ConfigDir, ConfigFile)
}

func NewDefaultConfig() *Config {
	discoverIP := net.ParseIP(DefaultDiscoverIP)
	ip := net.ParseIP(DefaultIP)

	return &Config{
		DiscoverIP: &discoverIP,
		DiscoverIface: DefaultDiscoverIface,
		IP: &ip,
		Devices: []*Device{},
		filepath: DefaultConfigPath(),
	}
}


