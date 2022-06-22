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
)

func NewMStreamCommand() *cobra.Command {
	var filePrefix string
	var dir string
	cfg := config.NewDefaultConfig()
	cfg.Load()
	cmd := &cobra.Command{
		Use:       fmt.Sprintf("mstream start|stop"),
		Short:     "Start/stop streaming for a device",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"start", "stop"},
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := command.NewApiClient(cfg)
			switch args[0] {
			case "start":
				err := apiClient.MStreamConnectToDevices()
				if err != nil {
					return err
				}
				err = apiClient.MStreamPersist(dir, filePrefix)
				if err != nil {
					return err
				}
				return apiClient.MStreamStartAll()
			case "stop":
				err := apiClient.MStreamStopAll()
				if err != nil {
					return err
				}
				return apiClient.MStreamFlush()
			default:
				return errors.New("Wrong streaming command. Must be one of start/stop")
			}
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Directory path where to persist data")
	cmd.Flags().StringVar(&filePrefix, "file-prefix", "", "File name prefix")

	return cmd
}
