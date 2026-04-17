package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

// getStatus is called when an HTTP request is received on the StatusURI path.
//
// Deprecated: replaced by makeAbout.
func getStatus(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		statuses := make(api.Statuses)

		for _, serv := range services.Core {
			code, reason := serv.State()
			statuses[serv.Name()] = api.Status{
				State:  code.String(),
				Reason: reason,
			}
		}

		services.Servers.Range(func(_ int64, serv services.Server) bool {
			code, reason := serv.State()
			statuses[serv.Name()] = api.Status{
				State:  code.String(),
				Reason: reason,
			}

			return true
		})

		handleError(w, logger, writeJSON(w, statuses))
	}
}
