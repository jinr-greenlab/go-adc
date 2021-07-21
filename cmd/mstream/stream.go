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
	"fmt"
	"github.com/spf13/cobra"
	"jinr.ru/greenlab/go-adc/pkg/command"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/srv"
	"os"
)

func NewStreamCommand() *cobra.Command {
	var device string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	cmd := &cobra.Command{
		Use:           "stream",
		Short:         "Start MStream for device",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := command.NewApiClient(cfg)
			err := apiClient.MStream(srv.MStreamActionStart, device)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				return nil
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&device, DeviceOptionName, "", "Device name")

	return cmd
}
