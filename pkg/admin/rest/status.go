package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
)

// getStatus is called when an HTTP request is received on the StatusURI path.
//
// Deprecated: replaced by makeAbout.
func getStatus(logger *log.Logger, core map[string]service.Service,
	protoServices map[string]proto.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		statuses := make(api.Statuses)

		for name, serv := range core {
			code, reason := serv.State().Get()
			statuses[name] = api.Status{
				State:  code.Name(),
				Reason: reason,
			}
		}

		for name, serv := range protoServices {
			code, reason := serv.State().Get()
			statuses[name] = api.Status{
				State:  code.Name(),
				Reason: reason,
			}
		}

		handleError(w, logger, writeJSON(w, statuses))
	}
}
