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

package cmd

import (
	"fmt"
	"github.com/imroc/req"
	"jinr.ru/greenlab/go-adc/pkg/srv"

	"jinr.ru/greenlab/go-adc/pkg/config"
)

type Client struct {
	*config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		Config: cfg,
	}
}

func (c *Client) RegGet(device, regnum string) {
	req.Get(fmt.Sprintf("http://%s:%s/api/regget/%s/%s", c.Config.IP, srv.ApiPort, device, regnum))
}
