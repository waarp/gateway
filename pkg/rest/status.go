package rest

import (
	"code.waarp.fr/waarp/gateway-ng/pkg/log"
	"net/http"
)

// This is the access path to this handler
const statusUri string = "/status"

// Status handler is the handler in charge of status requests
type statusHandler struct {
	logger *log.Logger
}

// Function called when an HTTP request is received on the statusUri path.
// For now, it just send an OK status code.
func (handler *statusHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.logger.Debug("Received status request")
	_, _ = writer.Write([]byte{})

}
