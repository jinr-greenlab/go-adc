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

package reg

import (
	"github.com/spf13/cobra"
	"jinr.ru/greenlab/go-adc/pkg/command"
	"jinr.ru/greenlab/go-adc/pkg/config"
)

func NewWriteCommand() *cobra.Command {
	var device, addr, value string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	cmd := &cobra.Command{
		Use:   "write",
		Short: "Write value to register",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := command.NewApiClient(cfg)
			err := apiClient.RegWrite(device, addr, value)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&device, DeviceOptionName, "", "Device name")
	cmd.MarkFlagRequired(DeviceOptionName)
	cmd.Flags().StringVar(&addr, AddrOptionName, "", "Register address (hexadecimal)")
	cmd.MarkFlagRequired(AddrOptionName)
	cmd.Flags().StringVar(&value, ValueOptionName, "", "Register value (hexadecimal)")
	cmd.MarkFlagRequired(ValueOptionName)

	return cmd
}
