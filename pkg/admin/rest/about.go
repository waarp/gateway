package rest

import (
	"fmt"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

func makeAbout(logger *log.Logger) http.HandlerFunc {
	gmt := time.FixedZone("GMT", 0)

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

		// TODO: replace with a middleware to add these headers in all responses
		w.Header().Set("Server", fmt.Sprintf("waarp-gatewayd/%s", version.Num))
		w.Header().Set("Date", time.Now().In(gmt).Format(time.RFC1123))
		w.Header().Set(api.DateHeader, time.Now().Format(time.RFC1123))
		w.WriteHeader(http.StatusOK)

		respBody := map[string]any{
			"coreServices": core,
			"servers":      servers,
			"clients":      clients,
		}

		handleError(w, logger, writeJSON(w, respBody))
	}
}
