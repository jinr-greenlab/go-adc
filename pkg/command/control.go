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

package command

import (
	"context"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/srv/control"
)

// StartControlServer ...
func StartControlServer(cfg *config.Config) error {
	ctx := context.Background()

	s, err := control.NewControlServer(ctx, cfg)
	if err != nil {
		return err
	}
	return s.Run()
}