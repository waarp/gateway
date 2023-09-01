package tasks

import (
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func testError(err error) *types.TransferError {
	var te *types.TransferError
	if errors.As(err, &te) {
		return te
	}

	return types.NewTransferError(types.TeInternal, "test error: %v", err)
}
