package srv

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"net"
	"net/http"
	"strconv"
)

type RegHex struct {
	RegNum string // hexadecimal
	RegValue string // hexadecimal
}

type Reg struct {
	RegNum uint16
	RegValue uint16
}

// Start
func (s *RegServer) StartApiServer() error {
	log.Debug("Starting API server: address: %s port: %d", s.Config.IP, ApiPort)
	s.configureRouter()
	httpServer := &http.Server{
		Handler: s.Router,
		Addr:    fmt.Sprintf("%s:%d", s.Config.IP, ApiPort),
	}
	return httpServer.ListenAndServe()
}

func (s *RegServer) configureRouter() {
	s.Router = mux.NewRouter()
	subRouter := s.Router.PathPrefix("/api").Subrouter()
	// regnum and regval must be hexadecimal integers
	subRouter.HandleFunc("/reg/get/{device}/{regnum:0x[0-9abcdef]{4}}", s.handleRegGet()).Methods("GET")
	subRouter.HandleFunc("/reg/set/{device}", s.handleRegSet()).Methods("POST")
}

func (s *RegServer) handleRegGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		log.Debug("Handling RegGet request: device: %s, regNum: %s", vars["device"], vars["regnum"])

		dev := s.Config.GetDeviceByName(vars["device"])
		if dev == nil {
			http.Error(w, fmt.Sprintf("Device %s not found", vars["device"]), http.StatusNotFound)
			return
		}

		parsedRegNum, err := strconv.ParseUint(vars["regnum"], 0, 16)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		regValue, err := s.GetRegState(uint16(parsedRegNum))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(&RegHex{
			RegNum: fmt.Sprintf("%x", uint16(parsedRegNum)),
			RegValue: fmt.Sprintf("%x", regValue),
		})
	}
}

func (s *RegServer) handleRegSet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		log.Debug("Handling RegSet request: device: %s", vars["device"])

		dev := s.Config.GetDeviceByName(vars["device"])
		if dev == nil {
			http.Error(w, fmt.Sprintf("Device %s not found", vars["device"]), http.StatusNotFound)
			return
		}

		deviceUdpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dev.IP, RegPort))
		if err != nil {
			http.Error(w, fmt.Sprintf("Can not resolve device address: %s", vars["device"]), http.StatusBadRequest)
			return
		}

		regHex := &RegHex{}
		err = json.NewDecoder(r.Body).Decode(regHex)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		parsedRegNum, err := strconv.ParseUint(regHex.RegNum, 0, 16)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		parsedRegValue, err := strconv.ParseUint(regHex.RegValue, 0, 16)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		regOps := []*layers.RegOp{
			{
				Read: false,
				RegNum: uint16(parsedRegNum),
				RegValue: uint16(parsedRegValue),
			},
		}

		err = s.RegRequest(regOps, deviceUdpAddr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}
