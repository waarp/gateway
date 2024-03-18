package pipeline

import (
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// fileErrToTransferErr takes an error returned by a file operation function
// (like os.Open or os.Create) and returns the corresponding types.TransferError.
func fileErrToTransferErr(err error) *types.TransferError {
	if errors.Is(err, fs.ErrNotExist) {
		return types.NewTransferError(types.TeFileNotFound, "file not found")
	}

	if errors.Is(err, fs.ErrPermission) {
		return types.NewTransferError(types.TeForbidden, "file operation not allowed")
	}

	return types.NewTransferError(types.TeUnknown, "file operation failed")
}
