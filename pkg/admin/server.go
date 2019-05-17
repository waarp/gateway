package admin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	//"net"
	"net/http"
	//"strings"
	"time"

	"code.bcarlin.xyz/go/logging"

	"code.waarp.fr/waarp/gateway-ng/pkg/gatewayd"
	"github.com/gorilla/mux"
)

const apiURI = "/api"

// Server is the administration service
type Server struct {
	*gatewayd.WG

	server http.Server
}

// listen starts the HTTP server listener on the configured port
func (admin *Server) listen() {
	admin.Logger.Admin.Infof("Listening at address %s", admin.server.Addr)
	var err error
	if admin.server.TLSConfig == nil {
		err = admin.server.ListenAndServe()
	} else {
		err = admin.server.ListenAndServeTLS("", "")
	}

	if err != http.ErrServerClosed {
		admin.Logger.Admin.Criticalf("Unexpected error: %s", err)
	}
}

// checkAddress checks if the address given in the configuration is a
// valid address on which the server can listen
func checkAddress(strAddr string) error {
	host, port, err := net.SplitHostPort(strAddr)
	if err != nil {
		return err
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if _, err := net.LookupIP(host); err != nil {
		return fmt.Errorf("invalid admin address '%s'", host)
	}
	if _, err := net.LookupPort("tcp", port); err != nil {
		return  fmt.Errorf("invalid admin port '%s'", port)
	}
	return nil
}

// initServer initializes the HTTP server instance using the parameters defined
// in the Admin configuration.
// If the configuration is invalid, this function returns an error.
func (admin *Server) initServer() error {
	// Load REST admin address
	addr := admin.Config.Admin.Address
	err := checkAddress(addr)
	if err != nil {
		return err
	}

	// Load TLS configuration
	certFile := admin.Config.Admin.SslCert
	keyFile := admin.Config.Admin.SslKey
	var tlsConfig *tls.Config = nil
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("could not load REST certificate (%s)", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	// Add the REST handler
	handler := mux.NewRouter()
	handler.Use(mux.CORSMethodMiddleware(handler), Authentication(admin.Logger))
	apiHandler := handler.PathPrefix(apiURI).Subrouter()
	apiHandler.HandleFunc(statusUri, GetStatus).
		Methods(http.MethodGet)

	// Create http.Server instance
	admin.server = http.Server{
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

	if err := admin.initServer(); err != nil {
		return err
	}

	go admin.listen()

	admin.Logger.Admin.Info("Server started.")
	return nil
}

// Stop halts the admin service by first trying to shut it down gracefully.
// If it fails after a 10 seconds delay, the service is forcefully stopped.
func (admin *Server) Stop() {
	admin.Logger.Admin.Info("Shutdown initiated.")

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	err := admin.server.Shutdown(ctx)

	if err != nil && err != http.ErrServerClosed {
		admin.Logger.Admin.Warningf("Failed to shutdown gracefully : %s", err)
		_ = admin.server.Close()
		admin.Logger.Admin.Warning("The admin was forcefully stopped.")
	} else {
		admin.Logger.Admin.Info("Shutdown complete.")
	}
}
