package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// getStatus is called when an HTTP request is received on the StatusURI path.
func getStatus(logger *log.Logger, services map[string]service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var statuses = make(api.Statuses)
		for name, serv := range services {
			code, reason := serv.State().Get()
			statuses[name] = api.Status{
				State:  code.Name(),
				Reason: reason,
			}
		}

		err := writeJSON(w, statuses)
		handleError(w, logger, err)
	}
}
