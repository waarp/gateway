package rest

import (
	"fmt"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

func makeAbout(logger *log.Logger, core map[string]service.Service,
	protoServices map[string]proto.Service,
) http.HandlerFunc {
	gmt := time.FixedZone("GMT", 0)

	return func(w http.ResponseWriter, _ *http.Request) {
		var (
			coreServices []api.Service
			servers      []api.Service
			clients      []api.Service
		)

		for name, serv := range core {
			code, reason := serv.State().Get()

			coreServices = append(coreServices, api.Service{
				Name:   name,
				State:  code.Name(),
				Reason: reason,
			})
		}

		for name, serv := range protoServices {
			code, reason := serv.State().Get()

			servers = append(servers, api.Service{
				Name:   name,
				State:  code.Name(),
				Reason: reason,
			})
		}

		for name, client := range pipeline.Clients {
			code, reason := client.State().Get()

			clients = append(clients, api.Service{
				Name:   name,
				State:  code.Name(),
				Reason: reason,
			})
		}

		// TODO: replace with a middleware to add these headers in all responses
		w.Header().Set("Server", fmt.Sprintf("waarp-gatewayd/%s", version.Num))
		w.Header().Set("Date", time.Now().In(gmt).Format(time.RFC1123))
		w.Header().Set(api.DateHeader, time.Now().Format(time.RFC1123))
		w.WriteHeader(http.StatusOK)

		respBody := map[string]any{
			"coreServices": coreServices,
			"servers":      servers,
			"clients":      clients,
		}

		handleError(w, logger, writeJSON(w, respBody))
	}
}
