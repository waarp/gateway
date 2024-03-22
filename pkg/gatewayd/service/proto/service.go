// Package proto defines an interface which all local servers should implement
// in order to be instantiated as services.
package proto

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Service is the interface of a transfer server (implementing a protocol)
// which is considered to be a service.
type Service interface {
	// Start is the method called to start the service.
	Start(serv *model.LocalAgent) error

	Stopper
	Statter
	TransferManager
}

type Stopper interface {
	// Stop is the method called to stop the service. It may include a timeout.
	Stop(ctx context.Context) error
}

type Statter interface {
	// State returns the state of the service.
	State() *state.State
}

type TransferManager interface {
	// ManageTransfers returns a map of the transfers currently running on the
	// server, along with a few functions to manage each of those transfers.
	ManageTransfers() *service.TransferMap
}
