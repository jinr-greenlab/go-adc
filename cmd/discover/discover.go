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
	"github.com/spf13/cobra"
	"jinr.ru/greenlab/go-adc/pkg/discover"
)

const (
	AddressOptionName = "address"
	PortOptionName = "port"
	IfaceNameOptionName = "iface-name"
)

func NewDiscoverCommand() *cobra.Command {
	var address, port, ifaceName string
	cmd := &cobra.Command{
		Use:           "discover",
		Short:         "Start discover server",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := discover.NewServer(address, port, ifaceName)
			if err != nil {
				return err
			}
			return server.Run()
		},
	}
	cmd.Flags().StringVar(&address, AddressOptionName, "", "Address to bind. E.g. 239.192.1.1")
	cmd.Flags().StringVar(&port, PortOptionName, "", "Port number to bind. E.g. 33303")
	cmd.Flags().StringVar(&ifaceName, IfaceNameOptionName, "", "Interface name to listen on. E.g. eth0")

	return cmd
}
