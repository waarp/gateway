package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

type progressReporter struct {
	*time.Ticker
	dbErr chan error
	done  chan struct{}
}

func (p *progressReporter) Run(trans *model.Transfer, db *database.DB, logger *log.Logger) {
	p.Ticker = time.NewTicker(time.Second)
	p.dbErr = make(chan error)
	p.done = make(chan struct{})
	upd := *trans

	go func() {
		defer close(p.dbErr)
		for {
			select {
			case <-p.done:
				return
			case <-p.C:
			}

			upd.Progress = atomic.LoadUint64(&trans.Progress)
			if err := db.Update(&upd); err != nil {
				logger.Criticalf("Failed to update transfer progress: %s", err.Error())
				p.dbErr <- err
				return
			}
		}
	}()
}

func (p *progressReporter) stop() {
	p.Ticker.Stop()
	close(p.done)
}

// TransferStream represents the Pipeline of an incoming transfer made to the
// gateway. It is a `os.File` wrapper which adds MFT operations at the stream's
// creation, during reads/writes, and at the streams closure.
type TransferStream struct {
	*os.File
	*Pipeline
	Paths
	*progressReporter
}

// getOldTransfer searches if the given transfer has a corresponding entry in
// the database. If it does, the given transfer will be replaced by the one from
// the database. If no corresponding entry can be found, a new one will be created.
func (t *TransferStream) getOldTransfer() *model.PipelineError {
	// If an ID is present, this is a client transfer, and all info are already
	// present, no need to query the database.
	if t.Transfer.ID != 0 {
		return nil
	}

	// If no RemoteTransferID was given, then there is no old entry to retrieve,
	// creates a new entry instead.
	if t.Transfer.RemoteTransferID == "" {
		return t.createTransfer(t.Transfer)
	}

	// Search a transfer with the given RemoteTransferID.
	getTrans := &model.Transfer{
		RemoteTransferID: t.Transfer.RemoteTransferID,
		AccountID:        t.Transfer.AccountID,
	}
	if err := t.DB.Get(getTrans); err != nil && err != database.ErrNotFound {
		t.Logger.Criticalf("Failed to retrieve transfer: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	} else if err == nil {
		t.Transfer = getTrans
		return nil
	}

	// If no transfer entry is found, then create a new one instead.
	return t.createTransfer(t.Transfer)
}

func countTransfer(trans *model.Transfer, logger *log.Logger) error {
	if trans.IsServer {
		if err := TransferInCount.add(); err != nil {
			logger.Error("Incoming transfer limit reached")
			return err
		}
	} else {
		if err := TransferOutCount.add(); err != nil {
			logger.Error("Outgoing transfer limit reached")
			return err
		}
	}
	return nil
}

// NewTransferStream initialises a new stream for the given transfer. This stream
// can then be used to execute a transfer.
func NewTransferStream(ctx context.Context, logger *log.Logger, db *database.DB,
	paths Paths, trans *model.Transfer) (*TransferStream, error) {
	if err := countTransfer(trans, logger); err != nil {
		return nil, err
	}

	t := &TransferStream{
		Pipeline: &Pipeline{
			DB:       db,
			Logger:   logger,
			Transfer: trans,
			Ctx:      ctx,
		},
		Paths: paths,
	}

	if err := t.getOldTransfer(); err != nil {
		return nil, err
	}
	t.Logger = log.NewLogger(fmt.Sprintf("Pipeline %d", trans.ID))

	if t.Transfer.Error.Code != types.TeOk {
		t.Transfer.Error = types.TransferError{}
		if err := db.Update(t.Transfer); err != nil {
			logger.Criticalf("Failed to reset transfer error: %s", err.Error())
			return nil, &model.PipelineError{Kind: model.KindDatabase}
		}
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
		InPath: utils.GetPath(t.Pipeline.Rule.InPath, utils.Elems{
			{paths.ServerRoot, false},
			{paths.InDirectory, true},
			{paths.GatewayHome, false},
		}),
		OutPath: utils.GetPath(t.Pipeline.Rule.OutPath, utils.Elems{
			{paths.ServerRoot, false},
			{paths.OutDirectory, true},
			{paths.GatewayHome, false},
		}),
	}
	if err := t.setTrueFilepath(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TransferStream) createTransfer(trans *model.Transfer) *model.PipelineError {

	if err := t.DB.Create(trans); err != nil {
		if _, ok := err.(*database.ErrInvalid); ok {
			t.Logger.Errorf("Failed to create transfer entry: %s", err.Error())
			return model.NewPipelineError(types.TeForbidden, err.Error())
		}
		t.Logger.Criticalf("Failed to create transfer entry: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}
	t.Logger.Infof("Transfer was given ID n°%d", trans.ID)
	return nil
}

func (t *TransferStream) setTrueFilepath() *model.PipelineError {
	if t.Rule.IsSend {
		fullPath := utils.GetPath(t.Transfer.SourceFile, utils.Elems{
			{t.Rule.OutPath, true},
			{t.Paths.ServerOut, true},
			{t.Paths.ServerRoot, false},
			{t.Paths.OutDirectory, true},
			{t.Paths.GatewayHome, false},
		})
		t.Transfer.TrueFilepath = fullPath
	} else {
		fullPath := utils.GetPath(t.Transfer.DestFile, utils.Elems{
			{t.Rule.WorkPath, true},
			{t.Paths.ServerWork, true},
			{t.Paths.ServerRoot, false},
			{t.Paths.WorkDirectory, true},
			{t.Paths.GatewayHome, false},
		})
		t.Transfer.TrueFilepath = fullPath + ".tmp"
	}
	if err := t.DB.Update(t.Transfer); err != nil {
		t.Logger.Criticalf("Failed to update transfer filepath: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}
	return nil
}

// Start opens/creates the stream's local file. If necessary, the method also
// creates the file's parent directories.
func (t *TransferStream) Start() *model.PipelineError {
	if t.File != nil {
		return nil
	}

	if !t.Rule.IsSend {
		if err := makeDir(t.Transfer.TrueFilepath); err != nil {
			t.Logger.Errorf("Failed to create temp directory: %s", err)
			return model.NewPipelineError(types.TeForbidden, err.Error())
		}
	}

	var err *model.PipelineError
	if t.File, err = getFile(t.Logger, t.Rule, t.Transfer); err != nil {
		return err
	}

	t.progressReporter = &progressReporter{}
	t.progressReporter.Run(t.Transfer, t.DB, t.Logger)

	return nil
}

func (t *TransferStream) Read(p []byte) (n int, err error) {
	off, err := t.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	return t.ReadAt(p, off)
}

func (t *TransferStream) Write(p []byte) (n int, err error) {
	off, err := t.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	return t.WriteAt(p, off)
}

// ReadAt reads the stream, starting at the given offset.
func (t *TransferStream) ReadAt(p []byte, off int64) (n int, err error) {
	if t.Transfer.Step == types.StepPreTasks {
		t.Transfer.TaskNumber = 0
		t.Transfer.Step = types.StepData
		if dbErr := t.DB.Update(t.Transfer); dbErr != nil {
			t.Logger.Criticalf("Failed to update upload transfer step to 'DATA': %s", dbErr)
			return 0, &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	if e := checkSignal(t.Ctx, t.Signals, t.dbErr); e != nil {
		return 0, e
	}

	n, err = t.File.ReadAt(p, off)
	atomic.AddUint64(&t.Transfer.Progress, uint64(n))
	if err != nil && err != io.EOF {
		t.Transfer.Error = types.NewTransferError(types.TeDataTransfer, err.Error())
		err = &model.PipelineError{Kind: model.KindTransfer, Cause: t.Transfer.Error}
	}

	return n, err
}

// WriteAt writes the given bytes to the stream, starting at the given offset.
func (t *TransferStream) WriteAt(p []byte, off int64) (n int, err error) {
	if t.Transfer.Step == types.StepPreTasks {
		t.Transfer.TaskNumber = 0
		t.Transfer.Step = types.StepData
		if dbErr := t.DB.Update(t.Transfer); dbErr != nil {
			t.Logger.Criticalf("Failed to update download transfer step to 'DATA': %s", dbErr)
			return 0, &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	if e := checkSignal(t.Ctx, t.Signals, t.dbErr); e != nil {
		return 0, e
	}

	n, err = t.File.WriteAt(p, off)
	atomic.AddUint64(&t.Transfer.Progress, uint64(n))
	if err != nil {
		t.Transfer.Error = types.NewTransferError(types.TeDataTransfer, err.Error())
		err = &model.PipelineError{Kind: model.KindTransfer, Cause: t.Transfer.Error}
	}

	return n, err
}

// Close closes the file and stops the progress tracker.
func (t *TransferStream) Close() error {
	if err := t.File.Close(); err != nil {
		t.Logger.Warningf("Failed to close file '%s': %s", t.File.Name(),
			err.(*os.PathError).Err.Error())
	}

	t.progressReporter.stop()
	if err := t.DB.Update(t.Transfer); err != nil {
		t.Logger.Criticalf("Failed to update transfer progress: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}

	return nil
}

// Move moves the file from the temporary work directory to its final destination
// (if the file is the transfer's destination). The method returns an error if
// the file cannot be moved.
func (t *TransferStream) Move() error {
	if t.Rule.IsSend {
		return nil
	}

	filepath := utils.GetPath(t.Transfer.DestFile, utils.Elems{
		{t.Rule.InPath, true},
		{t.Paths.ServerIn, true},
		{t.Paths.ServerRoot, false},
		{t.Paths.InDirectory, true},
		{t.Paths.GatewayHome, false},
	})

	if t.Transfer.TrueFilepath == filepath || t.Transfer.TrueFilepath == "" {
		return nil
	}

	if err := makeDir(filepath); err != nil {
		t.Logger.Errorf("Failed to create destination directory: %s", err.Error())
		return model.NewPipelineError(types.TeFinalization, err.Error())
	}

	if err := tasks.MoveFile(t.Transfer.TrueFilepath, filepath); err != nil {
		t.Logger.Errorf("Failed to move temp file: %s", err.Error())
		return model.NewPipelineError(types.TeFinalization, err.Error())
	}

	t.Transfer.TrueFilepath = filepath
	if err := t.DB.Update(t.Transfer); err != nil {
		t.Logger.Errorf("Failed to update transfer filepath: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}
	return nil
}
