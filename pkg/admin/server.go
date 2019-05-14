package admin

import (
	"code.bcarlin.xyz/go/logging"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
	"code.waarp.fr/waarp/gateway-ng/pkg/log"
	"github.com/gorilla/mux"
)

// This is the REST service interface
type Server struct {
	Config *conf.ServerConfig
	Logger *log.Logger

	server http.Server
}

// Starts the server listener, and restarts the service if an unexpected
// error occurred while listening
func (admin *Server) listen() {
	admin.Logger.Infof("Listening at address %s", admin.server.Addr)
	var err error
	if admin.server.TLSConfig == nil {
		err = admin.server.ListenAndServe()
	} else {
		err = admin.server.ListenAndServeTLS("", "")
	}

	if err != http.ErrServerClosed {
		admin.Logger.Criticalf("Unexpected error: %s", err)
	}
}

// Starts the http.Server instance
func (admin *Server) startServer() error {
	// Load REST admin address
	addr := admin.Config.Admin.Address

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
	apiHandler := handler.PathPrefix("/api").Subrouter()
	apiHandler.HandleFunc(statusUri, GetStatus).
		Methods(http.MethodGet)

	// Create http.Server instance
	admin.server = http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
		ErrorLog:  admin.Logger.AsStdLog(logging.ERROR),
	}
	return nil
}

// Starts the REST service
func (admin *Server) Start() error {
	if admin.Logger == nil {
		admin.Logger = &log.Logger{
			Logger: logging.GetLogger("admin"),
		}
		_ = admin.Logger.SetOutput("stdout", "")
	}

	if admin.Config == nil {
		return fmt.Errorf("missing REST configuration")
	}

	if err := admin.startServer(); err != nil {
		return err
	}

	go admin.listen()

	admin.Logger.Info("Server started.")
	return nil
}

// Stops the REST service by first trying to shut down the service gracefully.
// If it fails after a 10 seconds delay, the service is forcefully stopped.
func (admin *Server) Stop() {
	admin.Logger.Info("Shutdown initiated.")

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	err := admin.server.Shutdown(ctx)

	if err != nil && err != http.ErrServerClosed {
		admin.Logger.Warningf("Failed to shutdown gracefully : %s", err)
		_ = admin.server.Close()
		admin.Logger.Warning("The admin was forcefully stopped.")
	} else {
		admin.Logger.Info("Shutdown complete.")
	}
}
