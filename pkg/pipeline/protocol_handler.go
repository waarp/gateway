package pipeline

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type ClientConstructor func(*log.Logger, *model.TransferContext) (Client, error)

var ClientConstructors = map[string]ClientConstructor{}

// Client is the interface defining a protocol client. All protocol clients
// (SFTP, R66, HTTP...) must implement this interface in order to be usable by
// the transfer pipeline.
// The client must also provide a constructor and add it to the ClientConstructors
// map.
type Client interface {

	// Request opens a connection to the transfer partner and then sends a
	// transfer request to the remote.
	Request() error

	// Data is the method which transfers the file content between the given
	// DataStream and the remote.
	Data(DataStream) error

	// EndTransfer informs the partner of the transfer's end, and then closes
	// the connection.
	EndTransfer() error

	ErrorHandler
}

type Server interface {
	ErrorHandler
}

// ErrorHandler is an interface which must be implemented by both clients and
// servers, providing a function which informs the remote partner when an error
// occurred.
type ErrorHandler interface {
	// SendError sends the given error to the remote, and then closes the connection.
	SendError(error)
}

// PreTasksHandler is an interface which clients can optionally implement in
// order to give more control over the execution of pre-tasks on the remote partner.
// If the client does not implement this interface, it will be assumed that
// pre-tasks are executed just after the reception of the request (i.e. at the
// end of the Client.Request method).
type PreTasksHandler interface {

	// BeginPreTasks tells the remote partner to begin executing its pre-tasks.
	// The function returns once the pre-tasks are over.
	BeginPreTasks() error

	// EndPreTasks informs the remote partner that the client has finished executing
	// its pre-tasks.
	EndPreTasks() error
}

// PostTasksHandler is an interface which clients can optionally implement in
// order to give more control over the execution of post-tasks on the remote partner.
// If the client does not implement this interface, it will be assumed that
// post-tasks are executed just before closing the connection (i.e. at the
// beginning of the Client.EndTransfer method).
type PostTasksHandler interface {

	// BeginPostTasks tells the remote partner to begin executing its post-tasks.
	// The function returns once the post-tasks are over.
	BeginPostTasks() error

	// EndPreTasks informs the remote partner that the client has finished executing
	// its post-tasks.
	EndPostTasks() error
}

// PauseHandler is an interface which clients and servers can optionally
// implement in order to allow the agent to inform the remote partner when a
// transfer has been paused by a user. If this interface is not implemented,
// Client.SendError or Server.SendError will be used instead.
type PauseHandler interface {
	// Pause informs the partner that the transfer has been paused.
	Pause() error
}

// CancelHandler is an interface which clients and servers can optionally
// implement in order to allow the agent to inform the remote partner when a
// transfer has been cancelled by a user. If this interface is not implemented,
// Client.SendError or Server.SendError will be used instead.
type CancelHandler interface {
	// Cancel informs the partner that the transfer has been cancelled.
	Cancel() error
}
