package admin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"code.bcarlin.xyz/go/logging"

	"code.waarp.fr/waarp/gateway-ng/pkg/gatewayd"
	"github.com/gorilla/mux"
)

const apiURI = "/api"

// Server is the administration service
type Server struct {
	*gatewayd.WG

	listener chan signal
	server   *http.Server
}

type signal byte

const (
	LISTENING signal = iota
	SHUTDOWN
)

// listen starts the HTTP server listener on the configured port
func (admin *Server) listen() {
	admin.Logger.Admin.Infof("Listening at address %s", admin.server.Addr)
	var err error
	admin.listener <- LISTENING
	if admin.server.TLSConfig == nil {
		err = admin.server.ListenAndServe()
	} else {
		err = admin.server.ListenAndServeTLS("", "")
	}

	if err != http.ErrServerClosed {
		admin.Logger.Admin.Criticalf("Unexpected error: %s", err)
	}
	admin.listener <- SHUTDOWN
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
func (admin *Server) initServer() error {
	// Load REST admin address
	addr, err := checkAddress(admin.Config.Admin.Address)
	if err != nil {
		return err
	}

	// Load TLS configuration
	certFile := admin.Config.Admin.TLSCert
	keyFile := admin.Config.Admin.TLSKey
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
		admin.Logger.Admin.Info("No TLS certificate found, using plain HTTP.")
	}

	// Add the REST handler
	handler := mux.NewRouter()
	handler.Use(mux.CORSMethodMiddleware(handler), Authentication(admin.Logger))
	apiHandler := handler.PathPrefix(apiURI).Subrouter()
	apiHandler.HandleFunc(statusURI, GetStatus).
		Methods(http.MethodGet)

	// Create http.Server instance
	admin.server = &http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
		ErrorLog:  admin.Logger.Admin.AsStdLog(logging.ERROR),
	}
	return nil
}

// Start launches the administration service. If the service cannot be launched,
// the function returns an error.
func (admin *Server) Start() error {
	if admin.WG == nil {
		return fmt.Errorf("missing application configuration")
	}

	admin.Logger.Admin.Info("Startup command received.")
	if admin.server != nil {
		admin.Logger.Admin.Warning("The admin server is already running")
		return nil
	}

	if err := admin.initServer(); err != nil {
		admin.Logger.Admin.Errorf("Failed to start: %s", err)
		return err
	}

	admin.listener = make(chan signal, 2)
	go admin.listen()
	<-admin.listener

	admin.Logger.Admin.Info("Server started.")
	return nil
}

// Stop halts the admin service by first trying to shut it down gracefully.
// If it fails after a 10 seconds delay, the service is forcefully stopped.
func (admin *Server) Stop() {
	admin.Logger.Admin.Info("Shutdown command received...")
	if admin.server == nil {
		admin.Logger.Admin.Warning("The server was already stopped.")
		return
	}

	ctx, f := context.WithTimeout(context.Background(), time.Second*10)
	err := admin.server.Shutdown(ctx)
	f()

	if err == nil {
		<-admin.listener
		admin.Logger.Admin.Info("Shutdown complete.")
	} else {
		admin.Logger.Admin.Warningf("Failed to shutdown gracefully : %s", err)
		_ = admin.server.Close()
		admin.Logger.Admin.Warning("The admin was forcefully stopped.")
	}
	admin.server = nil
}
