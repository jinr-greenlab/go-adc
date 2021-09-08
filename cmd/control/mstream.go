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

package control

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"jinr.ru/greenlab/go-adc/pkg/command"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/srv/control"
)

func NewMStreamCommand() *cobra.Command {
	var device string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	cmd := &cobra.Command{
		Use:    "mstream",
		Short:  "Start/stop streaming for a device",
		Args:   cobra.ExactArgs(1),
		ValidArgs: []string{control.ActionStart, control.ActionStop},
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := command.NewApiClient(cfg)
			switch args[0] {
			case control.ActionStart:
				if device != "" {
					return apiClient.MStreamStart(device)
				}
				return apiClient.MStreamStartAll()
			case control.ActionStop:
				if device != "" {
					return apiClient.MStreamStop(device)
				}
				return apiClient.MStreamStopAll()
			default:
				return errors.New(
					fmt.Sprintf("Wrong streaming command. Must be one of %s/%s", control.ActionStart, control.ActionStop))
			}
		},
	}
	cmd.Flags().StringVar(&device, DeviceOptionName, "", "Device name")

	return cmd
}
