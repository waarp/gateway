package pipeline

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/statemachine"
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

	file *os.File
}

func newFileStream(pipeline *Pipeline, isResume bool) (*fileStream, *types.TransferError) {
	stream := &fileStream{
		Pipeline: pipeline,
	}

	if !isResume && !pipeline.TransCtx.Rule.IsSend {
		pipeline.TransCtx.Transfer.LocalPath += ".part"
		if dbErr := pipeline.UpdateTrans(); dbErr != nil {
			return nil, dbErr
		}
	}

	file, err := stream.getFile()
	if err != nil {
		return nil, err
	}

	stream.file = file

	return stream, nil
}

func (f *fileStream) updateTrans() *types.TransferError {
	return f.Pipeline.doUpdateTrans(f.handleError)
}

func (f *fileStream) updateProgress(n int) *types.TransferError {
	atomic.AddInt64(&f.TransCtx.Transfer.Progress, int64(n))

	return f.updateTrans()
}

func (f *fileStream) Read(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != stateReading {
		f.handleStateErr("Read", curr)

		return 0, errStateMachine
	}

	n, err := f.file.Read(p)
	if uErr := f.updateProgress(n); uErr != nil {
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
	if curr := f.machine.Current(); curr != stateWriting {
		f.handleStateErr("Write", curr)

		return 0, errStateMachine
	}

	n, err := f.file.Write(p)
	if uErr := f.updateProgress(n); uErr != nil {
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
	if curr := f.machine.Current(); curr != stateReading {
		f.handleStateErr("ReadAt", curr)

		return 0, errStateMachine
	}

	n, err := f.file.ReadAt(p, off)
	if uErr := f.updateProgress(n); uErr != nil {
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
	if curr := f.machine.Current(); curr != stateWriting {
		f.handleStateErr("WriteAt", curr)

		return 0, errStateMachine
	}

	n, err := f.file.WriteAt(p, off)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err == nil {
		return n, nil
	}

	f.handleError(types.TeDataTransfer, "Failed to writeAt to the file stream",
		err.Error())

	return n, errWrite
}

func (f *fileStream) handleStateErr(fun string, currentState statemachine.State) {
	f.handleError(types.TeInternal, "File stream state machine violation",
		fmt.Sprintf("cannot call %s while in state %s", fun, currentState))
}

func (f *fileStream) handleError(code types.TransferErrorCode, msg, cause string) {
	f.errOnce.Do(func() {
		if mErr := f.machine.Transition(stateError); mErr != nil {
			f.Logger.Warning("Failed to transition to 'error' state: %v", mErr)
		}

		if err := f.file.Close(); err != nil {
			f.Logger.Warning("Failed to close transfer file: %s", err)
		}

		f.errDo(code, msg, cause)
	})
}

// close the file and stop the progress tracker.
func (f *fileStream) close() *types.TransferError {
	if curr := f.machine.Current(); curr != stateDataEnd {
		f.handleStateErr("close", f.machine.Current())

		return errStateMachine
	}

	stat, sErr := f.file.Stat()
	if sErr != nil {
		f.handleError(types.TeInternal, "Failed to get final file info", sErr.Error())

		return types.NewTransferError(types.TeInternal, "failed to get final file info")
	}

	if fErr := f.file.Close(); fErr != nil {
		f.Logger.Warning("Failed to close file: %s", fErr)
	}

	f.TransCtx.Transfer.Filesize = stat.Size()

	if dbErr := f.updateTrans(); dbErr != nil {
		return dbErr
	}

	return nil
}

// move the file from the temporary work directory to its final destination
// (if the file is the transfer's destination). The method returns an error if
// the file cannot be moved.
func (f *fileStream) move() *types.TransferError {
	if curr := f.machine.Current(); curr != stateDataEnd {
		f.handleStateErr("move", f.machine.Current())

		return errStateMachine
	}

	moveErr := types.NewTransferError(types.TeFinalization, "failed to move temp file")

	if f.TransCtx.Rule.IsSend {
		return nil
	}

	file, err := filepath.Rel(makeLocalDir(f.TransCtx), f.TransCtx.Transfer.LocalPath)
	if err != nil {
		f.handleError(types.TeInternal, "could not split the filename from the file path", err.Error())

		return moveErr
	}

	file = strings.TrimSuffix(file, ".part")

	var dest string
	if f.TransCtx.Transfer.IsServer() {
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

	if err := createDir(dest); err != nil {
		f.handleError(types.TeFinalization, "Failed to create destination directory",
			err.Error())

		return moveErr
	}

	if err := tasks.MoveFile(f.TransCtx.Transfer.LocalPath, dest); err != nil {
		f.handleError(types.TeFinalization, "Failed to move temp file", err.Error())

		return moveErr
	}

	f.TransCtx.Transfer.LocalPath = dest
	if dbErr := f.updateTrans(); dbErr != nil {
		return dbErr
	}

	return nil
}

func (f *fileStream) stop() {
	if fErr := f.file.Close(); fErr != nil {
		f.Logger.Warning("Failed to close file: %s", fErr)
	}
}
