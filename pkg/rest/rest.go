package rest

import "code.waarp.fr/waarp/gateway-ng/pkg/log"

// The port on which the REST interface is listening
const PORT string = ":8080"


// Starts all the REST handlers
func StartRestService(logger *log.Logger) {
	go StartStatusHandler(PORT, logger)
}
