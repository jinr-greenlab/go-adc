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

func NewSetCommand() *cobra.Command {
	var device, regNum, regValue string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	cmd := &cobra.Command{
		Use:           "set",
		Short:         "Set reg value",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := command.NewApiClient(cfg)
			apiClient.RegSet(device, regNum, regValue)
			return nil
		},
	}
	cmd.Flags().StringVar(&device, DeviceOptionName, "", "Device name")
	cmd.Flags().StringVar(&regNum, RegNumOptionName, "", "Register address (hexadecimal)")
	cmd.Flags().StringVar(&regValue, RegValueOptionName, "", "Register value (hexadecimal)")

	return cmd
}
