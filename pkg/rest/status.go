package rest

import (
    "net/http"
    "sync"

    "code.waarp.fr/waarp/gateway-ng/pkg/log"
)

// This is the access path to this handler
const BASE_URI string = "/status"

// Status handler is the handler in charge of status requests
type StatusHandler struct {
    mutex  sync.Mutex
    logger *log.Logger
}

// Function called when an HTTP request is received on the BASE_URI path.
// For now, it just send an OK status code.
func (handler *StatusHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
    handler.mutex.Lock()
    defer handler.mutex.Unlock()

    handler.logger.Info("Received status request.")
    writer.WriteHeader(200)
}

// Starts the Status handler
func StartStatusHandler(port string, logger *log.Logger) {
    handler := &StatusHandler{logger: logger}
    http.Handle(BASE_URI, handler)
    cause := http.ListenAndServe(port, handler)

    if cause == http.ErrServerClosed {
        handler.logger.Info("Status handler shutdown.")
    } else {
        handler.logger.Error("Unexpected status handler shutdown.")
    }
}