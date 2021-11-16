// Package internal regroups all the types and interfaces used in transfer pipelines.
package internal

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func leaf(s string) utils.Leaf     { return utils.Leaf(s) }
func branch(s string) utils.Branch { return utils.Branch(s) }

// GetFile opens/creates (depending on the transfer's direction) the file pointed
// by the transfer's local path and returns it as a *os.File.
func GetFile(logger *log.Logger, rule *model.Rule, trans *model.Transfer) (*os.File, *types.TransferError) {
	if rule.IsSend {
		file, err := os.OpenFile(trans.LocalPath, os.O_RDONLY, 0o600)
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

		return nil, err
	}

	file, err := os.OpenFile(trans.LocalPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		logger.Errorf("Failed to create destination file (%s): %s", trans.LocalPath, err)

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
func CreateDir(file string) *types.TransferError {
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return FileErrToTransferErr(err)
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

	switch {
	// Partner client <- GW server
	case transCtx.Transfer.IsServer && transCtx.Rule.IsSend:
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			leaf(transCtx.Rule.LocalDir), leaf(transCtx.LocalAgent.SendDir),
			branch(transCtx.LocalAgent.RootDir), leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	// Partner client -> GW server
	case transCtx.Transfer.IsServer && !transCtx.Rule.IsSend:
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			leaf(transCtx.Rule.TmpLocalRcvDir), leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.TmpReceiveDir), leaf(transCtx.LocalAgent.ReceiveDir),
			branch(transCtx.LocalAgent.RootDir), leaf(transCtx.Paths.DefaultTmpDir),
			leaf(transCtx.Paths.DefaultInDir), branch(transCtx.Paths.GatewayHome))
	// GW client -> Partner server
	case !transCtx.Transfer.IsServer && transCtx.Rule.IsSend:
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			leaf(transCtx.Rule.LocalDir), leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	// GW client <- Partner server
	case !transCtx.Transfer.IsServer && !transCtx.Rule.IsSend:
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			leaf(transCtx.Rule.TmpLocalRcvDir), leaf(transCtx.Rule.LocalDir),
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
