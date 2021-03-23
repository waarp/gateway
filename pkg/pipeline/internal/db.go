package internal

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

func UpdateError(db *database.DB, logger *log.Logger, trans *model.Transfer,
	code types.TransferErrorCode, details string) {
	trans.Error = types.NewTransferError(code, details)
	if dbErr := db.Update(trans).Cols("error_code", "error_details").
		Run(); dbErr != nil {
		logger.Errorf("Failed to update transfer error: %s", dbErr)
	}
}
