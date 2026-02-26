package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

func makeAbout(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var (
			core    []api.Service
			servers []api.Service
			clients []api.Service
		)

		for name, serv := range services.Core {
			code, reason := serv.State()

			core = append(core, api.Service{
				Name:   name,
				State:  code.String(),
				Reason: reason,
			})
		}

		for name, serv := range services.Servers {
			code, reason := serv.State()

			servers = append(servers, api.Service{
				Name:   name,
				State:  code.String(),
				Reason: reason,
			})
		}

		for name, client := range services.Clients {
			code, reason := client.State()

			clients = append(clients, api.Service{
				Name:   name,
				State:  code.String(),
				Reason: reason,
			})
		}

		w.WriteHeader(http.StatusOK)

		respBody := map[string]any{
			"coreServices": core,
			"servers":      servers,
			"clients":      clients,
		}

		handleError(w, logger, writeJSON(w, respBody))
	}
}
