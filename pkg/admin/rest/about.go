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

		for _, serv := range services.Core {
			code, reason := serv.State()

			core = append(core, api.Service{
				Name:   serv.Name(),
				State:  code.String(),
				Reason: reason,
			})
		}

		services.Servers.Range(func(_ int64, serv services.Server) bool {
			code, reason := serv.State()

			servers = append(servers, api.Service{
				Name:   serv.Name(),
				State:  code.String(),
				Reason: reason,
			})

			return true
		})

		services.Clients.Range(func(_ int64, cli services.Client) bool {
			code, reason := cli.State()

			clients = append(clients, api.Service{
				Name:   cli.Name(),
				State:  code.String(),
				Reason: reason,
			})

			return true
		})

		w.WriteHeader(http.StatusOK)

		respBody := map[string]any{
			"coreServices": core,
			"servers":      servers,
			"clients":      clients,
		}

		handleError(w, logger, writeJSON(w, respBody))
	}
}
