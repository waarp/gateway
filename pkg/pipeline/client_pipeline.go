package pipeline

import (
	"errors"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// NewClientPipeline initializes and returns a new ClientPipeline for the given
// transfer.
func NewClientPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) (*Pipeline, error) {
	pip, pipErr := newPipeline(db, logger, transCtx)
	if pipErr != nil {
		logger.Error("Failed to initialize the client transfer pipeline: %v", pipErr)

		tErr := types.NewTransferError(types.TeInternal, pipErr.Error())
		errors.As(pipErr, &tErr)

		transCtx.Transfer.Status = types.StatusError
		transCtx.Transfer.Error = *tErr

		if dbErr := db.Update(transCtx.Transfer).Run(); dbErr != nil {
			logger.Error("Failed to update the transfer error: %s", dbErr)
		}

		return nil, pipErr
	}

	if dbErr := pip.UpdateTrans(); dbErr != nil {
		logger.Error("Failed to update the transfer details: %s", dbErr)

		return nil, ErrDatabase
	}

	return pip, nil
}
