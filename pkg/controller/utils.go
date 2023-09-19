package controller

import (
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func asTransferError(defaultCode types.TransferErrorCode, err error) *types.TransferError {
	tErr := types.NewTransferError(defaultCode, err.Error())
	errors.As(err, &tErr)

	return tErr
}
