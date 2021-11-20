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

package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"net/http"
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
	log.Debug("Initializing API server with address: %s port: %d", cfg.IP, ApiPort)

	s := &ApiServer{
		Context: ctx,
		Config: cfg,
		discover: discover,
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
	subRouter.HandleFunc("/devices", s.handleDevices()).Methods("GET")
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
