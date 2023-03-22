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
//	Schemes: http
//	Host: localhost:8003
//	Version: 1.0.0
//	Contact:
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
package mstream

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	ApiPort = 8001
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

type Persist struct {
	Dir        string
	FilePrefix string
}

type ApiServer struct {
	context.Context
	*config.Config
	*mux.Router
	mstream *MStreamServer
}

func NewApiServer(ctx context.Context, cfg *config.Config, mstream *MStreamServer) (*ApiServer, error) {
	log.Info("Initializing API server with address: %s port: %d", cfg.IP, ApiPort)

	s := &ApiServer{
		Context: ctx,
		Config:  cfg,
		mstream: mstream,
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

func (s *ApiServer) configureRouter() {
	s.Router = mux.NewRouter()
	subRouter := s.Router.PathPrefix("/api").Subrouter()
	// swagger:operation POST /persist mstream getMstream
	// ---
	// summary: checks if mstream persist
	// description: --
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/persist", s.handlePersist()).Methods("POST")
	// swagger:operation GET /flush mstream getFlush
	// ---
	// summary: flush mstream
	// description: --
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/flush", s.handleFlush()).Methods("GET")
	// swagger:operation GET /last_event/{deviceName} mstream getLastEvent
	// ---
	// summary: last event mstream
	// description: --
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	subRouter.HandleFunc("/last_event/{deviceName}", s.handleLastEvent()).Methods("GET")
	// swagger:operation GET /connect_to_devices mstream getConnect
	// ---
	// summary: connects mstream to adc boards
	// description: --
	// responses:
	//   "200":
	//     "$ref": "#/responses/okResp"
	//   "400":
	//     "$ref": "#/responses/badReq"
	//subRouter.HandleFunc("/connect_to_devices", s.handleConnectToDevices()).Methods("GET")
	s.Router.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(http.Dir("./swaggerui/"))))
}

func (s *ApiServer) handlePersist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		persist := &Persist{}
		err := json.NewDecoder(r.Body).Decode(persist)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Debug("Handling persist request: filePrefix: %s", persist.FilePrefix)
		s.mstream.Persist(persist.Dir, persist.FilePrefix)
	}
}

func (s *ApiServer) handleFlush() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Handling flush request")
		s.mstream.Flush()
	}
}

func (s *ApiServer) handleLastEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceName := mux.Vars(r)["deviceName"]
		if ch, ok := s.mstream.lastEventChs[deviceName]; ok {
			select {
			case lastEvent := <-ch:
				s.mstream.mu.Lock()
				s.mstream.lastEvent[deviceName] = lastEvent
				s.mstream.mu.Unlock()
				w.Write(MstreamHeaderJson(lastEvent))
			default:
				s.mstream.mu.RLock()
				lastEvent := s.mstream.lastEvent[deviceName]
				s.mstream.mu.RUnlock()
				if len(lastEvent) != 0 {
					w.Write(MstreamHeaderJson(lastEvent))
				} else {
					http.Error(w, "no content", http.StatusNoContent)
				}
			}
		} else {
			http.Error(w, "device not found", http.StatusNotFound)
		}
	}
}

//func (s *ApiServer) handleConnectToDevices() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		log.Debug("Handling connect to devices request")
//		err := s.mstream.ConnectToDevices()
//		if err != nil {
//			http.Error(w, err.Error(), http.StatusBadGateway)
//		}
//	}
//}
