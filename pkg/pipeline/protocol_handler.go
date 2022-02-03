// Package pipeline defines the execution of a transfer through the Pipeline
// struct. A pipeline can be initiated using either the NewClientPipeline or the
// NewServerPipeline function (one for each side of the transfer). Once initiated,
// a Pipeline exposes multiple functions for every step of the transfer.
//
// The package also defines the interfaces which should be implemented by the
// gateway's different protocol handlers (clients & servers).
package pipeline

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

//nolint:gochecknoglobals // global var is used by design
// ClientConstructors is a map containing constructors for the various clients
// supported by the gateway. It associates each protocol with the constructor for
// its client. In order for the gateway to be able to execute a transfer in a
// given protocol as a client, the constructor for the protocol's client must
// be added to this map.
var ClientConstructors = map[string]ClientConstructor{}

// ClientConstructor is the type defining the signature which all clients
// constructor functions must satisfy.
type ClientConstructor func(*Pipeline) (Client, *types.TransferError)

// Client is the interface defining a protocol client. All protocol clients
// (SFTP, R66, HTTP, etc) must implement this interface in order to be usable by
// the transfer pipeline.
// The client must also provide a constructor and add it to the ClientConstructors
// map.
type Client interface {
	// Request opens a connection to the transfer partner and then sends a
	// transfer request to the remote.
	Request() *types.TransferError

	// Data is the method which transfers the file content between the given
	// DataStream and the remote.
	Data(DataStream) *types.TransferError

	// EndTransfer informs the partner of the transfer's end, and then closes
	// the connection.
	EndTransfer() *types.TransferError

	// SendError sends the given error to the remote partner, and then closes
	// the connection.
	SendError(*types.TransferError)
}

// Server is the interface exposing the various handler functions which servers
// must implement in order to initiate a ServerPipeline. The only mandatory
// function is SendError. Optionally, servers can also implement the PauseHandler
// and CancelHandler interfaces to allow more refined interruption handling.
type Server interface {
	// Interrupt informs the partner that the transfer has been interrupted by
	// an external factor.
	Interrupt()
}

// PreTasksHandler is an interface which clients can optionally implement in
// order to give more control over the execution of pre-tasks on the remote partner.
// If the client does not implement this interface, it will be assumed that
// pre-tasks are executed just after the reception of the request (so at the
// end of the Client.Request method).
type PreTasksHandler interface {
	// BeginPreTasks tells the remote partner to begin executing its pre-tasks.
	// The function returns once the pre-tasks are over.
	BeginPreTasks() *types.TransferError

	// EndPreTasks informs the remote partner that the client has finished executing
	// its pre-tasks.
	EndPreTasks() *types.TransferError
}

// PostTasksHandler is an interface which clients can optionally implement in
// order to give more control over the execution of post-tasks on the remote partner.
// If the client does not implement this interface, it will be assumed that
// post-tasks are executed just before closing the connection (so at the
// beginning of the Client.EndTransfer method).
type PostTasksHandler interface {
	// BeginPostTasks tells the remote partner to begin executing its post-tasks.
	// The function returns once the post-tasks are over.
	BeginPostTasks() *types.TransferError

	// EndPostTasks informs the remote partner that the client has finished executing
	// its post-tasks.
	EndPostTasks() *types.TransferError
}

// PauseHandler is an interface which clients and servers can optionally
// implement in order to allow the agent to inform the remote partner when a
// transfer has been paused by a user. If this interface is not implemented,
// Client.SendError or Server.SendError will be used instead.
type PauseHandler interface {
	// Pause informs the partner that the transfer has been paused.
	Pause() *types.TransferError
}

// CancelHandler is an interface which clients and servers can optionally
// implement in order to allow the agent to inform the remote partner when a
// transfer has been canceled by a user. If this interface is not implemented,
// Client.SendError or Server.SendError will be used instead.
type CancelHandler interface {
	// Cancel informs the partner that the transfer has been canceled.
	Cancel() *types.TransferError
}
