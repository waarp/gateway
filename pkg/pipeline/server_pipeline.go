package pipeline

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

// GetServerTransfer searches the database for an interrupted transfer with the
// given remoteID and made with the given account. If such a transfer is found,
// the request is considered a retry, and the old entry is thus returned.
//
// If the transfer cannot be found, a new one is created from the information
// given, and then returned.
func GetServerTransfer(db *database.DB, logger *log.Logger, trans *model.Transfer,
) (*model.Transfer, *types.TransferError) {

	err := db.Get(trans, "status<>? AND is_server=? AND remote_transfer_id=? AND account_id=?",
		types.StatusRunning, true, trans.RemoteTransferID, trans.AccountID).Run()
	if err == nil {
		return trans, nil
	}
	if !database.IsNotFound(err) {
		logger.Errorf("Failed to retrieve old server transfer: %s", err)
		return nil, errDatabase
	}

	if err := db.Insert(trans).Run(); err != nil {
		logger.Errorf("Failed to insert new server transfer: %s", err)
		return nil, errDatabase
	}
	return trans, nil
}

// NewServerPipeline initialises and returns a new pipeline suitable for a
// server transfer.
func NewServerPipeline(db *database.DB, trans *model.Transfer,
) (*Pipeline, *types.TransferError) {
	logger := log.NewLogger(fmt.Sprintf("Pipeline %d", trans.ID))
	transCtx, err := model.GetTransferInfo(db, logger, trans)
	if err != nil {
		return nil, err
	}
	return newPipeline(db, logger, transCtx)
}
