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
	"jinr.ru/greenlab/go-adc/pkg/command"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"time"
)

func NewListCommand() *cobra.Command {
	cfg := config.NewDefaultConfig()
	cfg.Load()
	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List discovered devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := command.NewApiClient(cfg)
			devices, err := apiClient.ListDevices()
			if err != nil {
				return err
			}
			for _, device := range devices {
				fmt.Printf(device.String())
				now := uint64(time.Now().UnixNano()) * uint64(time.Nanosecond) / uint64(time.Millisecond)
				if now - device.Timestamp > 3000 {
					fmt.Printf("!!! Device is offline")
				}
			}
			return nil
		},
	}
	return cmd
}
