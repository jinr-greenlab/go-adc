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
	"fmt"
	"github.com/spf13/cobra"
	"jinr.ru/greenlab/go-adc/pkg/command"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"sort"
)

func NewReadCommand() *cobra.Command {
	var device, addr string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	cmd := &cobra.Command{
		Use:   "read",
		Short: "Read value from register",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := command.NewApiClient(cfg)
			if addr != "" {
				value, err := apiClient.RegRead(device, addr)
				if err != nil {
					return err
				}
				fmt.Printf("Register state: %s = %s\n", addr, value)
				return nil
			}
			regs, err := apiClient.RegReadAll(device)
			if err != nil {
				return err
			}
			var keys []string
			for key := range regs {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Printf("Register state: %s = %s\n", key, regs[key])
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&device, DeviceOptionName, "", "Device name")
	cmd.MarkFlagRequired(DeviceOptionName)
	cmd.Flags().StringVar(&addr, AddrOptionName, "", "Register address (hexadecimal)")

	return cmd
}
