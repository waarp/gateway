package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
)

// getStatus is called when an HTTP request is received on the StatusURI path.
//
// Deprecated: replaced by makeAbout.
func getStatus(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		statuses := make(api.Statuses)

		for name, serv := range services.Core {
			code, reason := serv.State()
			statuses[name] = api.Status{
				State:  code.String(),
				Reason: reason,
			}
		}

		for name, serv := range services.Servers {
			code, reason := serv.State()
			statuses[name] = api.Status{
				State:  code.String(),
				Reason: reason,
			}
		}

		handleError(w, logger, writeJSON(w, statuses))
	}
}
