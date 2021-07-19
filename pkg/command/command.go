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
	"errors"
	"fmt"
	"github.com/imroc/req"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/srv"
)

type ApiClient struct {
	*config.Config
	ApiPrefix string
}

func NewApiClient(cfg *config.Config) *ApiClient {
	return &ApiClient{
		Config: cfg,
		ApiPrefix: fmt.Sprintf("http://%s:%d/api", cfg.IP, srv.ApiPort),
	}
}

func (c *ApiClient) regGetUrl(device, regnum string) string {
	return fmt.Sprintf("%s/reg/get/%s/%s", c.ApiPrefix, device, regnum)
}

func (c *ApiClient) regSetUrl(device string) string {
	return fmt.Sprintf("%s/reg/set/%s", c.ApiPrefix, device)
}

// RegGet low level api request to get the value of a register for a device
func (c *ApiClient) RegGet(device, regnum string) (string, error) {
	r, err := req.Get(c.regGetUrl(device, regnum))
	if err != nil {
		return "", err
	}

	if r.Response().StatusCode != 200 {
		return "", errors.New(r.Response().Status)
	}

	reg := &srv.RegHex{}
	err = r.ToJSON(reg)
	if err != nil {
		return "", err
	}
	return reg.RegValue, nil
}

// RegSet low level api request to set the value of a register to a value for a device
func (c *ApiClient) RegSet(device, regnum, regval string) error {
	reg := &srv.RegHex{
		RegNum: regnum,
		RegValue: regval,
	}
	r, err := req.Post(c.regSetUrl(device), req.BodyJSON(reg))
	if err != nil {
		return err
	}

	if ! (r.Response().StatusCode != 200) {
		return errors.New(r.Response().Status)
	}
	return nil
}
