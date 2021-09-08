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
	"fmt"
	"github.com/spf13/cobra"
	"jinr.ru/greenlab/go-adc/pkg/command"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"net"
)

const (
	IPOptionName = "ip"
)

func NewStartCommand() *cobra.Command {
	var ip string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	cmd := &cobra.Command{
		Use:           "start",
		Short:         "Start control server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ip != "" {
				parsedIP := net.ParseIP(ip)
				cfg.IP = &parsedIP
			}
			return command.StartControlServer(cfg)
		},
	}
	cmd.Flags().StringVar(&ip, IPOptionName, "", fmt.Sprintf("IP to bind. E.g. %s", config.DefaultIP))

	return cmd
}
