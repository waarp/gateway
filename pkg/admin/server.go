package admin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp/gateway-ng/pkg/gatewayd"
	"code.waarp.fr/waarp/gateway-ng/pkg/tk/service"
	"github.com/gorilla/mux"
)

const apiURI = "/api"

// Server is the administration service
type Server struct {
	*gatewayd.WG

	state   service.State
	server  http.Server
}

// listen starts the HTTP server listener on the configured port
func (s *Server) listen() {
	s.Logger.Admin.Infof("Listening at address %s", s.server.Addr)
	var err error
	s.state.Set(service.RUNNING, "")

	go func() {
		if s.server.TLSConfig == nil {
			err = s.server.ListenAndServe()
		} else {
			err = s.server.ListenAndServeTLS("", "")
		}
		if err != http.ErrServerClosed {
			s.Logger.Admin.Errorf("Unexpected error: %s", err)
			s.state.Set(service.ERROR, err.Error())
		} else {
			s.state.Set(service.DOWN, "")
		}
	}()

}

// checkAddress checks if the address given in the configuration is a
// valid address on which the server can listen
func checkAddress(addr string) (string, error) {
	l, err := net.Listen("tcp", addr)
	if err == nil {
		defer l.Close()
		return l.Addr().String(), nil
	}
	return "", err
}

// initServer initializes the HTTP server instance using the parameters defined
// in the Admin configuration.
// If the configuration is invalid, this function returns an error.
func (s *Server) initServer() error {
	// Load REST s address
	addr, err := checkAddress(s.Config.Admin.Address)
	if err != nil {
		return err
	}

	// Load TLS configuration
	certFile := s.Config.Admin.TLSCert
	keyFile := s.Config.Admin.TLSKey
	var tlsConfig *tls.Config
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("could not load REST certificate (%s)", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	} else {
		s.Logger.Admin.Info("No TLS certificate found, using plain HTTP.")
	}

	// Add the REST handler
	handler := mux.NewRouter()
	handler.Use(mux.CORSMethodMiddleware(handler), Authentication(s.Logger))
	apiHandler := handler.PathPrefix(apiURI).Subrouter()
	apiHandler.HandleFunc(statusURI, GetStatus).
		Methods(http.MethodGet)

	// Create http.Server instance
	s.server = http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
		ErrorLog:  s.Logger.Admin.AsStdLog(logging.ERROR),
	}
	return nil
}

// Start launches the administration service. If the service cannot be launched,
// the function returns an error.
func (s *Server) Start() error {
	if s.WG == nil {
		return fmt.Errorf("missing application configuration")
	}

	s.Logger.Admin.Info("Startup command received...")
	if state, _ := s.state.Get(); state == service.RUNNING {
		s.Logger.Admin.Info("Cannot start because the server is already running.")
		return nil
	}

	if err := s.initServer(); err != nil {
		s.Logger.Admin.Errorf("Failed to start: %s", err)
		return err
	}

	s.listen()

	s.Logger.Admin.Info("Server started.")
	return nil
}

// Stop halts the admin service by first trying to shut it down gracefully.
// If it fails, the service is forcefully stopped.
func (s *Server) Stop(ctx context.Context) error {
	s.Logger.Admin.Info("Shutdown command received...")
	if state, _ := s.state.Get(); state != service.RUNNING {
		s.Logger.Admin.Info("Cannot stop because the server is not running.")
		return nil
	}

	err := s.server.Shutdown(ctx)

	if err == nil {
		s.Logger.Admin.Info("Shutdown complete.")
	} else {
		s.Logger.Admin.Warningf("Failed to shutdown gracefully : %s", err)
		err = s.server.Close()
		s.Logger.Admin.Warning("The server was forcefully stopped.")
	}
	return err
}
