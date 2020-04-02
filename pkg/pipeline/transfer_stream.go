package pipeline

import (
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

// TransferStream represents the Pipeline of an incoming transfer made to the
// gateway. It is a `os.File` wrapper which adds MFT operations at the stream's
// creation, during reads/writes, and at the streams closure.
type TransferStream struct {
	*os.File
	*Pipeline
}

// NewTransferStream initialises a new stream for the given transfer. This stream
// can then be used to execute a transfer.
func NewTransferStream(logger *log.Logger, db *database.Db, root string,
	trans model.Transfer) (*TransferStream, model.TransferError) {

	if trans.ID == 0 {
		if err := createTransfer(logger, db, &trans); err.Code != model.TeOk {
			return nil, err
		}
	}

	t := &TransferStream{
		Pipeline: &Pipeline{
			Db:       db,
			Logger:   logger,
			Root:     root,
			Transfer: &trans,
		},
	}

	t.Pipeline.rule = &model.Rule{ID: trans.RuleID}
	if err := t.Db.Get(t.rule); err != nil {
		return nil, model.NewTransferError(model.TeInternal, err.Error())
	}

	t.Signals = make(chan model.Signal)
	Signals.Store(t.Transfer.ID, t.Signals)

	t.proc = &tasks.Processor{
		Db:       t.Db,
		Logger:   t.Logger,
		Rule:     t.rule,
		Transfer: t.Transfer,
		Signals:  t.Signals,
	}
	return t, model.TransferError{}
}

// Start opens/creates the stream's local file. If necessary, the method also
// creates the file's parent directories.
func (t *TransferStream) Start() (err model.TransferError) {
	if !t.rule.IsSend {
		if err := makeDir(t.Root, t.rule.Path); err != nil {
			return model.NewTransferError(model.TeForbidden, err.Error())
		}
	}

	t.File, err = getFile(t.Logger, t.Root, t.rule, t.Transfer)
	return
}
