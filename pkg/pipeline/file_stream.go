package pipeline

import (
	"bytes"
	"errors"
	"hash"
	"io"
	"sync/atomic"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// FileStream represents the Pipeline of an incoming transfer made to the
// gateway. It is a `os.File` wrapper which adds MFT operations at the stream's
// creation, during reads/writes, and at the streams closure.
type FileStream struct {
	*Pipeline

	file fs.File
}

func newFileStream(pipeline *Pipeline, isResume bool) (*FileStream, *Error) {
	stream := &FileStream{
		Pipeline: pipeline,
	}

	if !isResume && !pipeline.TransCtx.Rule.IsSend {
		pipeline.TransCtx.Transfer.LocalPath += ".part"
		if err := pipeline.UpdateTrans(); err != nil {
			return nil, err
		}
	}

	file, err := stream.getFile()
	if err != nil {
		return nil, pipeline.internalError(err.code, err.details, err.cause)
	}

	stream.file = file

	return stream, nil
}

func (f *FileStream) updateProgress(n int) *Error {
	atomic.AddInt64(&f.TransCtx.Transfer.Progress, int64(n))

	return f.UpdateTrans()
}

func (f *FileStream) Read(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != stateReading {
		return 0, f.stateErr("Read", curr)
	}

	n, err := f.file.Read(p)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err != nil {
		if errors.Is(err, io.EOF) {
			return n, io.EOF
		}

		return n, f.internalError(types.TeInternal, "failed to read file", err)
	}

	if f.Trace.OnRead != nil {
		if testErr := f.Trace.OnRead(f.TransCtx.Transfer.Progress); testErr != nil {
			return n, f.internalErrorWithMsg(types.TeInternal, "read trace error",
				"failed to read file", testErr)
		}
	}

	return n, nil
}

func (f *FileStream) Write(p []byte) (int, error) {
	if curr := f.machine.Current(); curr != stateWriting {
		return 0, f.stateErr("Write", curr)
	}

	n, err := f.file.Write(p)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err != nil {
		return n, f.internalError(types.TeInternal, "failed to write file", err)
	}

	if f.Trace.OnWrite != nil {
		if testErr := f.Trace.OnWrite(f.TransCtx.Transfer.Progress); testErr != nil {
			return n, f.internalErrorWithMsg(types.TeInternal, "write trace error",
				"failed to write file", testErr)
		}
	}

	return n, nil
}

// ReadAt reads the stream, starting at the given offset.
func (f *FileStream) ReadAt(p []byte, off int64) (int, error) {
	if curr := f.machine.Current(); curr != stateReading {
		return 0, f.stateErr("ReadAt", curr)
	}

	n, err := f.file.ReadAt(p, off)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err != nil {
		if errors.Is(err, io.EOF) {
			return n, io.EOF
		}

		return n, f.internalError(types.TeInternal, "failed to read file", err)
	}

	if f.Trace.OnRead != nil {
		if testErr := f.Trace.OnRead(f.TransCtx.Transfer.Progress); testErr != nil {
			return n, f.internalErrorWithMsg(types.TeInternal, "read trace error",
				"failed to read file", testErr)
		}
	}

	return n, nil
}

// WriteAt writes the given bytes to the stream, starting at the given offset.
func (f *FileStream) WriteAt(p []byte, off int64) (int, error) {
	if curr := f.machine.Current(); curr != stateWriting {
		return 0, f.stateErr("WriteAt", curr)
	}

	n, err := f.file.WriteAt(p, off)
	if uErr := f.updateProgress(n); uErr != nil {
		return n, uErr
	}

	if err != nil {
		return n, f.internalError(types.TeInternal, "failed to write file", err)
	}

	if f.Trace.OnWrite != nil {
		if testErr := f.Trace.OnWrite(f.TransCtx.Transfer.Progress); testErr != nil {
			return n, f.internalErrorWithMsg(types.TeInternal, "write trace error",
				"failed to write file", testErr)
		}
	}

	return n, nil
}

// Seek changes the file's current offset to the given one. Any subsequent call
// to Read or Write will be made from that new offset. The transfer's progress
// will also be changed to this new offset.
func (f *FileStream) Seek(off int64, whence int) (int64, error) {
	if curr := f.machine.Current(); curr != stateWriting && curr != stateReading {
		return 0, f.stateErr("Seek", curr)
	}

	newOff, seekErr := f.file.Seek(off, whence)
	if seekErr != nil {
		return newOff, f.internalError(types.TeInternal, "failed to seek in file", seekErr)
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
		return f.stateErr("Sync", curr)
	}

	// If the fs supports it, we sync the file.
	if err := f.file.Sync(); err != nil {
		return f.internalError(types.TeInternal, "failed to flush file", err)
	}

	// Reset the update timer since we force an update.
	f.updTicker.Reset(TransferUpdateInterval)

	// Force an immediate database update.
	if dbErr := f.DB.Update(f.TransCtx.Transfer).Run(); dbErr != nil {
		return f.internalErrorWithMsg(types.TeInternal, "failed to update transfer",
			"database error", dbErr)
	}

	return nil
}

func (f *FileStream) Stat() (fs.FileInfo, error) {
	info, err := f.file.Stat()
	if err != nil {
		return nil, f.internalError(types.TeInternal, "failed to stat file", err)
	}

	return info, nil
}

var ErrHashMismatch = errors.New("file hash mismatch")

func (f *FileStream) CheckHash(hasher hash.Hash, expected []byte) error {
	if err := f.machine.Transition(stateHashCheck); err != nil {
		return f.stateErr("Hash", f.machine.Current())
	}

	if _, err := f.file.Seek(0, io.SeekStart); err != nil {
		return f.internalError(types.TeInternal, "failed to seek file for hashing", err)
	}

	if _, err := io.Copy(hasher, f.file); err != nil {
		return f.internalError(types.TeInternal, "failed to read file for hashing", err)
	}

	actual := hasher.Sum(nil)
	if !bytes.Equal(actual, expected) {
		return f.internalError(types.TeIntegrity,
			"file hash does not match expected value", ErrHashMismatch)
	}

	if err := f.machine.Transition(stateWriting); err != nil {
		return f.stateErr("Hash", f.machine.Current())
	}

	return nil
}

// close the file and stop the progress tracker.
func (f *FileStream) close() *Error {
	if curr := f.machine.Current(); curr != stateDataEnd {
		return f.stateErr("Close", curr)
	}

	stat, sErr := f.file.Stat()
	if sErr != nil {
		return f.internalErrorWithMsg(types.TeInternal, "failed to get final file info",
			"file check failed", sErr)
	}

	if fErr := f.file.Close(); fErr != nil {
		f.Logger.Warningf("Failed to close file: %v", fErr)
	}

	if f.TransCtx.Transfer.Filesize == model.UnknownSize {
		f.TransCtx.Transfer.Filesize = stat.Size()
	}

	if dbErr := f.UpdateTrans(); dbErr != nil {
		return dbErr
	}

	if f.Trace.OnClose != nil {
		if err := f.Trace.OnClose(); err != nil {
			return f.internalErrorWithMsg(types.TeInternal, "file close trace error",
				"file check failed", err)
		}
	}

	return nil
}

// move the file from the temporary work directory to its final destination
// (if the file is the transfer's destination). The method returns an error if
// the file cannot be moved.
//
//nolint:funlen //best keep in one function
func (f *FileStream) move() *Error {
	if curr := f.machine.Current(); curr != stateDataEnd {
		return f.stateErr("Move", curr)
	}

	if f.TransCtx.Rule.IsSend {
		return nil
	}

	destFilename := f.TransCtx.Transfer.DestFilename
	if destFilename == "" {
		destFilename = f.TransCtx.Transfer.SrcFilename
	}

	var (
		dest    string
		pathErr error
	)

	leaf, branch := utils.Leaf, utils.Branch

	if f.TransCtx.Transfer.IsServer() {
		dest, pathErr = utils.GetPath(destFilename, leaf(f.TransCtx.Rule.LocalDir),
			leaf(f.TransCtx.LocalAgent.ReceiveDir), branch(f.TransCtx.LocalAgent.RootDir),
			leaf(f.TransCtx.Paths.DefaultInDir), branch(f.TransCtx.Paths.GatewayHome))
	} else {
		dest, pathErr = utils.GetPath(destFilename, leaf(f.TransCtx.Rule.LocalDir),
			leaf(f.TransCtx.Paths.DefaultInDir), branch(f.TransCtx.Paths.GatewayHome))
	}

	if pathErr != nil {
		return f.internalErrorWithMsg(types.TeFileNotFound,
			"failed to get the file's final destination path",
			"temp file rename failed",
			pathErr)
	}

	if f.TransCtx.Transfer.LocalPath == dest {
		return nil
	}

	if err := createDir(dest); err != nil {
		return f.internalErrorWithMsg(types.TeFinalization,
			"failed to create destination directory",
			"temp file rename failed",
			err)
	}

	if err := fs.MoveFile(f.TransCtx.Transfer.LocalPath, dest); err != nil {
		return f.internalErrorWithMsg(types.TeFinalization,
			"Failed to move temp file",
			"temp file rename failed",
			err)
	}

	f.TransCtx.Transfer.LocalPath = dest

	if dbErr := f.UpdateTrans(); dbErr != nil {
		return dbErr
	}

	if f.Trace.OnMove != nil {
		if err := f.Trace.OnMove(); err != nil {
			return f.internalErrorWithMsg(types.TeInternal, "file move trace error",
				"temp file rename failed", err)
		}
	}

	return nil
}

func (f *FileStream) stop() {
	if fErr := f.file.Close(); fErr != nil {
		f.Logger.Warningf("Failed to close file: %v", fErr)
	}
}
