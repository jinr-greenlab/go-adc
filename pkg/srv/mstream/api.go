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

type Persist struct {
	Dir string
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
		Config: cfg,
		mstream: mstream,
	}
	return s, nil
}

// Start
func (s *ApiServer) Run() error {
	log.Debug("Starting API server: address: %s port: %d", s.Config.IP, ApiPort)
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
	subRouter.HandleFunc("/persist", s.handlePersist()).Methods("POST")
	subRouter.HandleFunc("/flush", s.handleFlush()).Methods("GET")
	subRouter.HandleFunc("/connect_to_devices", s.handleConnectToDevices()).Methods("GET")
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

		err = s.mstream.EventHandler.Persist(persist.Dir, persist.FilePrefix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

func (s *ApiServer) handleFlush() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Handling flush request")
		s.mstream.EventHandler.Flush()
	}
}

func (s *ApiServer) handleConnectToDevices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Handling connect to devices request")
		err := s.mstream.ConnectToDevices()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
	}
}
