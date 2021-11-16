package pipeline

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
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

func newFileStream(pipeline *Pipeline, updateInterval time.Duration, isResume bool,
) (*fileStream, *types.TransferError) {
	stream := &fileStream{
		Pipeline: pipeline,
		ticker:   time.NewTicker(updateInterval),
		progress: pipeline.TransCtx.Transfer.Progress,
	}

	if !isResume && !pipeline.TransCtx.Rule.IsSend {
		pipeline.TransCtx.Transfer.LocalPath += ".part"
		if dbErr := pipeline.UpdateTrans("local_path"); dbErr != nil {
			return nil, dbErr
		}
	}

	file, err := internal.GetFile(stream.Logger, stream.TransCtx.Rule, stream.TransCtx.Transfer)
	if err != nil {
		return nil, err
	}

	stream.file = file

	var mErr error
	if stream.TransCtx.Rule.IsSend {
		mErr = stream.machine.Transition(internal.PipelineReadState)
	} else {
		mErr = stream.machine.Transition(internal.PipelineWriteState)
	}

	if mErr != nil {
		return nil, types.NewTransferError(types.TeInternal, mErr.Error())
	}

	return stream, nil
}

func (f *fileStream) updateProgress() *types.TransferError {
	select {
	case <-f.ticker.C:
		prog := atomic.LoadUint64(&f.progress)
		if prog == f.TransCtx.Transfer.Progress {
			return nil
		}

		f.TransCtx.Transfer.Progress = prog
		if dbErr := f.DB.Update(f.TransCtx.Transfer).Cols("progression").Run(); dbErr != nil {
			f.handleError(types.TeInternal, "Failed to update transfer progress",
				dbErr.Error())

			return errDatabase
		}
	default:
	}

	return nil
}

func (f *fileStream) Read(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != internal.PipelineReadState {
		f.handleStateErr("Read", curr)

		return 0, errStateMachine
	}

	f.wg.Add(1)
	n, err := f.file.Read(p)
	atomic.AddUint64(&f.progress, uint64(n))
	f.wg.Done()

	if uErr := f.updateProgress(); uErr != nil {
		return n, uErr
	}

	if err == nil {
		return n, nil
	}

	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}

	f.handleError(types.TeDataTransfer, "Failed to read from the file stream",
		err.Error())

	return n, errRead
}

func (f *fileStream) Write(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != internal.PipelineWriteState {
		f.handleStateErr("Write", curr)

		return 0, errStateMachine
	}

	f.wg.Add(1)
	n, err := f.file.Write(p)
	atomic.AddUint64(&f.progress, uint64(n))
	f.wg.Done()

	if uErr := f.updateProgress(); uErr != nil {
		return n, uErr
	}

	if err == nil {
		return n, nil
	}

	f.handleError(types.TeDataTransfer, "Failed to write to the file stream",
		err.Error())

	return n, errWrite
}

// ReadAt reads the stream, starting at the given offset.
func (f *fileStream) ReadAt(p []byte, off int64) (int, error) {
	if curr := f.machine.Current(); curr != internal.PipelineReadState {
		f.handleStateErr("ReadAt", curr)

		return 0, errStateMachine
	}

	f.wg.Add(1)
	n, err := f.file.ReadAt(p, off)
	atomic.AddUint64(&f.progress, uint64(n))
	f.wg.Done()

	if uErr := f.updateProgress(); uErr != nil {
		return n, uErr
	}

	if err == nil {
		return n, nil
	}

	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}

	f.handleError(types.TeDataTransfer, "Failed to readAt from the file stream",
		err.Error())

	return n, errRead
}

// WriteAt writes the given bytes to the stream, starting at the given offset.
func (f *fileStream) WriteAt(p []byte, off int64) (int, error) {
	if curr := f.machine.Current(); curr != internal.PipelineWriteState {
		f.handleStateErr("WriteAt", curr)

		return 0, errStateMachine
	}

	f.wg.Add(1)
	n, err := f.file.WriteAt(p, off)
	atomic.AddUint64(&f.progress, uint64(n))
	f.wg.Done()

	if err := f.updateProgress(); err != nil {
		return n, err
	}

	if err == nil {
		return n, nil
	}

	f.handleError(types.TeDataTransfer, "Failed to writeAt to the file stream",
		err.Error())

	return n, errWrite
}

func (f *fileStream) handleStateErr(fun, currentState string) {
	f.handleError(types.TeInternal, "File stream state machine violation",
		fmt.Sprintf("cannot call %s while in state %s", fun, currentState))
}

func (f *fileStream) handleError(code types.TransferErrorCode, msg, cause string) {
	f.errOnce.Do(func() {
		if mErr := f.machine.Transition("error"); mErr != nil {
			f.Logger.Warningf("Failed to transition to state 'error': %v", mErr)
		}

		fullMsg := fmt.Sprintf("%s: %s", msg, cause)
		f.Logger.Error(fullMsg)

		go func() {
			f.ticker.Stop()

			if err := f.file.Close(); err != nil {
				f.Logger.Warningf("Failed to close transfer file: %s", err)
			}

			f.TransCtx.Transfer.Error = *types.NewTransferError(code, fmt.Sprintf("%s: %s", msg, cause))
			f.TransCtx.Transfer.Progress = f.progress

			if dbErr := f.DB.Update(f.TransCtx.Transfer).Cols("progress", "error_code",
				"error_details").Run(); dbErr != nil {
				f.Logger.Errorf("Failed to update transfer error: %s", dbErr)
			}

			f.errorTasks()

			f.TransCtx.Transfer.Status = types.StatusError
			if dbErr := f.DB.Update(f.TransCtx.Transfer).Cols("status").Run(); dbErr != nil {
				f.Logger.Errorf("Failed to update transfer status to ERROR: %s", dbErr)
			}

			if err := f.machine.Transition("in error"); err != nil {
				f.handleStateErr("ErrorDone", f.machine.Current())

				return
			}
		}()
	})
}

// close closes the file and stops the progress tracker.
func (f *fileStream) close() *types.TransferError {
	if err := f.machine.Transition("close"); err != nil {
		f.handleStateErr("close", f.machine.Current())

		return errStateMachine
	}

	f.ticker.Stop()

	stat, sErr := f.file.Stat()
	if sErr != nil {
		f.handleError(types.TeInternal, "Failed to get final file info", sErr.Error())

		return types.NewTransferError(types.TeInternal, "failed to get final file info")
	}

	if fErr := f.file.Close(); fErr != nil {
		f.Logger.Warningf("Failed to close file: %s", fErr)
	}

	f.TransCtx.Transfer.Progress = atomic.LoadUint64(&f.progress)
	f.TransCtx.Transfer.Filesize = stat.Size()

	if dbErr := f.DB.Update(f.TransCtx.Transfer).Cols("progression", "filesize").Run(); dbErr != nil {
		f.handleError(types.TeInternal, "Failed to update final transfer progress",
			dbErr.Error())

		return errDatabase
	}

	return nil
}

// move moves the file from the temporary work directory to its final destination
// (if the file is the transfer's destination). The method returns an error if
// the file cannot be moved.
func (f *fileStream) move() *types.TransferError {
	if err := f.machine.Transition("move"); err != nil {
		f.handleStateErr("move", f.machine.Current())

		return errStateMachine
	}

	if f.TransCtx.Rule.IsSend {
		return nil
	}

	file := strings.TrimSuffix(filepath.Base(f.TransCtx.Transfer.LocalPath), ".part")

	var dest string
	if f.TransCtx.Transfer.IsServer {
		dest = utils.GetPath(file, leaf(f.TransCtx.Rule.LocalDir),
			leaf(f.TransCtx.LocalAgent.ReceiveDir), branch(f.TransCtx.LocalAgent.RootDir),
			leaf(f.TransCtx.Paths.DefaultInDir), branch(f.TransCtx.Paths.GatewayHome))
	} else {
		dest = utils.GetPath(file, leaf(f.TransCtx.Rule.LocalDir),
			leaf(f.TransCtx.Paths.DefaultInDir), branch(f.TransCtx.Paths.GatewayHome))
	}

	if f.TransCtx.Transfer.LocalPath == dest {
		return nil
	}

	moveErr := types.NewTransferError(types.TeFinalization, "failed to move temp file")

	if err := internal.CreateDir(dest); err != nil {
		f.handleError(types.TeFinalization, "Failed to create destination directory",
			err.Error())

		return moveErr
	}

	if err := tasks.MoveFile(f.TransCtx.Transfer.LocalPath, dest); err != nil {
		f.handleError(types.TeFinalization, "Failed to move temp file", err.Error())

		return moveErr
	}

	f.TransCtx.Transfer.LocalPath = dest
	if err := f.DB.Update(f.TransCtx.Transfer).Cols("local_path").Run(); err != nil {
		f.handleError(types.TeInternal, "Failed to update transfer filepath", err.Error())

		return errDatabase
	}

	return nil
}

func (f *fileStream) stop() {
	f.ticker.Stop()

	if fErr := f.file.Close(); fErr != nil {
		f.Logger.Warningf("Failed to close file: %s", fErr)
	}

	f.TransCtx.Transfer.Progress = atomic.LoadUint64(&f.progress)
	if dbErr := f.DB.Update(f.TransCtx.Transfer).Cols("progression").Run(); dbErr != nil {
		f.Logger.Errorf("Failed to update transfer progress at interruption: %s", dbErr)

		return
	}
}
