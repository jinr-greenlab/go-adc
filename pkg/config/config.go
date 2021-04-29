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
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type DiscoverConfig struct {
	Address string `json:"address,omitempty"`
	Port string `json:"port,omitempty"`
	Interface string `json:"interface"`
}

type MStreamPeer struct {
	Address string `json:"address,omitempty"`
	Port string `json:"port,omitempty"`
}

type MStreamConfig struct {
	Address string `json:"address,omitempty"`
	Port string `json:"port,omitempty"`
	Peers []*MStreamPeer `json:"peers"`
}

type Config struct {
	*DiscoverConfig `json:"discover,omitempty"`
	*MStreamConfig `json:"mstream,omitempty"`
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

func (c *Config) LoadConfig() error {
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
	return &Config{
		DiscoverConfig: &DiscoverConfig{
			Address: DefaultDiscoverAddress,
			Port: DefaultDiscoverPort,
			Interface: DefaultDiscoverInterface,
		},
		MStreamConfig: &MStreamConfig{
			Address: DefaultMStreamAddress,
			Port: DefaultMStreamPort,
			Peers: []*MStreamPeer{
				{
					Address: DefaultMStreamPeerAddress,
					Port: DefaultMStreamPeerPort,
				},
			},
		},
		filepath: DefaultConfigPath(),
	}
}


