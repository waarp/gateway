package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// getStatus is called when an HTTP request is received on the StatusURI path.
func getStatus(logger *log.Logger, db *database.DB, core map[string]service.Service,
	protoServices map[int64]proto.Service,
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

		var servers model.LocalAgents
		if err := db.Select(&servers).Run(); handleError(w, logger, err) {
			return
		}

		for i := range servers {
			serv, ok := protoServices[servers[i].ID]
			if !ok {
				logger.Warning("Could not find the '%s' service", servers[i].Name)

				continue
			}

			code, reason := serv.State().Get()
			statuses[servers[i].Name] = api.Status{
				State:  code.Name(),
				Reason: reason,
			}
		}

		err := writeJSON(w, statuses)
		handleError(w, logger, err)
	}
}
