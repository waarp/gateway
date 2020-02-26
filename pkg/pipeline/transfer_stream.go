package pipeline

import (
	"context"
	"io"
	"os"
	"path/filepath"

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
func NewTransferStream(ctx context.Context, logger *log.Logger, db *database.Db,
	root string, trans model.Transfer) (*TransferStream, error) {

	if trans.IsServer {
		if err := TransferInCount.add(); err != nil {
			return nil, err
		}
	} else {
		if err := TransferOutCount.add(); err != nil {
			return nil, err
		}
	}

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
			Ctx:      ctx,
		},
	}

	t.Pipeline.Rule = &model.Rule{ID: trans.RuleID}
	if err := t.Db.Get(t.Rule); err != nil {
		logger.Criticalf("Failed to retrieve transfer rule: %s", err.Error())
		return nil, &model.PipelineError{Kind: model.KindDatabase}
	}

	t.Signals = Signals.Add(t.Transfer.ID)

	t.proc = &tasks.Processor{
		Db:       t.Db,
		Logger:   t.Logger,
		Rule:     t.Rule,
		Transfer: t.Transfer,
		Signals:  t.Signals,
		Ctx:      ctx,
	}
	return t, nil
}

// Start opens/creates the stream's local file. If necessary, the method also
// creates the file's parent directories.
func (t *TransferStream) Start() (err *model.PipelineError) {
	if !t.Rule.IsSend {
		if err := makeDir(t.Root, t.Rule.Path); err != nil {
			t.Logger.Errorf("Failed to create dest directory: %s", err)
			return model.NewPipelineError(model.TeForbidden, err.Error())
		}
		if err := makeDir(t.Root, "tmp"); err != nil {
			t.Logger.Errorf("Failed to create temp directory: %s", err)
			return model.NewPipelineError(model.TeForbidden, err.Error())
		}
	}

	t.File, err = getFile(t.Logger, t.Root, t.Rule, t.Transfer)
	return
}

func (t *TransferStream) Read(p []byte) (n int, err error) {
	if t.Transfer.Step == model.StepPreTasks {
		t.Transfer.Step = model.StepData
		if dbErr := t.Transfer.Update(t.Db); dbErr != nil {
			return 0, &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	if e := checkSignal(t.Ctx, t.Signals); e != nil {
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
	if t.Transfer.Step == model.StepPreTasks {
		t.Transfer.Step = model.StepData
		if dbErr := t.Transfer.Update(t.Db); dbErr != nil {
			return 0, &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	if e := checkSignal(t.Ctx, t.Signals); e != nil {
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
	if t.Transfer.Step == model.StepPreTasks {
		t.Transfer.Step = model.StepData
		if dbErr := t.Transfer.Update(t.Db); dbErr != nil {
			return 0, &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	if e := checkSignal(t.Ctx, t.Signals); e != nil {
		return 0, e
	}

	n, err = t.File.ReadAt(p, off)
	t.Transfer.Progress += uint64(n)
	if err != nil && err != io.EOF {
		t.Transfer.Error = model.NewTransferError(model.TeDataTransfer, err.Error())
		err = &model.PipelineError{Kind: model.KindTransfer, Cause: t.Transfer.Error}
	}
	if dbErr := t.Transfer.Update(t.Db); dbErr != nil {
		return 0, &model.PipelineError{Kind: model.KindDatabase}
	}
	return n, err
}

// WriteAt writes the given bytes to the stream, starting at the given offset.
func (t *TransferStream) WriteAt(p []byte, off int64) (n int, err error) {
	if t.Transfer.Step == model.StepPreTasks {
		t.Transfer.Step = model.StepData
		if dbErr := t.Transfer.Update(t.Db); dbErr != nil {
			return 0, &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	if e := checkSignal(t.Ctx, t.Signals); e != nil {
		return 0, e
	}

	n, err = t.File.WriteAt(p, off)
	t.Transfer.Progress += uint64(n)
	if err != nil {
		t.Transfer.Error = model.NewTransferError(model.TeDataTransfer, err.Error())
		err = &model.PipelineError{Kind: model.KindTransfer, Cause: t.Transfer.Error}
	}
	if dbErr := t.Transfer.Update(t.Db); dbErr != nil {
		return 0, &model.PipelineError{Kind: model.KindDatabase}
	}
	return n, err
}

// Finalize closes the file, and then (if the file is the transfer's destination)
// moves the file from the temporary directory to its final destination.
// The method returns an error if the file cannot be move.
func (t *TransferStream) Finalize() *model.PipelineError {
	_ = t.File.Close()

	if !t.Rule.IsSend {
		path := filepath.Clean(filepath.Join(t.Root, t.Rule.Path, t.Transfer.DestPath))
		if err := os.Rename(t.File.Name(), path); err != nil {
			return model.NewPipelineError(model.TeFinalization, err.Error())
		}
	}
	return nil
}
