package pipeline

import (
	"context"
	"io"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// TransferStream represents the Pipeline of an incoming transfer made to the
// gateway. It is a `os.File` wrapper which adds MFT operations at the stream's
// creation, during reads/writes, and at the streams closure.
type TransferStream struct {
	*os.File
	*Pipeline
	Paths
}

// NewTransferStream initialises a new stream for the given transfer. This stream
// can then be used to execute a transfer.
func NewTransferStream(ctx context.Context, logger *log.Logger, db *database.DB,
	paths Paths, trans model.Transfer) (*TransferStream, error) {

	if trans.IsServer {
		if err := TransferInCount.add(); err != nil {
			logger.Error("Incoming transfer limit reached")
			return nil, err
		}
	} else {
		if err := TransferOutCount.add(); err != nil {
			logger.Error("Outgoing transfer limit reached")
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
			DB:       db,
			Logger:   logger,
			Transfer: &trans,
			Ctx:      ctx,
		},
		Paths: paths,
	}

	t.Pipeline.Rule = &model.Rule{ID: trans.RuleID}
	if err := t.DB.Get(t.Rule); err != nil {
		logger.Criticalf("Failed to retrieve transfer rule: %s", err.Error())
		return nil, &model.PipelineError{Kind: model.KindDatabase}
	}
	t.Signals = Signals.Add(t.Transfer.ID)

	t.proc = &tasks.Processor{
		DB:       t.DB,
		Logger:   t.Logger,
		Rule:     t.Rule,
		Transfer: t.Transfer,
		Signals:  t.Signals,
		Ctx:      ctx,
		InPath:   utils.GetPath(t.Pipeline.Rule.InPath, paths.ServerRoot, paths.InDirectory, paths.GatewayHome),
		OutPath:  utils.GetPath(t.Pipeline.Rule.OutPath, paths.ServerRoot, paths.OutDirectory, paths.GatewayHome),
	}
	if err := t.setTrueFilepath(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TransferStream) setTrueFilepath() *model.PipelineError {

	if t.Rule.IsSend {
		path := t.Transfer.SourceFile
		if t.Rule.OutPath != "" {
			path = utils.SlashJoin(t.Rule.OutPath, path)
			if t.Paths.ServerRoot != "" {
				path = utils.SlashJoin(t.Paths.ServerRoot, path)
			} else {
				path = utils.SlashJoin(t.Paths.GatewayHome, path)
			}
		} else {
			if t.Paths.ServerRoot != "" {
				path = utils.SlashJoin(t.Paths.ServerRoot, path)
			} else {
				path = utils.SlashJoin(t.Paths.OutDirectory, path)
			}
		}
		t.Transfer.TrueFilepath = path
	} else {
		path := t.Transfer.DestFile
		if t.Rule.WorkPath != "" {
			path = utils.SlashJoin(t.Rule.WorkPath, path)
			if t.Paths.ServerRoot != "" {
				path = utils.SlashJoin(t.Paths.ServerRoot, path)
			} else {
				path = utils.SlashJoin(t.Paths.GatewayHome, path)
			}
		} else {
			if t.Paths.ServerWork != "" {
				path = utils.SlashJoin(t.Paths.ServerWork, path)
			} else {
				path = utils.SlashJoin(t.Paths.WorkDirectory, path)
			}
		}
		t.Transfer.TrueFilepath = path
	}
	if err := t.Transfer.Update(t.DB); err != nil {
		t.Logger.Criticalf("Failed to update transfer filepath: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}
	return nil
}

// Start opens/creates the stream's local file. If necessary, the method also
// creates the file's parent directories.
func (t *TransferStream) Start() (err *model.PipelineError) {
	if oldStep := t.Transfer.Step; oldStep != "" {
		defer func() {
			if err == nil {
				t.Transfer.Step = oldStep
			}
		}()
	}
	t.Transfer.Step = model.StepSetup
	if err := t.Transfer.Update(t.DB); err != nil {
		t.Logger.Criticalf("Failed to update transfer step to 'SETUP': %s", err)
		return &model.PipelineError{Kind: model.KindDatabase}
	}

	if !t.Rule.IsSend {
		if err := makeDir(t.Transfer.TrueFilepath); err != nil {
			t.Logger.Errorf("Failed to create temp directory: %s", err)
			return model.NewPipelineError(model.TeForbidden, err.Error())
		}
	}

	t.File, err = getFile(t.Logger, t.Rule, t.Transfer)
	return
}

func (t *TransferStream) Read(p []byte) (n int, err error) {
	if t.Transfer.Step == model.StepPreTasks {
		t.Transfer.Step = model.StepData
		if dbErr := t.Transfer.Update(t.DB); dbErr != nil {
			return 0, &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	if e := checkSignal(t.Ctx, t.Signals); e != nil {
		return 0, e
	}

	n, err = t.File.Read(p)
	t.Transfer.Progress += uint64(n)
	if err := t.Transfer.Update(t.DB); err != nil {
		return 0, err
	}
	return
}

func (t *TransferStream) Write(p []byte) (n int, err error) {
	if t.Transfer.Step == model.StepPreTasks {
		t.Transfer.Step = model.StepData
		if dbErr := t.Transfer.Update(t.DB); dbErr != nil {
			return 0, &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	if e := checkSignal(t.Ctx, t.Signals); e != nil {
		return 0, e
	}

	n, err = t.File.Write(p)
	t.Transfer.Progress += uint64(n)
	if err := t.Transfer.Update(t.DB); err != nil {
		return 0, err
	}
	return
}

// ReadAt reads the stream, starting at the given offset.
func (t *TransferStream) ReadAt(p []byte, off int64) (n int, err error) {
	if t.Transfer.Step == model.StepPreTasks {
		t.Transfer.Step = model.StepData
		if dbErr := t.Transfer.Update(t.DB); dbErr != nil {
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
	if dbErr := t.Transfer.Update(t.DB); dbErr != nil {
		return 0, &model.PipelineError{Kind: model.KindDatabase}
	}
	return n, err
}

// WriteAt writes the given bytes to the stream, starting at the given offset.
func (t *TransferStream) WriteAt(p []byte, off int64) (n int, err error) {
	if t.Transfer.Step == model.StepPreTasks {
		t.Transfer.Step = model.StepData
		if dbErr := t.Transfer.Update(t.DB); dbErr != nil {
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
	if dbErr := t.Transfer.Update(t.DB); dbErr != nil {
		return 0, &model.PipelineError{Kind: model.KindDatabase}
	}
	return n, err
}

// Finalize closes the file, and then (if the file is the transfer's destination)
// moves the file from the temporary directory to its final destination.
// The method returns an error if the file cannot be move.
func (t *TransferStream) Finalize() *model.PipelineError {
	if !t.Rule.IsSend {
		path := t.Transfer.DestFile
		if t.Rule.InPath != "" {
			path = utils.SlashJoin(t.Rule.InPath, path)
			if t.Paths.ServerRoot != "" {
				path = utils.SlashJoin(t.Paths.ServerRoot, path)
			} else {
				path = utils.SlashJoin(t.Paths.GatewayHome, path)
			}
		} else {
			if t.Paths.ServerRoot != "" {
				path = utils.SlashJoin(t.Paths.ServerRoot, path)
			} else {
				path = utils.SlashJoin(t.Paths.InDirectory, path)
			}
		}

		if t.Transfer.TrueFilepath == path {
			return nil
		}

		if err := makeDir(path); err != nil {
			t.Logger.Errorf("Failed to create destination directory: %s", err.Error())
			return model.NewPipelineError(model.TeFinalization, err.Error())
		}
		dest, err := os.Create(path)
		if err != nil {
			t.Logger.Errorf("Failed to create destination file: %s", err.Error())
			return model.NewPipelineError(model.TeFinalization, err.Error())
		}

		if _, err := io.Copy(dest, t.File); err != nil {
			t.Logger.Errorf("Failed to copy temp file: %s", err.Error())
			return model.NewPipelineError(model.TeFinalization, err.Error())
		}

		t.Transfer.TrueFilepath = path
		if err := t.Transfer.Update(t.DB); err != nil {
			t.Logger.Errorf("Failed to update transfer filepath: %s", err.Error())
			return &model.PipelineError{Kind: model.KindDatabase}
		}

		if err := dest.Close(); err != nil {
			t.Logger.Warningf("Failed to close destination file: %s", err.Error())
		}
		if err := t.File.Close(); err != nil {
			t.Logger.Warningf("Failed to close work file: %s", err.Error())
		}
		if err := os.Remove(t.File.Name()); err != nil {
			t.Logger.Warningf("Failed to delete work file: %s", err.Error())
		}
	}
	return nil
}
