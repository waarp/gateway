package pipeline

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync/atomic"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/statemachine"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	errRead  = types.NewTransferError(types.TeDataTransfer, "failed to read data")
	errWrite = types.NewTransferError(types.TeDataTransfer, "failed to write data")
)

// FileStream represents the Pipeline of an incoming transfer made to the
// gateway. It is a `os.File` wrapper which adds MFT operations at the stream's
// creation, during reads/writes, and at the streams closure.
type FileStream struct {
	*Pipeline

	file fs.File
}

func newFileStream(pipeline *Pipeline, isResume bool) (*FileStream, *types.TransferError) {
	stream := &FileStream{
		Pipeline: pipeline,
	}

	if !isResume && !pipeline.TransCtx.Rule.IsSend {
		pipeline.TransCtx.Transfer.LocalPath.Path += ".part"
		if err := pipeline.updateTrans(); err != nil {
			return nil, err
		}
	}

	file, err := stream.getFile()
	if err != nil {
		return nil, err
	}

	stream.file = file

	return stream, nil
}

func (f *FileStream) updateTrans() *types.TransferError {
	return f.Pipeline.doUpdateTrans(f.handleError)
}

func (f *FileStream) updateProgress(n int) *types.TransferError {
	atomic.AddInt64(&f.TransCtx.Transfer.Progress, int64(n))

	return f.updateTrans()
}

func (f *FileStream) Read(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != stateReading {
		f.handleStateErr("Read", curr)

		return 0, errStateMachine
	}

	n, err := f.file.Read(p)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err != nil {
		if errors.Is(err, io.EOF) {
			return n, io.EOF
		}

		f.handleError(types.TeDataTransfer, "Failed to read from the file stream",
			err.Error())

		return n, errRead
	}

	if f.Trace.OnRead != nil {
		if testErr := wrapTestError(f.Trace.OnRead(f.TransCtx.Transfer.Progress)); testErr != nil {
			f.handleError(testErr.Code, "test error", testErr.Details)

			return n, testErr
		}
	}

	return n, nil
}

func (f *FileStream) Write(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != stateWriting {
		f.handleStateErr("Write", curr)

		return 0, errStateMachine
	}

	n, err := fs.WriteFile(f.file, p)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err != nil {
		f.handleError(types.TeDataTransfer, "Failed to write to the file stream",
			err.Error())

		return n, errWrite
	}

	if f.Trace.OnWrite != nil {
		if testErr := wrapTestError(f.Trace.OnWrite(f.TransCtx.Transfer.Progress)); testErr != nil {
			f.handleError(testErr.Code, "test error", testErr.Details)

			return n, testErr
		}
	}

	return n, nil
}

// ReadAt reads the stream, starting at the given offset.
func (f *FileStream) ReadAt(p []byte, off int64) (int, error) {
	if curr := f.machine.Current(); curr != stateReading {
		f.handleStateErr("ReadAt", curr)

		return 0, errStateMachine
	}

	n, err := fs.ReadAtFile(f.file, p, off)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err != nil {
		if errors.Is(err, io.EOF) {
			return n, io.EOF
		}

		f.handleError(types.TeDataTransfer, "Failed to readAt from the file stream",
			err.Error())

		return n, errRead
	}

	if f.Trace.OnRead != nil {
		if testErr := wrapTestError(f.Trace.OnRead(off)); testErr != nil {
			f.handleError(testErr.Code, "test error", testErr.Details)

			return n, testErr
		}
	}

	return n, nil
}

// WriteAt writes the given bytes to the stream, starting at the given offset.
func (f *FileStream) WriteAt(p []byte, off int64) (int, error) {
	if curr := f.machine.Current(); curr != stateWriting {
		f.handleStateErr("WriteAt", curr)

		return 0, errStateMachine
	}

	n, err := fs.WriteAtFile(f.file, p, off)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err != nil {
		f.handleError(types.TeDataTransfer, "Failed to writeAt to the file stream",
			err.Error())

		return n, errWrite
	}

	if f.Trace.OnWrite != nil {
		if testErr := wrapTestError(f.Trace.OnWrite(off)); testErr != nil {
			f.handleError(testErr.Code, "test error", testErr.Details)

			return n, testErr
		}
	}

	return n, nil
}

// Seek changes the file's current offset to the given one. Any subsequent call
// to Read or Write will be made from that new offset. The transfer's progress
// will also be changed to this new offset.
func (f *FileStream) Seek(off int64, whence int) (int64, error) {
	if curr := f.machine.Current(); curr != stateWriting && curr != stateReading {
		f.handleStateErr("Seek", curr)

		return 0, errStateMachine
	}

	newOff, seekErr := fs.SeekFile(f.file, off, whence)
	if seekErr != nil {
		f.handleError(types.TeInternal, "Failed to seek in file", seekErr.Error())

		return 0, types.NewTransferError(types.TeInternal, "failed to seek in file")
	}

	f.TransCtx.Transfer.Progress = newOff

	if updErr := f.UpdateTrans(); updErr != nil {
		return newOff, updErr
	}

	return newOff, nil
}

// Sync commits the current contents of the file to stable storage (if the
// storage supports it). It also forces a database update, even if the update
// timer has not ticked yet (it differs from Pipeline.UpdateTrans in that aspect).
func (f *FileStream) Sync() error {
	if curr := f.machine.Current(); curr != stateWriting && curr != stateReading {
		f.handleStateErr("Seek", curr)

		return errStateMachine
	}

	// If the fs supports it, we sync the file.
	if err := fs.SyncFile(f.file); err != nil && !errors.Is(err, fs.ErrNotImplemented) {
		f.handleError(types.TeInternal, "Failed to sync file", err.Error())

		return types.NewTransferError(types.TeInternal, "failed to sync file")
	}

	// Reset the update timer since we force an update.
	f.updTicker.Reset(TransferUpdateInterval)

	// Force an immediate database update.
	if dbErr := f.DB.Update(f.TransCtx.Transfer).Run(); dbErr != nil {
		f.handleError(types.TeInternal, "Failed to update transfer",
			dbErr.Error())

		return ErrDatabase
	}

	return nil
}

func (f *FileStream) handleStateErr(fun string, currentState statemachine.State) {
	f.handleError(types.TeInternal, "File stream state machine violation",
		fmt.Sprintf("cannot call %s while in state %s", fun, currentState))
}

func (f *FileStream) handleError(code types.TransferErrorCode, msg, cause string) {
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
func (f *FileStream) close() *types.TransferError {
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

	if err := testError(f.Trace.OnClose); err != nil {
		f.handleError(types.TeInternal, "Failed to get final file info", err.Error())

		return err
	}

	return nil
}

// move the file from the temporary work directory to its final destination
// (if the file is the transfer's destination). The method returns an error if
// the file cannot be moved.
//
//nolint:funlen //no easy way to split the function
func (f *FileStream) move() *types.TransferError {
	if curr := f.machine.Current(); curr != stateDataEnd {
		f.handleStateErr("move", f.machine.Current())

		return errStateMachine
	}

	moveErr := types.NewTransferError(types.TeFinalization, "failed to move temp file")

	if f.TransCtx.Rule.IsSend {
		return nil
	}

	destFilename := f.TransCtx.Transfer.DestFilename
	if destFilename == "" {
		destFilename = f.TransCtx.Transfer.SrcFilename
	}

	var (
		dstURL  *url.URL
		pathErr error
	)

	if f.TransCtx.Transfer.IsServer() {
		dstURL, pathErr = utils.GetPath(destFilename, leaf(f.TransCtx.Rule.LocalDir),
			leaf(f.TransCtx.LocalAgent.ReceiveDir), branch(f.TransCtx.LocalAgent.RootDir),
			leaf(f.TransCtx.Paths.DefaultInDir), branch(f.TransCtx.Paths.GatewayHome))
	} else {
		dstURL, pathErr = utils.GetPath(destFilename, leaf(f.TransCtx.Rule.LocalDir),
			leaf(f.TransCtx.Paths.DefaultInDir), branch(f.TransCtx.Paths.GatewayHome))
	}

	if pathErr != nil {
		return types.NewTransferError(types.TeFileNotFound, pathErr.Error())
	}

	dest := (*types.URL)(dstURL)

	if f.TransCtx.Transfer.LocalPath.String() == dest.String() {
		return nil
	}

	dstFS, fsErr := fs.GetFileSystem(f.DB, dest)
	if fsErr != nil {
		f.handleError(types.TeFinalization, "Failed to instantiate destination file system",
			fsErr.Error())

		return moveErr
	}

	if err := createDir(dstFS, dest); err != nil {
		f.handleError(types.TeFinalization, "Failed to create destination directory",
			err.Error())

		return moveErr
	}

	newFS, movErr := tasks.MoveFile(f.DB, f.TransCtx.FS, &f.TransCtx.Transfer.LocalPath, dest)
	if movErr != nil {
		f.handleError(types.TeFinalization, "Failed to move temp file", movErr.Error())

		return moveErr
	}

	f.TransCtx.FS = newFS
	f.TransCtx.Transfer.LocalPath = *dest

	if dbErr := f.updateTrans(); dbErr != nil {
		return dbErr
	}

	if err := testError(f.Trace.OnMove); err != nil {
		f.handleError(types.TeInternal, "Failed to get final file info", err.Error())

		return err
	}

	return nil
}

func (f *FileStream) stop() {
	if fErr := f.file.Close(); fErr != nil {
		f.Logger.Warning("Failed to close file: %s", fErr)
	}
}
