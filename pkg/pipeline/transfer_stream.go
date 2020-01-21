package pipeline

import (
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// TransferStream represents the pipeline of an incoming transfer made to the
// gateway. It is a `os.File` wrapper which adds MFT operations at the stream's
// creation, during reads/writes, and at the streams closure.
type TransferStream struct {
	*os.File
	pipeline *pipeline
}

// NewServerStream creates a new stream for the given transfer and executes the
// transfer's pre tasks. If the stream cannot be created, or if the pre tasks
// fail, an error is returned.
func NewServerStream(trans model.Transfer, db *database.Db,
	logger *log.Logger, root string) (*TransferStream, error) {

	info, err := model.NewInTransferInfo(db, trans)
	if err != nil {
		return nil, err
	}

	p := &pipeline{
		Info:   *info,
		Db:     db,
		Logger: logger,
		Root:   root,
	}
	file, te := p.prologue()
	stream := &TransferStream{
		File:     file,
		pipeline: p,
	}

	if te.Code != model.TeOk {
		p.handleError(te)
		return stream, te
	}
	return stream, nil
}

// Exit executes the transfer's post tasks and sets its status to 'Done'. If an
// error occurs, during the execution, the error tasks are executed and the
// status is set to 'Error'.
func (t *TransferStream) Exit() model.TransferError {
	return t.pipeline.epilogue()
}

// ExitError executes the transfer's error tasks and then sets its status to
// 'Error'.
func (t *TransferStream) ExitError(err model.TransferError) {
	t.pipeline.handleError(err)
}
