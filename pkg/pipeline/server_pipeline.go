package pipeline

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

// GetOldTransfer searches the database for an interrupted transfer with the
// given remoteID and made with the given account. If such a transfer is found,
// the request is considered a retry, and the old entry is thus returned.
//
// If no transfer can be found, the entry is returned as is.
func GetOldTransfer(db *database.DB, logger *log.Logger, trans *model.Transfer,
) (*model.Transfer, *Error) {
	if trans.RemoteTransferID == "" {
		return trans, nil
	}

	var oldTrans model.Transfer

	err := db.Get(&oldTrans, "remote_transfer_id=? AND local_account_id=?",
		trans.RemoteTransferID, trans.LocalAccountID.Int64).OrderBy("start", false).Run()
	if err == nil {
		if oldTrans.Status == types.StatusRunning {
			return nil, NewError(types.TeForbidden,
				"cannot resume a currently running transfer")
		}

		return &oldTrans, nil
	}

	if !database.IsNotFound(err) {
		logger.Error("Failed to retrieve old server transfer: %s", err)

		return nil, NewErrorWith(types.TeInternal, "failed to retrieve old server transfer", err)
	}

	return trans, nil
}

// NewServerPipeline initializes and returns a new pipeline suitable for a
// server transfer.
func NewServerPipeline(db *database.DB, logger *log.Logger, trans *model.Transfer,
	snmpService *snmp.Service,
) (*Pipeline, *Error) {
	transCtx, ctxErr := model.GetTransferContext(db, logger, trans)
	if ctxErr != nil {
		return nil, NewError(types.TeInternal, "database error")
	}

	pipeline, pipErr := newPipeline(db, logger, transCtx, snmpService)
	if pipErr != nil {
		logger.Error("Failed to initialize the server transfer pipeline %d: %v",
			trans.ID, pipErr)

		return nil, pipErr
	}

	if transCtx.Rule.IsSend {
		pipeline.Logger.Info(
			"Starting download of file %q requested by %q on the server %q using rule %q",
			transCtx.Transfer.LocalPath, transCtx.LocalAccount.Login,
			transCtx.LocalAgent.Name, transCtx.Rule.Name)
	} else {
		pipeline.Logger.Info(
			"Starting upload of file %q requested by %q on the server %q using rule %q",
			transCtx.Transfer.LocalPath, transCtx.LocalAccount.Login,
			transCtx.LocalAgent.Name, transCtx.Rule.Name)
	}

	return pipeline, nil
}
