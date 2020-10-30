package pipeline

import (
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// DataStream is the interface representing the local source/destination of a
// transfer, typically, a TransferStream.
type DataStream interface {
	io.Reader
	io.Writer
	io.ReaderAt
	io.WriterAt
}

// Client is the interface defining a protocol client. All protocol clients
// (SFTP, R66, HTTP...) must implement this interface in order to be usable by
// the transfer executor.
// The `Client` should provide a constructor which can then be used by the
// transfer executor to initialize the client when the transfer starts. As a
// result, a `Client` is only meant to handle one transfer, and cannot be reused
// multiple times.
type Client interface {

	// Connect is the method which opens the TCP connection to the transfer
	// remote. The connection must be handled entirely by the client. The method
	// returns an error if the connection failed.
	Connect() *model.PipelineError

	// Authenticate is the method used to authenticate the connection made with
	// the `Connect` method. Thus, this method should never be called before the
	// `Connect` method. If the authentication fails, the method returns an error.
	Authenticate() *model.PipelineError

	// Request it the method which transmits the transfer request to the remote
	// using the specified protocol. The content of the file should not be sent
	// with this method. If the transfer request fails, an error is returned.
	Request() *model.PipelineError

	// Data is the method which transfers the file content to the remote. Once
	// the data has been transmitted, this method should close both the connection
	// and the local file. If an error occurs while transmitting the data, an
	// error is returned.
	Data(DataStream) *model.PipelineError

	// Close is the method which informs the remote that the data transfer is
	// finished, and that the post-tasks should now be executed. If the error
	// given as parameter is not nil, the remote will be informed of the error,
	// and will execute the error-tasks instead.
	Close(*model.PipelineError) *model.PipelineError
}
