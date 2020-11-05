package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/models"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

// getStatus is called when an HTTP request is received on the StatusURI path.
func getStatus(logger *log.Logger, services map[string]service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var statuses = make(models.Statuses)
		for name, serv := range services {
			code, reason := serv.State().Get()
			statuses[name] = models.Status{
				State:  code.Name(),
				Reason: reason,
			}
		}

		if err := writeJSON(w, statuses); err != nil {
			handleErrors(w, logger, err)
		}
	}
}
