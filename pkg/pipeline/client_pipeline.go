package pipeline

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

// NewClientPipeline initializes and returns a new ClientPipeline for the given
// transfer.
func NewClientPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
	snmpService *snmp.Service,
) (*Pipeline, *Error) {
	pip, pipErr := newPipeline(db, logger, transCtx, snmpService)
	if pipErr != nil {
		logger.Error("Failed to initialize the client transfer pipeline: %v", pipErr)

		transCtx.Transfer.Status = types.StatusError
		transCtx.Transfer.ErrCode = pipErr.code
		transCtx.Transfer.ErrDetails = pipErr.Details()

		if dbErr := db.Update(transCtx.Transfer).Run(); dbErr != nil {
			logger.Error("Failed to update the transfer error: %s", dbErr)
		}

		return nil, pipErr
	}

	if dbErr := pip.UpdateTrans(); dbErr != nil {
		logger.Error("Failed to update the transfer details: %s", dbErr)

		return nil, NewErrorWith(types.TeInternal, "Failed to update the transfer details", dbErr)
	}

	return pip, nil
}
