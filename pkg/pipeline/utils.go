package pipeline

import (
	"os"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// fileErrToTransferErr takes an error returned by a file operation function
// (like os.Open or os.Create) and returns the corresponding types.TransferError.
func fileErrToTransferErr(err error) *types.TransferError {
	if os.IsNotExist(err) {
		return types.NewTransferError(types.TeFileNotFound, "file not found")
	}

	if os.IsPermission(err) {
		return types.NewTransferError(types.TeForbidden, "file operation not allowed")
	}

	return types.NewTransferError(types.TeUnknown, "file operation failed")
}
