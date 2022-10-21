// Package service defines an interface implemented by all the gateway's
// modules.
package service

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
)

// Service is the interface of an object which is considered to be a service.
type Service interface {
	// Start is the method called to start the service.
	Start() error

	// Stop is the method called to stop the service. It may include a timeout.
	Stop(ctx context.Context) error

	// State returns the state of the service.
	State() *state.State
}
