// Package admin contains the administration module which allows the user to
// manage the gateway via an HTTP interface.
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
)

const (
	// ServiceName is the name of the administration interface service
	ServiceName = "Admin"
)

// Server is the administration service
type Server struct {
	DB       *database.DB
	Services map[string]service.Service

	logger *log.Logger
	state  service.State
	server http.Server
}

// listen starts the HTTP server listener on the configured port
func listen(s *Server) {
	s.logger.Infof("Listening at address %s", s.server.Addr)

	go func() {
		s.state.Set(service.Running, "")
		var err error
		if s.server.TLSConfig == nil {
			err = s.server.ListenAndServe()
		} else {
			err = s.server.ListenAndServeTLS("", "")
		}
		if err != http.ErrServerClosed {
			s.logger.Errorf("Unexpected error: %s", err)
			s.state.Set(service.Error, err.Error())
		} else {
			s.state.Set(service.Offline, "")
		}
	}()

}

// checkAddress checks if the address given in the configuration is a
// valid address on which the server can listen
func checkAddress() (string, error) {
	config := &conf.GlobalConfig.ServerConf.Admin
	addr := net.JoinHostPort(config.Host, fmt.Sprint(config.Port))
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
func initServer(s *Server) error {
	// Load REST s address
	addr, err := checkAddress()
	if err != nil {
		return err
	}

	// Load TLS configuration
	config := &conf.GlobalConfig.ServerConf.Admin
	var tlsConfig *tls.Config
	if config.TLSCert != "" && config.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
		if err != nil {
			return fmt.Errorf("could not load REST certificate (%s)", err)
		}
		tlsConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
	} else {
		s.logger.Info("No TLS certificate configured, using plain HTTP.")
	}

	handler := MakeHandler(s.logger, s.DB, s.Services)

	// Create http.Server instance
	s.server = http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
		ErrorLog:  s.logger.AsStdLog(logging.ERROR),
	}
	return nil
}

// Start launches the administration service. If the service cannot be launched,
// the function returns an error.
func (s *Server) Start() error {
	s.logger = log.NewLogger(ServiceName)

	s.logger.Info("Startup command received...")
	if state, _ := s.state.Get(); state != service.Offline && state != service.Error {
		s.logger.Infof("Cannot start because the server is already running.")
		return nil
	}

	s.state.Set(service.Starting, "")

	if err := initServer(s); err != nil {
		s.logger.Errorf("Failed to start: %s", err)
		s.state.Set(service.Error, err.Error())
		return err
	}

	listen(s)

	s.state.Set(service.Running, "")
	s.logger.Info("Server started")
	return nil
}

// Stop halts the admin service by first trying to shut it down gracefully.
// If it fails, the service is forcefully stopped.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutdown command received...")
	if state, _ := s.state.Get(); state != service.Running {
		s.logger.Info("Cannot stop because the server is not running")
		return nil
	}

	s.state.Set(service.ShuttingDown, "")
	err := s.server.Shutdown(ctx)

	if err == nil {
		s.logger.Info("Shutdown complete")
	} else {
		s.logger.Warningf("Failed to shutdown gracefully : %s", err)
		err = s.server.Close()
		s.logger.Warning("The server was forcefully stopped")
	}
	s.state.Set(service.Offline, "")
	return err
}

// State returns the state of the service
func (s *Server) State() *service.State {
	return &s.state
}
