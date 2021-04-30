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

package mstream

import (
	"github.com/spf13/cobra"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/mstream"
)

const (
	AddressOptionName = "address"
	PortOptionName = "port"
)

func NewMStreamCommand() *cobra.Command {
	var address, port string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	mstreamConfig := cfg.MStreamConfig
	cmd := &cobra.Command{
		Use:           "mstream",
		Short:         "Start mstream server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if address != "" {
				mstreamConfig.Address = address
			}
			if port != "" {
				mstreamConfig.Port = port
			}

			server, err := mstream.NewServer(mstreamConfig)
			if err != nil {
				return err
			}
			return server.Run()
		},
	}
	cmd.Flags().StringVar(&address, AddressOptionName, "", "Address to bind. E.g. 192.168.1.2")
	cmd.Flags().StringVar(&port, PortOptionName, "", "Port number to bind. E.g. 33301")

	return cmd
}
