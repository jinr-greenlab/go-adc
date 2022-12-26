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

// go-adc64 API
//
// # RESTful APIs to interact with go-adc64 server
//
// Terms Of Service:
//
// Schemes: http
// Host: localhost:8003
// Version: 1.0.0
// Contact:
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Security:
//	- api_key:
//
//	SecurityDefinitions:
//	api_key:
//	     type: apiKey
//	     name: KEY
//	     in: header
//
// swagger:meta
package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"jinr.ru/greenlab/go-adc/pkg/config"
	devicepkg "jinr.ru/greenlab/go-adc/pkg/device"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
	"jinr.ru/greenlab/go-adc/pkg/srv/control/ifc"
)

const (
	ApiPort = 8000
)

// Success response
// swagger:response okResp
type RespOk struct {
	// in:body
	Body struct {
		// HTTP status code 200 - OK
		Code int `json:"code"`
	}
} // Error Bad Request
// swagger:response badReq
type ReqBadRequest struct {
	// in:body
	Body struct {
		// HTTP status code 400 -  Bad Request
		Code int `json:"code"`
	}
}

// RegHex ...
type RegHex struct {
	Addr  string // hexadecimal
	Value string // hexadecimal
}

type TrigSetup struct {
	Timer     string `json:"timer"`
	Threshold string `json:"threshold"`
	Lemo      string `json:"lemo"`
}

type MAFSetup struct {
	Selector int
	BLC      int
}

type InvertSetup struct {
	Invert bool
}

type FirSetup struct {
	Coef     []uint16
	Roundoff uint16
}

type ReadoutWindowSetup struct {
	Size    uint16
	Latency uint16
}

type ZsSetup struct {
	Zs bool
}

type ApiServer struct {
	context.Context
	*config.Config
	*mux.Router
	ctrl ifc.ControlServer
}

var _ ifc.ApiServer = &ApiServer{}

func NewApiServer(ctx context.Context, cfg *config.Config, ctrl ifc.ControlServer) (ifc.ApiServer, error) {
	log.Info("Initializing API server with address: %s port: %d", cfg.IP, ApiPort)

	s := &ApiServer{
		Context: ctx,
		Config:  cfg,
		ctrl:    ctrl,
	}
	return s, nil
}

func (s *ApiServer) regReadHex(addr uint16, device string) (*RegHex, error) {
	d, err := s.ctrl.GetDeviceByName(device)
	if err != nil {
		return nil, err
	}
	reg, err := d.RegRead(addr)
	if err != nil {
		return nil, err
	}
	hexAddr, hexValue := reg.Hex()
	return &RegHex{
		Addr:  hexAddr,
		Value: hexValue,
	}, nil
}

func (s *ApiServer) regReadAllHex(device string) ([]*RegHex, error) {
	d, err := s.ctrl.GetDeviceByName(device)
	if err != nil {
		return nil, err
	}
	regs, err := d.RegReadAll()
	if err != nil {
		return nil, err
	}
	regsHex := []*RegHex{}
	for _, reg := range regs {
		hexAddr, hexValue := reg.Hex()
		regsHex = append(regsHex, &RegHex{Addr: hexAddr, Value: hexValue})
	}
	return regsHex, nil
}

// Start
func (s *ApiServer) Run() error {
	log.Info("Starting API server: address: %s port: %d", s.Config.IP, ApiPort)
	s.configureRouter()
	httpServer := &http.Server{
		Handler: s.Router,
		Addr:    fmt.Sprintf("%s:%d", s.Config.IP, ApiPort),
	}
	return httpServer.ListenAndServe()
}

func (s *ApiServer) configureRouter() {
	s.Router = mux.NewRouter()
	subRouter := s.Router.PathPrefix("/api").Subrouter()
	// swagger:operation GET /r/device/addr get register
	// ---
	// summary: read register
	// description: --
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/reg/r/{device}/{addr:0x[0-9abcdef]{4}}", s.handleRegRead()).Methods("GET")
	// swagger:operation GET /r/device read all registers
	// ---
	// summary: write register
	// description:
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/reg/r/{device}", s.handleRegReadAll()).Methods("GET")
	// swagger:operation POST /w/device/addr write register
	// ---
	// summary: write register
	// description:
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/reg/w/{device}", s.handleRegWrite()).Methods("POST")
	// swagger:operation GET /mstream/{action:start|stop}/device start/stop
	// ---
	// summary: start/stop acquisition for device
	// description:
	// responses:
	//   "200":mstream/{action:start|stop}/
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/mstream/{action:start|stop}/{device}", s.handleMStreamAction()).Methods("GET")
	// swagger:operation GET /mstream/{action:start|stop}
	// ---
	// summary: start/stop acquisition for all devices
	// description:
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/mstream/{action:start|stop}", s.handleMStreamActionAll()).Methods("GET")
	subRouter.HandleFunc("/trigger/{device}", s.handleTrigger()).Methods("POST")
	subRouter.HandleFunc("/maf/{device}", s.handleMAF()).Methods("POST")
	subRouter.HandleFunc("/invert_signal/{device}", s.handleInvert()).Methods("POST")
	subRouter.HandleFunc("/fir/{device}", s.handleFir()).Methods("POST")
	subRouter.HandleFunc("/readout_window/{device}", s.handleReadoutWindow()).Methods("POST")
	subRouter.HandleFunc("/channels/{device}", s.handleChannels()).Methods("POST")
	subRouter.HandleFunc("/zs/{device}", s.handleZs()).Methods("POST")
	s.Router.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(http.Dir("./swaggerui/"))))
}

func (s *ApiServer) handleRegRead() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		log.Debug("Handling reg read request: device: %s, addr: %s", vars["device"], vars["addr"])

		addr, err := strconv.ParseUint(vars["addr"], 0, 16)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		regHex, err := s.regReadHex(uint16(addr), vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(regHex)
	}
}

func (s *ApiServer) handleRegReadAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		log.Debug("Handling reg read all request: device: %s", vars["device"])

		regsHex, err := s.regReadAllHex(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(regsHex)
	}
}

func (s *ApiServer) handleRegWrite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		regHex := &RegHex{}
		err := json.NewDecoder(r.Body).Decode(regHex)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Debug("Handling reg write request: device: %s addr: %s value: %s",
			vars["device"], regHex.Addr, regHex.Value)

		reg, err := layers.NewRegFromHex(regHex.Addr, regHex.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		err = device.RegWrite(reg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

func (s *ApiServer) handleMStreamAction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		log.Debug("Handling MStream action request: device: %s action: %s", vars["device"], vars["action"])
		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		switch vars["action"] {
		case "start":
			err = device.MStreamStart()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		case "stop":
			err := device.MStreamStop()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		default:
			err := srv.ErrUnknownOperation{
				What: "Wrong MStream action. Must be one of start/stop",
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

func (s *ApiServer) handleMStreamActionAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		log.Debug("Handling MStream action request for all devices: action: %s", vars["action"])
		switch vars["action"] {
		case "start":
			for _, d := range s.ctrl.GetAllDevices() {
				err := d.MStreamStart()
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
			}
		case "stop":
			for _, d := range s.ctrl.GetAllDevices() {
				err := d.MStreamStop()
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
			}
		default:
			err := srv.ErrUnknownOperation{
				What: "Wrong MStream action. Must be one of start/stop",
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

func (s *ApiServer) handleTrigger() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		setup := &TrigSetup{}
		err := json.NewDecoder(r.Body).Decode(setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if setup.Timer != "" {
			val, _ := strconv.ParseBool(setup.Timer)
			err = device.SetTrigger(devicepkg.RegTrigStatusBitTimer, val)
		}
		if setup.Threshold != "" {
			val, _ := strconv.ParseBool(setup.Threshold)
			err = device.SetTrigger(devicepkg.RegTrigStatusBitThreshold, val)
		}
		if setup.Lemo != "" {
			val, _ := strconv.ParseBool(setup.Lemo)
			err = device.SetTrigger(devicepkg.RegTrigStatusBitLemo, val)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

func (s *ApiServer) handleMAF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		setup := &MAFSetup{}

		err := json.NewDecoder(r.Body).Decode(setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		err = device.SetMafSelector(setup.Selector)
		err = device.SetMafBlcThresh(setup.BLC)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

func (s *ApiServer) handleInvert() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		setup := &InvertSetup{}

		err := json.NewDecoder(r.Body).Decode(setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		err = device.SetInvert(setup.Invert)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

func (s *ApiServer) handleFir() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		setup := &FirSetup{}

		err := json.NewDecoder(r.Body).Decode(setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		err = device.SetRoundoff(setup.Roundoff)
		err = device.SetFirCoef(setup.Coef)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

func (s *ApiServer) handleReadoutWindow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		setup := &ReadoutWindowSetup{}

		err := json.NewDecoder(r.Body).Decode(setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		err = device.SetWindowSize(setup.Size)
		err = device.SetLatency(setup.Latency)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

func (s *ApiServer) handleChannels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		setup := &layers.ChannelsSetup{}

		err := json.NewDecoder(r.Body).Decode(setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		err = device.SetChannels(*setup)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

func (s *ApiServer) handleZs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		setup := &ZsSetup{}

		err := json.NewDecoder(r.Body).Decode(setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		device, err := s.ctrl.GetDeviceByName(vars["device"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		err = device.SetZs(setup.Zs)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}
