// Package services provides lists of all the gateway's internal services.
// This includes core services, clients and servers.
package services

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

//nolint:gochecknoglobals //global vars are required here
var (
	Core    = map[string]Service{}
	Clients = map[string]Client{}
	Servers = map[string]Server{}
)

type (
	Service = protocol.StartStopper
	Client  = protocol.Client
	Server  = protocol.Server
)
