package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
)

type Service struct {
	Name   string
	State  string
	Reason string
}

func ListServices() (core, servers, clients []Service) {
	for _, serv := range services.Core {
		code, reason := serv.State()

		core = append(core, Service{
			Name:   serv.Name(),
			State:  code.String(),
			Reason: reason,
		})
	}

	services.Servers.Range(func(_ int64, serv services.Server) bool {
		code, reason := serv.State()

		servers = append(servers, Service{
			Name:   serv.Name(),
			State:  code.String(),
			Reason: reason,
		})

		return true
	})

	services.Clients.Range(func(_ int64, client services.Client) bool {
		code, reason := client.State()

		clients = append(clients, Service{
			Name:   client.Name(),
			State:  code.String(),
			Reason: reason,
		})

		return true
	})

	return core, servers, clients
}
