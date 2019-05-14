package admin

import (
	"code.bcarlin.xyz/go/logging"
	"context"
	"crypto/tls"
	"fmt"
	"net"
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
func (service *Server) listen() {
	service.Logger.Infof("Listening on port %s", service.server.Addr)
	var err error
	if service.server.TLSConfig == nil {
		err = service.server.ListenAndServe()
	} else {
		err = service.server.ListenAndServeTLS("", "")
	}

	if err != http.ErrServerClosed {
		service.Logger.Criticalf("Unexpected error: %s", err)
	}
}

// Starts the http.Server instance
func (service *Server) startServer() error {
	// Load REST service address
	port := service.Config.Rest.Port
	_, err := net.LookupPort("tcp", port)
	if err != nil {
		return fmt.Errorf("invalid REST port (%s)", port)
	}
	addr := ":" + port

	// Load TLS configuration
	certFile := service.Config.Rest.SslCert
	keyFile := service.Config.Rest.SslKey
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
	handler.Use(mux.CORSMethodMiddleware(handler), Authentication(service.Logger))
	apiHandler := handler.PathPrefix("/api").Subrouter()
	apiHandler.HandleFunc(statusUri, GetStatus).
		Methods(http.MethodGet)

	// Create http.Server instance
	service.server = http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
		ErrorLog:  service.Logger.AsStdLog(logging.ERROR),
	}
	return nil
}

// Starts the REST service
func (service *Server) Start() error {
	if service.Logger == nil {
		service.Logger = &log.Logger{
			Logger: logging.GetLogger("admin"),
		}
		_ = service.Logger.SetOutput("stdout", "")
	}

	if service.Config == nil {
		return fmt.Errorf("missing REST configuration")
	}

	if err := service.startServer(); err != nil {
		return err
	}

	go service.listen()

	service.Logger.Info("Server started.")
	return nil
}

// Stops the REST service by first trying to shut down the service gracefully.
// If it fails after a 10 seconds delay, the service is forcefully stopped.
func (service *Server) Stop() {
	service.Logger.Info("Shutdown initiated.")

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	err := service.server.Shutdown(ctx)

	if err != nil && err != http.ErrServerClosed {
		service.Logger.Warningf("Failed to shutdown gracefully : %s", err)
		_ = service.server.Close()
		service.Logger.Warning("The service was forcefully stopped.")
	} else {
		service.Logger.Info("Shutdown complete.")
	}
}
