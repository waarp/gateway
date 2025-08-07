package internal

import "code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"

type Service struct {
	Name   string
	State  string
	Reason string
}

func ListServices() (core, servers, clients []Service) {
	for name, serv := range services.Core {
		code, reason := serv.State()

		core = append(core, Service{
			Name:   name,
			State:  code.String(),
			Reason: reason,
		})
	}

	for name, serv := range services.Servers {
		code, reason := serv.State()

		servers = append(servers, Service{
			Name:   name,
			State:  code.String(),
			Reason: reason,
		})
	}

	for name, client := range services.Clients {
		code, reason := client.State()

		clients = append(clients, Service{
			Name:   name,
			State:  code.String(),
			Reason: reason,
		})
	}

	return core, servers, clients
}
