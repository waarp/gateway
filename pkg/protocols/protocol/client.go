package protocol

import (
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

type SendFile interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

type ReceiveFile interface {
	io.Writer
	io.WriterAt
	io.Seeker
}

// TransferClient is the interface defining a protocol client. All protocol
// clients must implement this interface in order to be usable by the transfer
// pipeline.
//
// If the protocol permits it, the client can also implement the PreTasksHandler,
// PostTasksHandler, PauseHandler and CancelHandler interfaces to allow more
// refined control over the transfer.
//
// Important note: To avoid sending its own errors back to the server, The
// ClientPipeline will NOT call EndTransfer after an error produced by the
// TransferClient itself. Thus, it is the TransferClient's responsibility to
// notify the server when an error occurs locally on the client's side.
type TransferClient interface {
	// Request opens a connection to the transfer partner and then sends a
	// transfer request to the remote.
	Request() *pipeline.Error

	// Send uploads the given file content to the remote partner.
	Send(file SendFile) *pipeline.Error

	// Receive downloads data from the remote partner and writes it to the given
	// file.
	Receive(file ReceiveFile) *pipeline.Error

	// EndTransfer informs the partner of the transfer's end, and then closes
	// the connection.
	EndTransfer() *pipeline.Error

	// SendError sends the given error to the remote partner, and then closes
	// the connection.
	SendError(code types.TransferErrorCode, msg string)
}

// PreTasksHandler is an interface which clients can optionally implement in
// order to give more control over the execution of pre-tasks on the remote partner.
// If the client does not implement this interface, it will be assumed that
// pre-tasks are executed just after the reception of the request (so at the
// end of the TransferClient.Request method).
type PreTasksHandler interface {
	// BeginPreTasks tells the remote partner to begin executing its pre-tasks.
	// The function returns once the pre-tasks are over.
	BeginPreTasks() *pipeline.Error

	// EndPreTasks informs the remote partner that the client has finished executing
	// its pre-tasks.
	EndPreTasks() *pipeline.Error
}

// PostTasksHandler is an interface which clients can optionally implement in
// order to give more control over the execution of post-tasks on the remote partner.
// If the client does not implement this interface, it will be assumed that
// post-tasks are executed just before closing the connection (so at the
// beginning of the TransferClient.EndTransfer method).
type PostTasksHandler interface {
	// BeginPostTasks tells the remote partner to begin executing its post-tasks.
	// The function returns once the post-tasks are over.
	BeginPostTasks() *pipeline.Error

	// EndPostTasks informs the remote partner that the client has finished executing
	// its post-tasks.
	EndPostTasks() *pipeline.Error
}

// PauseHandler is an interface which clients and servers can optionally
// implement in order to allow the agent to inform the remote partner when a
// transfer has been paused by a user. If this interface is not implemented,
// TransferClient.SendError or Server.SendError will be used instead.
type PauseHandler interface {
	// Pause informs the partner that the transfer has been paused.
	Pause() *pipeline.Error
}

// CancelHandler is an interface which clients and servers can optionally
// implement in order to allow the agent to inform the remote partner when a
// transfer has been canceled by a user. If this interface is not implemented,
// TransferClient.SendError or Server.SendError will be used instead.
type CancelHandler interface {
	// Cancel informs the partner that the transfer has been canceled.
	Cancel() *pipeline.Error
}
