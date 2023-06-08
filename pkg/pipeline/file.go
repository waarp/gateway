package pipeline

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func leaf(s string) utils.Leaf     { return utils.Leaf(s) }
func branch(s string) utils.Branch { return utils.Branch(s) }

// getFilesize returns the size of the given file. If the file does not exist or
// cannot be accessed, it returns the UnknownSize value (-1).
func getFilesize(file string) int64 {
	if info, err := os.Stat(file); err != nil {
		return model.UnknownSize
	} else {
		return info.Size()
	}
}

// GetFile opens/creates (depending on the transfer's direction) the file pointed
// by the transfer's local path and returns it as a *os.File.
func (f *fileStream) getFile() (*os.File, *types.TransferError) {
	trans := f.TransCtx.Transfer

	if f.TransCtx.Rule.IsSend {
		file, err := os.OpenFile(trans.LocalPath, os.O_RDONLY, 0o600)
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

				return nil, types.NewTransferError(types.TeForbidden, err.Error())
			}
		}

		return file, nil
	}

	if err := createDir(trans.LocalPath); err != nil {
		f.Logger.Error("Failed to create temp directory: %s", err)

		return nil, err
	}

	file, err := os.OpenFile(trans.LocalPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		f.Logger.Error("Failed to create destination file (%s): %s", trans.LocalPath, err)

		return nil, fileErrToTransferErr(err)
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
func createDir(file string) *types.TransferError {
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0o700); err != nil {
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
func (p *Pipeline) setFilePaths() {
	srcFilename := p.TransCtx.Transfer.SrcFilename
	destFilename := p.TransCtx.Transfer.DestFilename

	if destFilename == "" {
		destFilename = p.TransCtx.Transfer.SrcFilename
	}

	p.setCustomFilePaths(srcFilename, destFilename)
}

func (p *Pipeline) setCustomFilePaths(srcFilename, destFilename string) {
	if !p.TransCtx.Transfer.IsServer() && p.TransCtx.Transfer.RemotePath == "" {
		if p.TransCtx.Rule.IsSend {
			p.TransCtx.Transfer.RemotePath = path.Join(p.TransCtx.Rule.RemoteDir, destFilename)
		} else {
			p.TransCtx.Transfer.RemotePath = path.Join(p.TransCtx.Rule.RemoteDir, srcFilename)
		}
	}

	if p.TransCtx.Transfer.LocalPath == "" {
		p.TransCtx.Transfer.LocalPath = makeLocalPath(p.TransCtx, srcFilename, destFilename)
	}
}

func makeLocalPath(transCtx *model.TransferContext, srcFilename,
	destFilename string,
) string {
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
func (p *Pipeline) checkFileExist() *types.TransferError {
	trans := p.TransCtx.Transfer

	info, err := os.Stat(trans.LocalPath)
	if err != nil {
		if os.IsNotExist(err) {
			p.Logger.Error("Failed to open transfer file %s: file does not exist", trans.LocalPath)

			return types.NewTransferError(types.TeFileNotFound, "file does not exist")
		}

		if os.IsPermission(err) {
			p.Logger.Error("Failed to open transfer file %s: permission denied", trans.LocalPath)

			return types.NewTransferError(types.TeForbidden, "permission to open file denied")
		}

		p.Logger.Error("Failed to open transfer file %s: %s", trans.LocalPath, err)

		return types.NewTransferError(types.TeUnknown, fmt.Sprintf("unknown file error: %s", err))
	}

	trans.Filesize = info.Size()

	return nil
}
