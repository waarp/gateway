// Package service defines an interface implemented by all the gateway's
// modules.
package service

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service/state"
)

type Stopper interface {
	// Stop is the method called to stop the service. It may include a timeout.
	Stop(ctx context.Context) error
}

// Service is the interface of an object which is considered to be a service.
type Service interface {
	// Start is the method called to start the service.
	Start() error

	Stopper

	// State returns the state of the service.
	State() *state.State
}

// ProtoService is the interface of a transfer server (implementing a protocol)
// which is considered to be a service.
type ProtoService interface {
	// Start is the method called to start the service.
	Start(*model.LocalAgent) error

	Stopper

	// State returns the state of the service.
	State() *state.State

	// ManageTransfers returns a map of the transfers currently running on the
	// server, along with a few functions to manage each of those transfers.
	ManageTransfers() *TransferMap
}
