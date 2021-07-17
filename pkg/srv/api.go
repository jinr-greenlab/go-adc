package srv

import (
	"fmt"
	"github.com/gorilla/mux"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"net"
	"net/http"
	"strconv"
	"time"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

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
	subRouter.HandleFunc("/regget/{device}/{regnum:0x[0-9abcdef]{4}}", s.handleRegGet())
}

func (s *RegServer) handleRegGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//vars := mux.Vars(r)
		//w.Write([]byte(fmt.Sprintf("Hello from RegGet: device: %s regnum: %s\n", vars["device"], vars["regnum"])))

		vars := mux.Vars(r)
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

		parsedRegnum, err := strconv.ParseUint(vars["regnum"], 0, 16)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		regnum := uint16(parsedRegnum)

		regOps := []*layers.RegOp{
			{
				Read: true,
				RegNum: regnum,
			},
		}
		err = s.RegRequest(regOps, deviceUdpAddr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		time.Sleep(1000 * time.Millisecond)
		s.GetRegState(regnum)

		w.Write([]byte(fmt.Sprintf("Hello from Reg Get: regnum: %x\n", regnum)))
	}
}
