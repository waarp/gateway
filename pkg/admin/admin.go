// Package admin contains the administration module which allows the user to
// manage the gateway via an HTTP interface.
package admin

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/lib/log"
	"go.step.sm/crypto/pemutil"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/names"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
)

var ErrMissingKeyFile = errors.New("missing certificate private key")

// Server is the administration service.
type Server struct {
	DB            *database.DB
	CoreServices  map[string]service.Service
	ProtoServices map[uint64]proto.Service

	logger *log.Logger
	state  state.State
	server http.Server
}

// listen starts the HTTP server listener on the configured port.
func listen(s *Server) {
	s.logger.Info("Listening at address %s", s.server.Addr)

	go func() {
		s.state.Set(state.Running, "")

		var err error

		if s.server.TLSConfig == nil {
			err = s.server.ListenAndServe()
		} else {
			err = s.server.ListenAndServeTLS("", "")
		}

		if !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("Unexpected error: %s", err)
			s.state.Set(state.Error, err.Error())
		} else {
			s.state.Set(state.Offline, "")
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

func (s *Server) makeTLSConfig() (*tls.Config, error) {
	certFile := conf.GlobalConfig.Admin.TLSCert
	keyFile := conf.GlobalConfig.Admin.TLSKey
	passphrase := conf.GlobalConfig.Admin.TLSPassphrase

	if keyFile == "" {
		return nil, ErrMissingKeyFile
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if passphrase == "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("could not parse TLS certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}

		return tlsConfig, nil
	}

	keyCryptPEM, err := os.ReadFile(filepath.Clean(keyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	keyBlock, _ := pem.Decode(keyCryptPEM)
	if keyBlock == nil {
		//nolint:goerr113 //this is a base error
		return nil, fmt.Errorf("key file does not contain a valid PEM block")
	}

	keyDER, err := pemutil.DecryptPEMBlock(keyBlock, []byte(passphrase))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	keyBlock.Bytes = keyDER
	keyBlock.Headers = nil
	keyPEM := pem.EncodeToMemory(keyBlock)

	certPEM, err := os.ReadFile(filepath.Clean(certFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("could not parse TLS certificate: %w", err)
	}

	tlsConfig.Certificates = []tls.Certificate{cert}

	return tlsConfig, nil
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

	var tlsConfig *tls.Config

	if conf.GlobalConfig.Admin.TLSCert != "" {
		// Load TLS configuration
		tlsConfig, err = serv.makeTLSConfig()
		if err != nil {
			serv.logger.Error("Failed to make TLS configuration: %s", err)

			return err
		}
	} else {
		serv.logger.Info("No TLS certificate configured, using plain HTTP.")
	}

	handler := MakeHandler(serv.logger, serv.DB, serv.CoreServices, serv.ProtoServices)

	// Create http.Server instance
	serv.server = http.Server{
		Addr:              addr,
		TLSConfig:         tlsConfig,
		Handler:           handler,
		ErrorLog:          serv.logger.AsStdLogger(log.LevelError),
		ReadHeaderTimeout: time.Second,
	}

	return nil
}

// Start launches the administration service. If the service cannot be launched,
// the function returns an error.
func (s *Server) Start() error {
	s.logger = conf.GetLogger(names.AdminServiceName)

	s.logger.Info("Startup command received...")

	if st, _ := s.state.Get(); st != state.Offline && st != state.Error {
		s.logger.Info("Cannot start because the server is already running.")

		return nil
	}

	s.state.Set(state.Starting, "")

	if err := initServer(s); err != nil {
		s.logger.Error("Failed to start: %s", err)
		s.state.Set(state.Error, err.Error())

		return err
	}

	listen(s)

	s.state.Set(state.Running, "")
	s.logger.Info("Server started")

	return nil
}

// Stop halts the admin service by first trying to shut it down gracefully.
// If it fails, the service is forcefully stopped.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutdown command received...")

	if st, _ := s.state.Get(); st != state.Running {
		s.logger.Info("Cannot stop because the server is not running")

		return nil
	}

	s.state.Set(state.ShuttingDown, "")
	err := s.server.Shutdown(ctx)

	if err == nil {
		s.logger.Info("Shutdown complete")
	} else {
		s.logger.Warning("Failed to shutdown gracefully : %s", err)
		err = s.server.Close()
		s.logger.Warning("The server was forcefully stopped")
	}

	s.state.Set(state.Offline, "")

	if err != nil {
		return fmt.Errorf("an error occurred while stopping the server: %w", err)
	}

	return nil
}

// State returns the state of the service.
func (s *Server) State() *state.State {
	return &s.state
}
