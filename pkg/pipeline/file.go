package pipeline

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func leaf(s string) utils.Leaf     { return utils.Leaf(s) }
func branch(s string) utils.Branch { return utils.Branch(s) }

// getFile opens/creates (depending on the transfer's direction) the file pointed
// by the transfer's local path and returns it as a *os.File.
func (f *fileStream) getFile() (*os.File, *types.TransferError) {
	trans := f.TransCtx.Transfer

	if f.TransCtx.Rule.IsSend {
		file, err := os.OpenFile(trans.LocalPath, os.O_RDONLY, 0o600)
		if err != nil {
			f.Logger.Errorf("Failed to open source file: %s", err)

			return nil, fileErrToTransferErr(err)
		}

		if trans.Progress != 0 {
			if _, err := file.Seek(int64(trans.Progress), io.SeekStart); err != nil {
				f.Logger.Errorf("Failed to seek inside file: %s", err)

				return nil, types.NewTransferError(types.TeForbidden, err.Error())
			}
		}

		return file, nil
	}

	if err := createDir(trans.LocalPath); err != nil {
		f.Logger.Errorf("Failed to create temp directory: %s", err)

		return nil, err
	}

	file, err := os.OpenFile(trans.LocalPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		f.Logger.Errorf("Failed to create destination file (%s): %s", trans.LocalPath, err)

		return nil, fileErrToTransferErr(err)
	}

	if trans.Progress != 0 {
		if _, err := file.Seek(int64(trans.Progress), io.SeekStart); err != nil {
			f.Logger.Errorf("Failed to seek inside file: %s", err)

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
// transfer's context. For the local path, the building process is as follow:
//
//   GatewayHome
//       ├─────────────────────────────────────────────────────┐                       ↑
//   Server root*                                     Default in/out/tmp dir     Lower priority
//       ├───────────────────────────┐
//       |                 Server in/out/tmp dir*                                Higher priority
//   Rule local path                                                                   ↓
//
//  *only applicable in server transfers
//
// For remote paths, only the rule's remote dir is added (if defined) before the
// file name.
func (p *Pipeline) makeFilePaths() {
	p.TransCtx.Transfer.RemotePath = path.Join("/", p.TransCtx.Rule.RemoteDir,
		p.TransCtx.Transfer.RemotePath)

	switch {
	// Partner client <- GW server
	case p.TransCtx.Transfer.IsServer && p.TransCtx.Rule.IsSend:
		p.TransCtx.Transfer.LocalPath = utils.GetPath(p.TransCtx.Transfer.LocalPath,
			leaf(p.TransCtx.Rule.LocalDir), leaf(p.TransCtx.LocalAgent.SendDir),
			branch(p.TransCtx.LocalAgent.RootDir), leaf(p.TransCtx.Paths.DefaultOutDir),
			branch(p.TransCtx.Paths.GatewayHome))
	// Partner client -> GW server
	case p.TransCtx.Transfer.IsServer && !p.TransCtx.Rule.IsSend:
		p.TransCtx.Transfer.LocalPath = utils.GetPath(p.TransCtx.Transfer.LocalPath,
			leaf(p.TransCtx.Rule.TmpLocalRcvDir), leaf(p.TransCtx.Rule.LocalDir),
			leaf(p.TransCtx.LocalAgent.TmpReceiveDir), leaf(p.TransCtx.LocalAgent.ReceiveDir),
			branch(p.TransCtx.LocalAgent.RootDir), leaf(p.TransCtx.Paths.DefaultTmpDir),
			leaf(p.TransCtx.Paths.DefaultInDir), branch(p.TransCtx.Paths.GatewayHome))
	// GW client -> Partner server
	case !p.TransCtx.Transfer.IsServer && p.TransCtx.Rule.IsSend:
		p.TransCtx.Transfer.LocalPath = utils.GetPath(p.TransCtx.Transfer.LocalPath,
			leaf(p.TransCtx.Rule.LocalDir), leaf(p.TransCtx.Paths.DefaultOutDir),
			branch(p.TransCtx.Paths.GatewayHome))
	// GW client <- Partner server
	case !p.TransCtx.Transfer.IsServer && !p.TransCtx.Rule.IsSend:
		p.TransCtx.Transfer.LocalPath = utils.GetPath(p.TransCtx.Transfer.LocalPath,
			leaf(p.TransCtx.Rule.TmpLocalRcvDir), leaf(p.TransCtx.Rule.LocalDir),
			leaf(p.TransCtx.Paths.DefaultTmpDir), leaf(p.TransCtx.Paths.DefaultOutDir),
			branch(p.TransCtx.Paths.GatewayHome))
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
			p.Logger.Errorf("Failed to open transfer file %s: file does not exist", trans.LocalPath)

			return types.NewTransferError(types.TeFileNotFound, "file does not exist")
		}

		if os.IsPermission(err) {
			p.Logger.Errorf("Failed to open transfer file %s: permission denied", trans.LocalPath)

			return types.NewTransferError(types.TeForbidden, "permission to open file denied")
		}

		p.Logger.Errorf("Failed to open transfer file %s: %s", trans.LocalPath, err)

		return types.NewTransferError(types.TeUnknown, fmt.Sprintf("unknown file error: %s", err))
	}

	trans.Filesize = info.Size()

	return nil
}
