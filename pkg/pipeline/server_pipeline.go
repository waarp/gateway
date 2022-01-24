package pipeline

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// GetServerTransfer searches the database for an interrupted transfer with the
// given remoteID and made with the given account. If such a transfer is found,
// the request is considered a retry, and the old entry is thus returned.
//
// If the transfer cannot be found, a new one is created from the information
// given, and then returned.
func GetServerTransfer(db *database.DB, logger *log.Logger, trans *model.Transfer,
) (*model.Transfer, *types.TransferError) {
	if trans.RemoteTransferID != "" {
		err := db.Get(trans, "is_server=? AND remote_transfer_id=? AND account_id=?",
			true, trans.RemoteTransferID, trans.AccountID).Run()
		if err == nil {
			if trans.Status == types.StatusRunning {
				return nil, types.NewTransferError(types.TeForbidden, "cannot "+
					"resume a currently running transfer")
			}

			return trans, nil
		}

		if !database.IsNotFound(err) {
			logger.Errorf("Failed to retrieve old server transfer: %s", err)

			return nil, errDatabase
		}
	}

	if err := db.Insert(trans).Run(); err != nil {
		logger.Errorf("Failed to insert new server transfer: %s", err)

		return nil, errDatabase
	}

	return trans, nil
}

// NewServerPipeline initializes and returns a new pipeline suitable for a
// server transfer.
func NewServerPipeline(db *database.DB, trans *model.Transfer,
) (*Pipeline, *types.TransferError) {
	logger := log.NewLogger(fmt.Sprintf("Pipeline %d (server)", trans.ID))

	transCtx, err := model.GetTransferContext(db, logger, trans)
	if err != nil {
		return nil, err
	}

	return newPipeline(db, logger, transCtx)
}
