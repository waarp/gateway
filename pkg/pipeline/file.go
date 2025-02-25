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
	if info, err := fs.Stat(file); err != nil {
		return model.UnknownSize
	} else {
		return info.Size()
	}
}

// GetFile opens/creates (depending on the transfer's direction) the file pointed
// by the transfer's local path and returns it as a fs.File.
func (f *FileStream) getFile() (fs.File, *Error) {
	trans := f.TransCtx.Transfer

	if f.TransCtx.Rule.IsSend {
		file, err := fs.Open(trans.LocalPath)
		if err != nil {
			f.Logger.Error("Failed to open source file: %s", err)

			return nil, fileErrToTransferErr(err)
		}

		stat, err := file.Stat()
		if err != nil {
			f.Logger.Error("Failed to retrieve the file's info: %s", err)

			return nil, fileErrToTransferErr(err)
		}

		trans.Filesize = stat.Size()

		if trans.Progress != 0 {
			if _, err := file.Seek(trans.Progress, io.SeekStart); err != nil {
				f.Logger.Error("Failed to seek inside file: %s", err)

				return nil, NewErrorWith(types.TeForbidden, "failed to seek inside file", err)
			}
		}

		return file, nil
	}

	if err := createDir(trans.LocalPath); err != nil {
		f.Logger.Error("Failed to create temp directory: %s", err)

		return nil, err
	}

	file, fsErr := fs.OpenFile(trans.LocalPath, fs.FlagReadWrite|fs.FlagCreate, 0o600)
	if fsErr != nil {
		f.Logger.Error("Failed to create destination file %q: %s", trans.LocalPath, fsErr)

		return nil, fileErrToTransferErr(fsErr)
	}

	if trans.Progress != 0 {
		if _, err := file.Seek(trans.Progress, io.SeekStart); err != nil {
			f.Logger.Error("Failed to seek inside file: %s", err)

			return nil, fileErrToTransferErr(err)
		}
	}

	return file, nil
}

// createDir takes a file path and creates all the file's parent directories if
// they don't exist.
func createDir(file string) *Error {
	if err := fs.MkdirAll(path.Dir(file)); err != nil {
		return fileErrToTransferErr(err)
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
	srcFilename := p.TransCtx.Transfer.SrcFilename
	destFilename := p.TransCtx.Transfer.DestFilename

	if destFilename == "" {
		destFilename = p.TransCtx.Transfer.SrcFilename
	}

	return p.setCustomFilePaths(srcFilename, destFilename)
}

func (p *Pipeline) setCustomFilePaths(srcFilename, destFilename string) error {
	if !p.TransCtx.Transfer.IsServer() && p.TransCtx.Transfer.RemotePath == "" {
		if p.TransCtx.Rule.IsSend {
			p.TransCtx.Transfer.RemotePath = path.Join(p.TransCtx.Rule.RemoteDir, destFilename)
		} else {
			p.TransCtx.Transfer.RemotePath = path.Join(p.TransCtx.Rule.RemoteDir, srcFilename)
		}
	}

	if p.TransCtx.Transfer.LocalPath == "" {
		if fPath, err := makeLocalPath(p.TransCtx, srcFilename, destFilename); err != nil {
			return fmt.Errorf("failed to build local path: %w", err)
		} else {
			if !p.TransCtx.Rule.IsSend && !fs.IsLocalPath(fPath) {
				return ErrNonLocalTmpFile
			}

			p.TransCtx.Transfer.LocalPath = fPath
		}
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
			f.Logger.Error("Failed to open transfer file %q: file does not exist", trans.LocalPath)

			return f.internalError(types.TeFileNotFound, "file does not exist", err)
		}

		if errors.Is(err, fs.ErrPermission) {
			f.Logger.Error("Failed to open transfer file %q: permission denied", trans.LocalPath)

			return f.internalError(types.TeForbidden, "permission to open file denied", err)
		}

		f.Logger.Error("Failed to open transfer file %q: %s", trans.LocalPath, err)

		return f.internalErrorWithMsg(types.TeUnknown, "unknown file error",
			"failed to open file", err)
	}

	trans.Filesize = info.Size()

	return nil
}
