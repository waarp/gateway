package admin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
)

const (
	// ServiceName is the nameof the administration interface service
	ServiceName = "Admin"

	// RestURI is the root path for the Rest API endpoints
	RestURI = "/api"
)

// Server is the administration service
type Server struct {
	Logger   *log.Logger
	Conf     *conf.ServerConfig
	Db       *database.Db
	Services map[string]service.Servicer

	state  service.State
	server http.Server
}

// listen starts the HTTP server listener on the configured port
func (s *Server) listen() {
	s.Logger.Infof("Listening at address %s", s.server.Addr)

	go func() {
		s.state.Set(service.Running, "")
		var err error
		if s.server.TLSConfig == nil {
			err = s.server.ListenAndServe()
		} else {
			err = s.server.ListenAndServeTLS("", "")
		}
		if err != http.ErrServerClosed {
			s.Logger.Errorf("Unexpected error: %s", err)
			s.state.Set(service.Error, err.Error())
		} else {
			s.state.Set(service.Offline, "")
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
	addr, err := checkAddress(s.Conf.Admin.Address)
	if err != nil {
		return err
	}

	// Load TLS configuration
	certFile := s.Conf.Admin.TLSCert
	keyFile := s.Conf.Admin.TLSKey
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
		s.Logger.Info("No TLS certificate configured, using plain HTTP.")
	}

	// Add the REST handler
	handler := mux.NewRouter()
	handler.Use(mux.CORSMethodMiddleware(handler), Authentication(s.Logger, s.Db))
	apiHandler := handler.PathPrefix(RestURI).Subrouter()
	apiHandler.HandleFunc(StatusURI, GetStatus(s.Services)).
		Methods(http.MethodGet)

	// Create http.Server instance
	s.server = http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
		ErrorLog:  s.Logger.AsStdLog(logging.ERROR),
	}
	return nil
}

// Start launches the administration service. If the service cannot be launched,
// the function returns an error.
func (s *Server) Start() error {
	if s.Logger == nil {
		s.Logger = log.NewLogger(ServiceName)
	}

	s.Logger.Info("Startup command received...")
	if state, _ := s.state.Get(); state != service.Offline && state != service.Error {
		s.Logger.Infof("Cannot start because the server is already running.")
		return nil
	}

	s.state.Set(service.Starting, "")

	if err := s.initServer(); err != nil {
		s.Logger.Errorf("Failed to start: %s", err)
		s.state.Set(service.Error, err.Error())
		return err
	}

	s.listen()

	s.Logger.Info("Server started.")
	return nil
}

// Stop halts the admin service by first trying to shut it down gracefully.
// If it fails, the service is forcefully stopped.
func (s *Server) Stop(ctx context.Context) error {
	s.Logger.Info("Shutdown command received...")
	if state, _ := s.state.Get(); state != service.Running {
		s.Logger.Info("Cannot stop because the server is not running.")
		return nil
	}

	s.state.Set(service.ShuttingDown, "")
	err := s.server.Shutdown(ctx)

	if err == nil {
		s.Logger.Info("Shutdown complete.")
	} else {
		s.Logger.Warningf("Failed to shutdown gracefully : %s", err)
		err = s.server.Close()
		s.Logger.Warning("The server was forcefully stopped.")
	}
	s.state.Set(service.Offline, "")
	return err
}

// State returns the state of the service
func (s *Server) State() *service.State {
	return &s.state
}
