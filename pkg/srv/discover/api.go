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
// RESTful APIs to interact with go-adc64 server
//
// Terms Of Service:
//
//     Schemes: http
//     Host: localhost:8003
//     Version: 1.0.0
//     Contact:
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Security:
//     - api_key:
//
//     SecurityDefinitions:
//     api_key:
//          type: apiKey
//          name: KEY
//          in: header
//
// swagger:meta
package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	ApiPort = 8003
)

type ApiServer struct {
	context.Context
	*config.Config
	*mux.Router
	discover *DiscoverServer
}

func NewApiServer(ctx context.Context, cfg *config.Config, discover *DiscoverServer) (*ApiServer, error) {
	log.Info("Initializing API server with address: %s port: %d", cfg.IP, ApiPort)

	s := &ApiServer{
		Context:  ctx,
		Config:   cfg,
		discover: discover,
	}
	return s, nil
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

func (s *ApiServer) configureRouter() {
	s.Router = mux.NewRouter()
	subRouter := s.Router.PathPrefix("/api").Subrouter()
	// swagger:operation GET /devices devices getDevices
	// ---
	// summary: Return a list of discovered devices
	// description: If the list exists, it will be returned else null will be returned.
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/devices", s.handleDevices()).Methods("GET")
	s.Router.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(http.Dir("./swaggerui/"))))
}

func (s *ApiServer) handleDevices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Handling devices request")
		devices, err := s.discover.state.GetAllDeviceDescriptions()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		json.NewEncoder(w).Encode(devices)
	}
}
