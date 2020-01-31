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
	trans model.Transfer) (*TransferStream, error) {

	if trans.ID == 0 {
		if err := createTransfer(logger, db, &trans); err != nil {
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

	t.Pipeline.Rule = &model.Rule{ID: trans.RuleID}
	if err := t.Db.Get(t.Rule); err != nil {
		return nil, err
	}

	t.Signals = Signals.Add(t.Transfer.ID)

	t.proc = &tasks.Processor{
		Db:       t.Db,
		Logger:   t.Logger,
		Rule:     t.Rule,
		Transfer: t.Transfer,
		Signals:  t.Signals,
	}
	return t, nil
}

// Start opens/creates the stream's local file. If necessary, the method also
// creates the file's parent directories.
func (t *TransferStream) Start() (err *model.PipelineError) {
	if !t.Rule.IsSend {
		if err := makeDir(t.Root, t.Rule.Path); err != nil {
			return model.NewPipelineError(model.TeForbidden, err.Error())
		}
	}

	t.File, err = getFile(t.Logger, t.Root, t.Rule, t.Transfer)
	return
}

func (t *TransferStream) Read(p []byte) (n int, err error) {
	if e := checkSignal(t.Signals); e != nil {
		return 0, e
	}

	n, err = t.File.Read(p)
	t.Transfer.Progress += uint64(n)
	if err := t.Transfer.Update(t.Db); err != nil {
		return 0, err
	}
	return
}

func (t *TransferStream) Write(p []byte) (n int, err error) {
	if e := checkSignal(t.Signals); e != nil {
		return 0, e
	}

	n, err = t.File.Write(p)
	t.Transfer.Progress += uint64(n)
	if err := t.Transfer.Update(t.Db); err != nil {
		return 0, err
	}
	return
}

// ReadAt reads the stream, starting at the given offset.
func (t *TransferStream) ReadAt(p []byte, off int64) (n int, err error) {
	if e := checkSignal(t.Signals); e != nil {
		t.Logger.Criticalf("SIGNAL RECEIVED ON READ")
		return 0, e
	}

	n, err = t.File.ReadAt(p, off)
	t.Transfer.Progress += uint64(n)
	if err := t.Transfer.Update(t.Db); err != nil {
		return 0, err
	}
	return
}

// WriteAt writes the given bytes to the stream, starting at the given offset.
func (t *TransferStream) WriteAt(p []byte, off int64) (n int, err error) {
	if e := checkSignal(t.Signals); e != nil {
		t.Logger.Criticalf("SIGNAL RECEIVED ON WRITE")
		return 0, e
	}

	n, err = t.File.WriteAt(p, off)
	t.Transfer.Progress += uint64(n)
	if err := t.Transfer.Update(t.Db); err != nil {
		return 0, err
	}
	return
}
