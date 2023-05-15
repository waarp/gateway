package pipeline

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// GetOldTransfer searches the database for an interrupted transfer with the
// given remoteID and made with the given account. If such a transfer is found,
// the request is considered a retry, and the old entry is thus returned.
//
// If no transfer can be found, the entry is returned as is.
func GetOldTransfer(db *database.DB, logger *log.Logger, trans *model.Transfer,
) (*model.Transfer, *types.TransferError) {
	if trans.RemoteTransferID == "" {
		return trans, nil
	}

	var oldTrans model.Transfer

	err := db.Get(&oldTrans, "remote_transfer_id=? AND local_account_id=?",
		trans.RemoteTransferID, trans.LocalAccountID.Int64).Run()
	if err == nil {
		if oldTrans.Status == types.StatusRunning {
			return nil, types.NewTransferError(types.TeForbidden,
				"cannot resume a currently running transfer")
		}

		return &oldTrans, nil
	}

	if !database.IsNotFound(err) {
		logger.Error("Failed to retrieve old server transfer: %s", err)

		return nil, errDatabase
	}

	return trans, nil
}

// NewServerPipeline initializes and returns a new pipeline suitable for a
// server transfer.
func NewServerPipeline(db *database.DB, trans *model.Transfer,
) (*Pipeline, *types.TransferError) {
	return newServerPipeline(db, trans)
}

func newServerPipeline(db *database.DB, trans *model.Transfer,
) (*Pipeline, *types.TransferError) {
	logger := conf.GetLogger(fmt.Sprintf("Pipeline %d (server)", trans.ID))

	transCtx, err := model.GetTransferContext(db, logger, trans)
	if err != nil {
		return nil, err
	}

	pipeline := newPipeline(db, logger, transCtx)

	if trans.ID == 0 {
		if err := db.Insert(trans).Run(); err != nil {
			logger.Error("failed to insert the new transfer entry: %s", err)

			return nil, errDatabase
		}

		*logger = *conf.GetLogger(fmt.Sprintf("Pipeline %d (server)", trans.ID))
	} else if err := pipeline.UpdateTrans(); err != nil {
		logger.Error("Failed to update the transfer details: %s", err)

		return nil, errDatabase
	}

	return pipeline, nil
}
