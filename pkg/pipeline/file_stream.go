package pipeline

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

var (
	errRead  = types.NewTransferError(types.TeDataTransfer, "failed to read data")
	errWrite = types.NewTransferError(types.TeDataTransfer, "failed to write data")
)

// fileStream represents the Pipeline of an incoming transfer made to the
// gateway. It is a `os.File` wrapper which adds MFT operations at the stream's
// creation, during reads/writes, and at the streams closure.
type fileStream struct {
	*Pipeline
	wg sync.WaitGroup

	file *os.File

	ticker   *time.Ticker
	progress uint64
}

func newFileStream(pipeline *Pipeline, updateInterval time.Duration) (*fileStream, error) {

	stream := &fileStream{
		Pipeline: pipeline,
		ticker:   time.NewTicker(updateInterval),
		progress: pipeline.transCtx.Transfer.Progress,
	}

	file, err := internal.GetFile(stream.logger, stream.transCtx.Rule, stream.transCtx.Transfer)
	if err != nil {
		return nil, err
	}
	stream.file = file

	if stream.transCtx.Rule.IsSend {
		err = stream.machine.Transition("reading")
	} else {
		err = stream.machine.Transition("writing")
	}

	return stream, nil
}

func (f *fileStream) updateProgress() error {
	select {
	case <-f.ticker.C:
		f.transCtx.Transfer.Progress = atomic.LoadUint64(&f.progress)
		if dbErr := f.db.Update(f.transCtx.Transfer).Cols("progression").Run(); dbErr != nil {
			f.handleError(types.TeInternal, "Failed to update transfer progress",
				dbErr.Error())
			return errDatabase
		}
	default:
	}
	return nil
}

func (f *fileStream) Read(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != "reading" {
		f.handleStateErr("Read", curr)
		return 0, errStateMachine
	}

	f.wg.Add(1)
	n, err := f.file.Read(p)
	atomic.AddUint64(&f.progress, uint64(n))
	if err := f.updateProgress(); err != nil {
		return n, err
	}
	f.wg.Done()

	if err != nil && err != io.EOF {
		f.handleError(types.TeDataTransfer, "Failed to read from the file stream",
			err.Error())
		return n, errRead
	}

	return n, err
}

func (f *fileStream) Write(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != "writing" {
		f.handleStateErr("Write", curr)
		return 0, errStateMachine
	}

	f.wg.Add(1)
	n, err := f.file.Write(p)
	atomic.AddUint64(&f.progress, uint64(n))
	if err := f.updateProgress(); err != nil {
		return n, err
	}
	f.wg.Done()

	if err != nil {
		f.handleError(types.TeDataTransfer, "Failed to write to the file stream",
			err.Error())
		return n, errWrite
	}

	return n, err
}

// ReadAt reads the stream, starting at the given offset.
func (f *fileStream) ReadAt(p []byte, off int64) (int, error) {
	if curr := f.machine.Current(); curr != "reading" {
		f.handleStateErr("ReadAt", curr)
		return 0, errStateMachine
	}

	f.wg.Add(1)
	n, err := f.file.ReadAt(p, off)
	atomic.AddUint64(&f.progress, uint64(n))
	if err := f.updateProgress(); err != nil {
		return n, err
	}
	f.wg.Done()

	if err != nil && err != io.EOF {
		f.handleError(types.TeDataTransfer, "Failed to readAt from the file stream",
			err.Error())
		return n, errRead
	}

	return n, err
}

// WriteAt writes the given bytes to the stream, starting at the given offset.
func (f *fileStream) WriteAt(p []byte, off int64) (int, error) {
	if curr := f.machine.Current(); curr != "writing" {
		f.handleStateErr("WriteAt", curr)
		return 0, errStateMachine
	}

	f.wg.Add(1)
	n, err := f.file.WriteAt(p, off)
	atomic.AddUint64(&f.progress, uint64(n))
	if err := f.updateProgress(); err != nil {
		return n, err
	}
	f.wg.Done()

	if err != nil {
		f.handleError(types.TeDataTransfer, "Failed to writeAt to the file stream",
			err.Error())
		return n, errWrite
	}

	return n, err
}

func (f *fileStream) handleStateErr(fun, currentState string) {
	f.handleError(types.TeInternal, "File stream state machine violation",
		fmt.Sprintf("cannot call %s while in state %s", fun, currentState))
}

func (f *fileStream) handleError(code types.TransferErrorCode, msg, cause string) {
	f.errOnce.Do(func() {
		_ = f.machine.Transition("error")
		fullMsg := fmt.Sprintf("%s: %s", msg, cause)
		f.logger.Error(fullMsg)

		if err := f.file.Close(); err != nil {
			f.logger.Warningf("Failed to close transfer file: %s", err)
		}

		go func() {
			internal.UpdateError(f.db, f.logger, f.transCtx.Transfer, code, fmt.Sprintf("%s: %s", msg, cause))

			f.errorTasks()

			f.transCtx.Transfer.Status = types.StatusError
			if dbErr := f.db.Update(f.transCtx.Transfer).Cols("status").Run(); dbErr != nil {
				f.logger.Errorf("Failed to update transfer status to ERROR: %s", dbErr)
			}

			if err := f.machine.Transition("in error"); err != nil {
				f.handleStateErr("ErrorDone", f.machine.Current())
				return
			}
		}()
	})
}

// close closes the file and stops the progress tracker.
func (f *fileStream) close() error {
	if err := f.machine.Transition("close"); err != nil {
		f.handleStateErr("close", f.machine.Current())
		return errStateMachine
	}
	f.ticker.Stop()

	if fErr := f.file.Close(); fErr != nil {
		f.logger.Warningf("Failed to close file: %s", fErr)
	}

	f.transCtx.Transfer.Progress = atomic.LoadUint64(&f.progress)
	if dbErr := f.db.Update(f.transCtx.Transfer).Cols("progression").Run(); dbErr != nil {
		f.handleError(types.TeInternal, "Failed to update final transfer progress",
			dbErr.Error())
		return errDatabase
	}

	return nil
}

// move moves the file from the temporary work directory to its final destination
// (if the file is the transfer's destination). The method returns an error if
// the file cannot be moved.
func (f *fileStream) move() error {
	if err := f.machine.Transition("move"); err != nil {
		f.handleStateErr("move", f.machine.Current())
		return errStateMachine
	}

	if f.transCtx.Rule.IsSend {
		return nil
	}

	file := strings.TrimRight(filepath.Base(f.transCtx.Transfer.LocalPath), ".part")
	var dest string
	if f.transCtx.Transfer.IsServer {
		dest = utils.GetPath(file, Leaf(f.transCtx.Rule.LocalDir),
			Leaf(f.transCtx.LocalAgent.LocalInDir), Branch(f.transCtx.LocalAgent.Root),
			Leaf(f.transCtx.Paths.DefaultInDir), Branch(f.transCtx.Paths.GatewayHome))
	} else {
		dest = utils.GetPath(file, Leaf(f.transCtx.Rule.LocalDir),
			Leaf(f.transCtx.Paths.DefaultInDir), Branch(f.transCtx.Paths.GatewayHome))
	}

	if f.transCtx.Transfer.LocalPath == dest {
		return nil
	}

	moveErr := types.NewTransferError(types.TeFinalization, "failed to move temp file")
	if err := internal.CreateDir(dest); err != nil {
		f.handleError(types.TeFinalization, "Failed to create destination directory",
			err.Error())
		return moveErr
	}

	if err := tasks.MoveFile(f.transCtx.Transfer.LocalPath, dest); err != nil {
		f.handleError(types.TeFinalization, "Failed to move temp file", err.Error())
		return moveErr
	}

	f.transCtx.Transfer.LocalPath = dest
	if err := f.db.Update(f.transCtx.Transfer).Cols("local_path").Run(); err != nil {
		f.handleError(types.TeInternal, "Failed to update transfer filepath", err.Error())
		return errDatabase
	}

	return nil
}

func (f *fileStream) stop() {
	f.ticker.Stop()

	if fErr := f.file.Close(); fErr != nil {
		f.logger.Warningf("Failed to close file: %s", fErr)
	}

	f.transCtx.Transfer.Progress = atomic.LoadUint64(&f.progress)
	if dbErr := f.db.Update(f.transCtx.Transfer).Cols("progression").Run(); dbErr != nil {
		f.logger.Errorf("Failed to update transfer progress at interruption: %s", dbErr)
		return
	}
}
