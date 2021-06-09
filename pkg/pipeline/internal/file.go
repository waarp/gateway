// Package internal regroups all the types and interfaces used in transfer pipelines.
package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func Leaf(s string) utils.Leaf     { return utils.Leaf(s) }
func Branch(s string) utils.Branch { return utils.Branch(s) }

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

func FileErrToTransferErr(err error) *types.TransferError {
	if os.IsNotExist(err) {
		return types.NewTransferError(types.TeFileNotFound, "file not found")
	}
	if os.IsPermission(err) {
		return types.NewTransferError(types.TeForbidden, "file operation not allowed")
	}

	return types.NewTransferError(types.TeUnknown, "file operation failed")
}

func MakeFilepaths(transCtx *model.TransferContext) {
	if transCtx.Transfer.Step != types.StepNone {
		return
	}

	transCtx.Transfer.RemotePath = utils.GetPath(transCtx.Transfer.RemotePath,
		Leaf(transCtx.Rule.RemoteDir))

	if transCtx.Rule.IsSend && transCtx.Transfer.IsServer {
		// Partner client <- GW server
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			Leaf(transCtx.Rule.LocalDir), Leaf(transCtx.LocalAgent.LocalOutDir),
			Branch(transCtx.LocalAgent.Root), Leaf(transCtx.Paths.DefaultOutDir),
			Branch(transCtx.Paths.GatewayHome))
	} else if transCtx.Transfer.IsServer {
		// Partner client -> GW server
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath+".part",
			Leaf(transCtx.Rule.LocalTmpDir), Leaf(transCtx.Rule.LocalDir),
			Leaf(transCtx.LocalAgent.LocalTmpDir), Leaf(transCtx.LocalAgent.LocalInDir),
			Branch(transCtx.LocalAgent.Root), Leaf(transCtx.Paths.DefaultTmpDir),
			Leaf(transCtx.Paths.DefaultInDir), Branch(transCtx.Paths.GatewayHome))
	} else if transCtx.Rule.IsSend {
		// GW client -> Partner server
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath,
			Leaf(transCtx.Rule.LocalDir), Leaf(transCtx.Paths.DefaultOutDir),
			Branch(transCtx.Paths.GatewayHome))
	} else {
		// GW client <- Partner server
		transCtx.Transfer.LocalPath = utils.GetPath(transCtx.Transfer.LocalPath+".part",
			Leaf(transCtx.Rule.LocalTmpDir), Leaf(transCtx.Rule.LocalDir),
			Leaf(transCtx.Paths.DefaultTmpDir), Leaf(transCtx.Paths.DefaultOutDir),
			Branch(transCtx.Paths.GatewayHome))
	}
}

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
