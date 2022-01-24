package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// getStatus is called when an HTTP request is received on the StatusURI path.
func getStatus(logger *log.Logger, core map[string]service.Service,
	proto map[string]service.ProtoService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		statuses := make(api.Statuses)

		for name, serv := range core {
			code, reason := serv.State().Get()
			statuses[name] = api.Status{
				State:  code.Name(),
				Reason: reason,
			}
		}

		for name, serv := range proto {
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
