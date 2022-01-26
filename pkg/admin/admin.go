// Package admin contains the administration module which allows the user to
// manage the gateway via an HTTP interface.
package admin

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"

	"code.bcarlin.xyz/go/logging"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// Server is the administration service.
type Server struct {
	DB            *database.DB
	CoreServices  map[string]service.Service
	ProtoServices map[string]service.ProtoService

	logger *log.Logger
	state  service.State
	server http.Server
}

// listen starts the HTTP server listener on the configured port.
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

		if !errors.Is(err, http.ErrServerClosed) {
			s.logger.Errorf("Unexpected error: %s", err)
			s.state.Set(service.Error, err.Error())
		} else {
			s.state.Set(service.Offline, "")
		}
	}()
}

// checkAddress checks if the address given in the configuration is a
// valid address on which the server can listen.
func checkAddress() (string, error) {
	config := &conf.GlobalConfig.Admin
	addr := net.JoinHostPort(config.Host, fmt.Sprint(config.Port))

	addr, err := conf.GetRealAddress(addr)
	if err != nil {
		return "", fmt.Errorf("failed to indirect the admin address: %w", err)
	}

	l, err := net.Listen("tcp", addr)
	if err == nil {
		defer l.Close() //nolint:errcheck // nothing to handle the error

		return l.Addr().String(), nil
	}

	return "", fmt.Errorf("canot open listener: %w", err)
}

// initServer initializes the HTTP server instance using the parameters defined
// in the Admin configuration.
// If the configuration is invalid, this function returns an error.
func initServer(serv *Server) error {
	// Load REST s address
	addr, err := checkAddress()
	if err != nil {
		return err
	}

	// Load TLS configuration
	config := &conf.GlobalConfig.Admin

	var tlsConfig *tls.Config

	if config.TLSCert != "" && config.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
		if err != nil {
			return fmt.Errorf("could not load REST certificate: %w", err)
		}

		tlsConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
	} else {
		serv.logger.Info("No TLS certificate configured, using plain HTTP.")
	}

	handler := MakeHandler(serv.logger, serv.DB, serv.CoreServices, serv.ProtoServices)

	// Create http.Server instance
	serv.server = http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
		ErrorLog:  serv.logger.AsStdLog(logging.ERROR),
	}

	return nil
}

// Start launches the administration service. If the service cannot be launched,
// the function returns an error.
func (s *Server) Start() error {
	s.logger = log.NewLogger(service.AdminServiceName)

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

	if err != nil {
		return fmt.Errorf("an error occurred while stopping the server: %w", err)
	}

	return nil
}

// State returns the state of the service.
func (s *Server) State() *service.State {
	return &s.state
}
