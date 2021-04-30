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
	"github.com/spf13/cobra"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/discover"
)

const (
	AddressOptionName = "address"
	PortOptionName = "port"
	IfaceNameOptionName = "iface-name"
)

func NewDiscoverCommand() *cobra.Command {
	var address, port, ifaceName string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	discoverConfig := cfg.DiscoverConfig
	cmd := &cobra.Command{
		Use:           "discover",
		Short:         "Start discover server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if address != "" {
				discoverConfig.Address = address
			}
			if port != "" {
				discoverConfig.Port = port
			}
			if ifaceName != "" {
				discoverConfig.Interface = ifaceName
			}
			server, err := discover.NewServer(discoverConfig)
			if err != nil {
				return err
			}
			return server.Run()
		},
	}
	cmd.Flags().StringVar(&address, AddressOptionName, "", fmt.Sprintf("Address to bind. E.g. %s", config.DefaultDiscoverAddress))
	cmd.Flags().StringVar(&port, PortOptionName, "", fmt.Sprintf("Port number to bind. E.g. %s", config.DefaultDiscoverPort))
	cmd.Flags().StringVar(&ifaceName, IfaceNameOptionName, "", "Interface name to listen on. E.g. eth0")

	return cmd
}
