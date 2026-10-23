package pipeline

import (
	"errors"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrNonLocalTmpFile = errors.New("temp received files must be local")

// getFilesize returns the size of the given file. If the file does not exist or
// cannot be accessed, it returns the UnknownSize value (-1).
func getFilesize(file string) int64 {
	info, err := fs.Stat(file)
	if err != nil {
		return model.UnknownSize
	}

	return info.Size()
}

// GetFile opens/creates (depending on the transfer's direction) the file pointed
// by the transfer's local path and returns it as a fs.File.
func (f *FileStream) getFile() (fs.File, *Error) {
	trans := f.TransCtx.Transfer

	if f.TransCtx.Rule.IsSend {
		file, opErr := fs.Open(trans.LocalPath)
		if opErr != nil {
			f.Logger.Errorf("Failed to open source file: %v", opErr)

			return nil, FileErrToTransferErr(opErr)
		}

		stat, statErr := file.Stat()
		if statErr != nil {
			f.Logger.Errorf("Failed to retrieve the file's info: %v", statErr)

			return nil, FileErrToTransferErr(statErr)
		}

		trans.Filesize = stat.Size()

		if trans.Progress != 0 {
			if _, err := file.Seek(trans.Progress, io.SeekStart); err != nil {
				f.Logger.Errorf("Failed to seek inside file: %v", err)

				return nil, NewErrorWith(err, types.TeForbidden, "failed to seek inside file")
			}
		}

		return file, nil
	}

	if err := createDir(trans.LocalPath); err != nil {
		f.Logger.Errorf("Failed to create temp directory: %v", err)

		return nil, err
	}

	filePerms := fs.FileMode(f.DB.Config.Paths.FilePerms)

	file, fsErr := fs.OpenFile(trans.LocalPath, fs.FlagReadWrite|fs.FlagCreate, filePerms)
	if fsErr != nil {
		f.Logger.Errorf("Failed to create destination file %q: %v", trans.LocalPath, fsErr)

		return nil, FileErrToTransferErr(fsErr)
	}

	if trans.Progress != 0 {
		if _, err := file.Seek(trans.Progress, io.SeekStart); err != nil {
			f.Logger.Errorf("Failed to seek inside file: %v", err)

			return nil, FileErrToTransferErr(err)
		}
	}

	return file, nil
}

// createDir takes a file path and creates all the file's parent directories if
// they don't exist.
func createDir(file string) *Error {
	if err := fs.MkdirAll(fs.Dir(file)); err != nil {
		return FileErrToTransferErr(err)
	}

	return nil
}

// setFilePaths builds the transfer's local & remote paths according to the
// transfer's context. For the local path, the building process is as follows:
//
//	 GatewayHome                                                                      ↑
//	     ├─────────────────────────────────────────────────────┐                 Less priority
//	 Server root*                                     Default in/out/tmp dir
//	     ├───────────────────────────┐                                           More priority
//	 Rule local path       Server in/out/tmp dir*                                     ↓
//
//	*only applicable in server transfers
//
// For remote paths, only the rule's remote dir is added (if defined) before the
// file name.
func (p *Pipeline) setFilePaths() error {
	srcFilename, destFilename := fileNames(p.TransCtx.Transfer)

	return p.setCustomFilePaths(srcFilename, destFilename)
}

// fileNames are the names the paths of a transfer are built from: the name
// at the destination is the one at the source when it is empty.
func fileNames(trans *model.Transfer) (srcFilename, destFilename string) {
	srcFilename, destFilename = trans.SrcFilename, trans.DestFilename
	if destFilename == "" {
		destFilename = srcFilename
	}

	return srcFilename, destFilename
}

// remotePath is the path of the file of a client transfer at the partner:
// the remote dir of the rule, then the name of the file there.
func remotePath(transCtx *model.TransferContext, srcFilename, destFilename string) string {
	if transCtx.Rule.IsSend {
		return path.Join(transCtx.Rule.RemoteDir, destFilename)
	}

	return path.Join(transCtx.Rule.RemoteDir, srcFilename)
}

// ExpectedPaths gives the local path of a transfer and, for a client
// transfer, its remote path, as the pipeline sets them when the transfer
// starts (setFilePaths); a path already set is kept. The transfer is not
// changed: the web interface shows where the file of a transfer not started
// yet will be read or written.
func ExpectedPaths(transCtx *model.TransferContext) (localPath, remote string, err error) {
	trans := transCtx.Transfer
	srcFilename, destFilename := fileNames(trans)
	localPath, remote = trans.LocalPath, trans.RemotePath

	if !trans.IsServer() && remote == "" {
		remote = remotePath(transCtx, srcFilename, destFilename)
	}

	if localPath == "" {
		if localPath, err = makeLocalPath(transCtx, srcFilename, destFilename); err != nil {
			return "", "", fmt.Errorf("failed to build local path: %w", err)
		}
	}

	return localPath, remote, nil
}

func (p *Pipeline) setCustomFilePaths(srcFilename, destFilename string) error {
	if !p.TransCtx.Transfer.IsServer() && p.TransCtx.Transfer.RemotePath == "" {
		p.TransCtx.Transfer.RemotePath = remotePath(p.TransCtx, srcFilename, destFilename)
	}

	if p.TransCtx.Transfer.LocalPath == "" {
		fPath, err := makeLocalPath(p.TransCtx, srcFilename, destFilename)
		if err != nil {
			return fmt.Errorf("failed to build local path: %w", err)
		}

		if !p.TransCtx.Rule.IsSend && !fs.IsLocalPath(fPath) {
			return fmt.Errorf("%q: %w", fPath, ErrNonLocalTmpFile)
		}

		p.TransCtx.Transfer.LocalPath = fPath
	}

	return nil
}

//nolint:wrapcheck //wrapping is done by the caller function (just above)
func makeLocalPath(transCtx *model.TransferContext, srcFilename,
	destFilename string,
) (string, error) {
	var (
		leaf   = utils.Leaf
		branch = utils.Branch
	)

	switch {
	// Partner client <- GW server
	case transCtx.Transfer.IsServer() && transCtx.Rule.IsSend:
		return utils.GetPath(srcFilename, leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.SendDir), branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultOutDir), branch(transCtx.Paths.GatewayHome))
	// Partner client -> GW server
	case transCtx.Transfer.IsServer() && !transCtx.Rule.IsSend:
		return utils.GetPath(destFilename, leaf(transCtx.Rule.TmpLocalRcvDir),
			leaf(transCtx.Rule.LocalDir), leaf(transCtx.LocalAgent.TmpReceiveDir),
			leaf(transCtx.LocalAgent.ReceiveDir), branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultTmpDir), leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	// GW client -> Partner server
	case !transCtx.Transfer.IsServer() && transCtx.Rule.IsSend:
		return utils.GetPath(srcFilename, leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.Paths.DefaultOutDir), branch(transCtx.Paths.GatewayHome))
	// GW client <- Partner server
	default:
		return utils.GetPath(destFilename, leaf(transCtx.Rule.TmpLocalRcvDir),
			leaf(transCtx.Rule.LocalDir), leaf(transCtx.Paths.DefaultTmpDir),
			leaf(transCtx.Paths.DefaultInDir), branch(transCtx.Paths.GatewayHome))
	}
}

// checkFileExist checks if the transfer's local path does point to a file. If
// the file does exist, it also updates the transfer's filesize field with the
// file's size. If the file does not exist, an error is returned.
func (f *FileStream) checkFileExist() *Error {
	trans := f.TransCtx.Transfer

	info, err := fs.Stat(trans.LocalPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			f.Logger.Errorf("Failed to open transfer file %q: file does not exist", trans.LocalPath)

			return f.internalError(types.TeFileNotFound, "file does not exist", err)
		}

		if errors.Is(err, fs.ErrPermission) {
			f.Logger.Errorf("Failed to open transfer file %q: permission denied", trans.LocalPath)

			return f.internalError(types.TeForbidden, "permission to open file denied", err)
		}

		f.Logger.Errorf("Failed to open transfer file %q: %v", trans.LocalPath, err)

		return f.internalErrorWithMsg(types.TeUnknown, "unknown file error",
			"failed to open file", err)
	}

	trans.Filesize = info.Size()

	return nil
}
