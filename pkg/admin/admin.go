// Package admin contains the administration module which allows the user to
// manage the gateway via an HTTP interface.
package admin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const ServiceName = "Admin"

var ErrMissingKeyFile = errors.New("missing certificate private key")

// Server is the administration service.
type Server struct {
	DB *database.DB

	state  utils.State
	logger *log.Logger
	server http.Server
}

// listen starts the HTTP server listener on the configured port.
func (s *Server) listen() error {
	list, lErr := net.Listen("tcp", s.server.Addr)
	if lErr != nil {
		return fmt.Errorf("failed to open listener: %w", lErr)
	}

	s.logger.Infof("Listening at address %q", list.Addr().String())

	go func() {
		var err error

		if s.server.TLSConfig == nil {
			err = s.server.Serve(list)
		} else {
			err = s.server.ServeTLS(list, "", "")
		}

		if !errors.Is(err, http.ErrServerClosed) {
			s.logger.Errorf("Unexpected error: %v", err)
		}
	}()

	return nil
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
		//nolint:err113 //this is a base error
		return nil, errors.New("key file does not contain a valid PEM block")
	}

	//nolint:staticcheck //this is needed for decrypting the key, even if the encryption is insecure
	keyDER, err := x509.DecryptPEMBlock(keyBlock, []byte(passphrase))
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
	config := &conf.GlobalConfig.Admin
	addr := conf.GetRealAddress(config.Host, utils.FormatUint(config.Port))

	var tlsConfig *tls.Config

	if conf.GlobalConfig.Admin.TLSCert != "" {
		var err error
		if tlsConfig, err = serv.makeTLSConfig(); err != nil {
			serv.logger.Errorf("Failed to make TLS configuration: %v", err)

			return err
		}
	} else {
		serv.logger.Info("No TLS certificate configured, using plain HTTP.")
	}

	handler := MakeHandler(serv.logger, serv.DB)

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
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(ServiceName)
	s.logger.Info("Starting administration service...")

	if err := initServer(s); err != nil {
		s.logger.Errorf("Failed to initialize server: %v", err)
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(ServiceName, err)

		return err
	}

	if err := s.listen(); err != nil {
		s.logger.Errorf("Failed to start listener: %v", err)
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(ServiceName, err)

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Info("Server started")

	return nil
}

// Stop halts the admin service by first trying to shut it down gracefully.
// If it fails, the service is forcefully stopped.
func (s *Server) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Shutdown command received...")

	err := s.server.Shutdown(ctx)
	if err == nil {
		s.logger.Info("Shutdown complete")
	} else {
		s.logger.Warningf("Failed to shutdown gracefully : %v", err)

		if err2 := s.server.Close(); err2 != nil {
			s.logger.Errorf("Failed to force shutdown: %v", err2)
		} else {
			s.logger.Warning("The server was forcefully stopped")
		}

		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(ServiceName, err)
	}

	s.state.Set(utils.StateOffline, "")

	if err != nil {
		return fmt.Errorf("an error occurred while stopping the server: %w", err)
	}

	return nil
}

// State returns the state of the service.
func (s *Server) State() (utils.StateCode, string) { return s.state.Get() }
