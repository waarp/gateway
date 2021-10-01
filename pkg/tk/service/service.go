// Package service defines an interface implemented by all the gateway's
// modules.
package service

import (
	"context"
	"sync"
)

const (
	// DatabaseServiceName is the name of the gatewayd database service.
	DatabaseServiceName = "Database"

	// AdminServiceName is the name of the administration interface service.
	AdminServiceName = "Admin"

	// ControllerServiceName is the name of the controller service.
	ControllerServiceName = "Controller"
)

// IsReservedServiceName returns whether the given service name is already a
// reserved name. Reserved names cannot be used as service names.
func IsReservedServiceName(name string) bool {
	return name == DatabaseServiceName || name == AdminServiceName ||
		name == ControllerServiceName
}

// Service is the interface of an object which is considered to be a service.
type Service interface {
	// Start is the method called to start the service.
	Start() error

	// Stop is the method called to stop the service. It may include a timeout.
	Stop(ctx context.Context) error

	// State returns the state of the service.
	State() *State
}

// ProtoService is the interface of an transfer server (implementing a protocol)
// which is considered to be a service.
type ProtoService interface {
	Service

	// ManageTransfers returns a map of the transfers currently running on the
	// server, along with a few functions to manage each of those transfers.
	ManageTransfers() *TransferMap
}

// StateCode represents the state of a service.
// FIXME: Should implement json.Marshaler and json.Unmarshaler.
type StateCode uint8

const (
	// Offline is the state used to indicate that a service is stopped because
	// it has been disabled or stopped by the administrator. If it is not
	// working because of any error, the error state must be used.
	Offline StateCode = iota

	// Starting means the service is in its starting phase.
	Starting

	// Running means the service is alive and functions normally.
	Running

	// ShuttingDown means that the service has been asked to exit.
	ShuttingDown

	// Error means that the service is not properly functioning.
	Error
)

// Name returns the human representation is StateCode as a string.
// FIXME: Should be String().
func (s StateCode) Name() string {
	switch s {
	case Starting:
		return "Starting"
	case Running:
		return "Running"
	case ShuttingDown:
		return "Shutting down"
	case Error:
		return "Error"
	default:
		return "Offline"
	}
}

// State handles a service global state. It associates a code with a reason.
// The reason contains an message explaining why the service is in this state.
type State struct {
	code   StateCode
	reason string
	mutex  sync.RWMutex
}

// Get returns a the state code and the associated reason.
func (s *State) Get() (StateCode, string) {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	return s.code, s.reason
}

// Set allows one to change the StateCode of a service and the associated
// reason.
func (s *State) Set(code StateCode, reason string) {
	defer s.mutex.Unlock()
	s.mutex.Lock()
	s.code = code
	s.reason = reason
}
