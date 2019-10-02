package admin

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

// Status is the status of the service
type Status struct {
	State  string
	Reason string
}

// Statuses maps a service name to its state
type Statuses map[string]Status

// getStatus is called when an HTTP request is received on the StatusURI path.
func getStatus(logger *log.Logger, services map[string]service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var statuses = make(Statuses)
		for name, serv := range services {
			code, reason := serv.State().Get()
			statuses[name] = Status{
				State:  code.Name(),
				Reason: reason,
			}
		}

		if err := writeJSON(w, statuses); err != nil {
			handleErrors(w, logger, err)
		}

	}
}
