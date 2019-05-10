package rest

import (
	"net/http"
	"sync"

	"code.waarp.fr/waarp/gateway-ng/pkg/log"
)

// This is the access path to this handler
const STATUS_URI string = "/status"

// Status handler is the handler in charge of status requests
type statusHandler struct {
	http.Handler

	mutex  sync.Mutex
	logger *log.Logger
}

// Function called when an HTTP request is received on the STATUS_URI path.
// For now, it just send an OK status code.
func (handler *statusHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()

	handler.logger.Info("Received status request.")
	_, _ = writer.Write([]byte{})

}
