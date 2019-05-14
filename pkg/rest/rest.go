package rest

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
)

// This is the REST service interface
type Service struct {
	Config *conf.ServerConfig
	Logger *log.Logger

	server http.Server
}

// Attempts to restart the REST service. If the attempt fails 3 times,
// log it and panic.
func (service *Service) attemptRestart()  {
	var retries = 3

	for retries > 0 {
		service.Stop()
		service.Logger.Infof(
			"Attempting to restart service (remaining attempts: %d)", retries)

		if err := service.Start(); err != nil {
			retries--
			service.Logger.Errorf("Failed to restart service: %s", err)
		} else {
			return
		}
	}
	service.Logger.Critical("Could not restart service. Exiting...")
	panic("REST fatal error")
}

// Starts the server listener, and restarts the service if an unexpected
// error occurred while listening
func (service *Service) listen() {
	service.Logger.Infof("Listening on port %s", service.server.Addr)
	var err error
	if service.server.TLSConfig == nil {
		err = service.server.ListenAndServe()
	} else {
		fmt.Printf("skip -> %t\n", service.server.TLSConfig.InsecureSkipVerify)
		err = service.server.ListenAndServeTLS("", "")
	}

	if err != http.ErrServerClosed {
		service.Logger.Errorf("Unexpected error: %s", err)
		service.attemptRestart()
	}
}

// Starts the http.Server instance
func (service *Service) startServer() error {
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
			Certificates:       []tls.Certificate{cert},
		}
	}

	// Add the REST handlers
	status := &statusHandler{ logger: service.Logger}
	handler := http.NewServeMux()
	handler.HandleFunc(statusUri, status.ServeHTTP)

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
func (service *Service) Start() error {
	if service.Logger == nil {
		service.Logger = &log.Logger{
			Logger: logging.GetLogger("rest"),
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

	service.Logger.Info("Service started.")
	return nil
}

// Stops the REST service by first trying to shut down the service gracefully.
// If it fails after a 10 seconds delay, the service is forcefully stopped.
func (service *Service) Stop() {
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
