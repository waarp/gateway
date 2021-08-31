// Package internal regroups all the types and interfaces used in transfer pipelines.
package internal

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func leaf(s string) utils.Leaf     { return utils.Leaf(s) }
func branch(s string) utils.Branch { return utils.Branch(s) }

// GetFile opens/creates (depending on the transfer's direction) the file pointed
// by the transfer's local path and returns it as a *os.File.
func GetFile(logger *log.Logger, rule *model.Rule, trans *model.Transfer) (*os.File, *types.TransferError) {

	path := trans.LocalPath
	if rule.IsSend {
		file, err := os.OpenFile(path, os.O_RDONLY, 0600)
		if err != nil {
			logger.Errorf("Failed to open source file: %s", err)
			return nil, FileErrToTransferErr(err)
		}
		if trans.Progress != 0 {
			if _, err := file.Seek(int64(trans.Progress), io.SeekStart); err != nil {
				logger.Errorf("Failed to seek inside file: %s", err)
				return nil, types.NewTransferError(types.TeForbidden, err.Error())
			}
		}
		return file, nil
	}

	if err := CreateDir(trans.LocalPath); err != nil {
		logger.Errorf("Failed to create temp directory: %s", err)
		return nil, types.NewTransferError(types.TeForbidden, err.Error())
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		logger.Errorf("Failed to create destination file (%s): %s", path, err)
		return nil, FileErrToTransferErr(err)
	}
	if trans.Progress != 0 {
		if _, err := file.Seek(int64(trans.Progress), io.SeekStart); err != nil {
			logger.Errorf("Failed to seek inside file: %s", err)
			return nil, FileErrToTransferErr(err)
		}
	}
	return file, nil
}

// CreateDir takes a file path and creates all the file's parent directories if
// they don't exist.
func CreateDir(path string) error {
	dir := filepath.Dir(path)
	if info, err := os.Lstat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return err
			}
		} else {
			return err
		}
	} else if !info.IsDir() {
		return fmt.Errorf("a file named '%s' already exist", filepath.Base(dir))
	}
	return nil
}

// FileErrToTransferErr takes an error returned by a file operation function
// (like os.Open or os.Create) and returns the corresponding types.TransferError.
func FileErrToTransferErr(err error) *types.TransferError {
	if os.IsNotExist(err) {
		return types.NewTransferError(types.TeFileNotFound, "file not found")
	}
	if os.IsPermission(err) {
		return types.NewTransferError(types.TeForbidden, "file operation not allowed")
	}

	return types.NewTransferError(types.TeUnknown, "file operation failed")
}

// MakeFilepaths builds the transfer's local & remote paths according to the
// transfer's context. For the local path, the building process is as follow:
//
//   GatewayHome                                                                      ↑
//       ├─────────────────────────────────────────────────────┐                 Less priority
//   Server root*                                     Default in/out/tmp dir
//       ├───────────────────────────┐                                           More priority
//   Rule local path       Server in/out/tmp dir*                                     ↓
//
//  *only applicable in server transfers
//
// For remote paths, only the rule's remote dir is added (if defined) before the
// file name.
func MakeFilepaths(transCtx *model.TransferContext) {
	transCtx.Transfer.RemotePath = path.Join("/", transCtx.Rule.RemoteDir,
		transCtx.Transfer.RemotePath)

	if transCtx.Rule.IsSend && transCtx.Transfer.IsServer {
		// Partner client <- GW server
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			leaf(transCtx.Rule.LocalDir), leaf(transCtx.LocalAgent.LocalOutDir),
			branch(transCtx.LocalAgent.Root), leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	} else if transCtx.Transfer.IsServer {
		// Partner client -> GW server
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			leaf(transCtx.Rule.LocalTmpDir), leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.LocalTmpDir), leaf(transCtx.LocalAgent.LocalInDir),
			branch(transCtx.LocalAgent.Root), leaf(transCtx.Paths.DefaultTmpDir),
			leaf(transCtx.Paths.DefaultInDir), branch(transCtx.Paths.GatewayHome))
	} else if transCtx.Rule.IsSend {
		// GW client -> Partner server
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			leaf(transCtx.Rule.LocalDir), leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	} else {
		// GW client <- Partner server
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			leaf(transCtx.Rule.LocalTmpDir), leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.Paths.DefaultTmpDir), leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	}
}

// CheckFileExist checks if the transfer's local path does point to a file. If
// the file does exist, it also updates the transfer's filesize field with the
// file's size. If the file does not exist, an error is returned.
func CheckFileExist(trans *model.Transfer, db *database.DB, logger *log.Logger) *types.TransferError {
	info, err := os.Stat(trans.LocalPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Errorf("Failed to open transfer file %s: file does not exist", trans.LocalPath)
			return types.NewTransferError(types.TeFileNotFound, "file does not exist")
		}
		if os.IsPermission(err) {
			logger.Errorf("Failed to open transfer file %s: permission denied", trans.LocalPath)
			return types.NewTransferError(types.TeForbidden, "permission to open file denied")
		}
		logger.Errorf("Failed to open transfer file %s: %s", trans.LocalPath, err)
		return types.NewTransferError(types.TeUnknown, fmt.Sprintf("unknown file error: %s", err))
	}
	trans.Filesize = info.Size()
	if err := db.Update(trans).Cols("filesize").Run(); err != nil {
		logger.Errorf("Failed to set file size: %s", err)
		return types.NewTransferError(types.TeInternal, "failed to set file size")
	}

	return nil
}
