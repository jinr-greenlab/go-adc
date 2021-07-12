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
	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/srv"
	"net"
	"time"
)

// This package is temporary. It is needed only during development for some intermedite testing.

type RegRW struct {
	*srv.RegServer
	*config.Config
}

func NewRegRW(cfg *config.Config) (*RegRW, error) {
	regServer, err := srv.NewRegServer(cfg)
	if err != nil {
		return nil, err
	}
	return &RegRW{
		RegServer: regServer,
		Config: cfg,
	}, nil
}


func (regrw *RegRW) RegRead(regNum uint16, deviceIP string) error {
	deviceUdpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", deviceIP, srv.RegPort))
	if err != nil {
		return err
	}
	go regrw.RegServer.Run()
	time.Sleep(1000 * time.Millisecond)
	regOps := []*layers.RegOp{
		{
			Read: true,
			RegNum: regNum,
		},
	}
	regrw.RegServer.RegRequest(regOps, deviceUdpAddr)
	time.Sleep(1000 * time.Millisecond)
	regrw.RegServer.GetRegState(regNum)
	time.Sleep(1000 * time.Millisecond)
	return nil
}

func (regrw *RegRW) RegWrite(regNum, regValue uint16, deviceIP string) error {
	deviceUdpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", deviceIP, srv.RegPort))
	if err != nil {
		return err
	}
	go regrw.RegServer.Run()
	time.Sleep(1000 * time.Millisecond)
	regOps := []*layers.RegOp{
		{
			Read: false,
			RegNum: regNum,
			RegValue: regValue,
		},
	}
	regrw.RegServer.RegRequest(regOps, deviceUdpAddr)
	time.Sleep(1000 * time.Millisecond)
	regrw.RegServer.GetRegState(regNum)
	time.Sleep(1000 * time.Millisecond)
	return nil
}


func (regrw *RegRW) StartMStream() error {
	go regrw.RegServer.Run()
	time.Sleep(1000 * time.Millisecond)
	regrw.RegServer.StartMStream()
	time.Sleep(1000 * time.Millisecond)
	regrw.RegServer.GetRegState(srv.CtrlReg)
	time.Sleep(1000 * time.Millisecond)
	return nil
}

func (regrw *RegRW) StopMStream() error {
	go regrw.RegServer.Run()
	time.Sleep(1000 * time.Millisecond)
	regrw.RegServer.StopMStream()
	time.Sleep(1000 * time.Millisecond)
	regrw.RegServer.GetRegState(srv.CtrlReg)
	time.Sleep(1000 * time.Millisecond)
	return nil
}

