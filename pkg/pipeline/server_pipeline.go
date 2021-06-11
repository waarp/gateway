package pipeline

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

// ServerPipeline is a struct regrouping a Pipeline with various server handlers
// for sending interruption signals to the transfer partner when needed.
type ServerPipeline struct {
	*Pipeline
	handlers Server
}

// GetOldServerTransfer searches the database for a transfer with the given
// remoteID and made with the given account. If the transfer cannot be found,
// it returns nil.
func GetOldServerTransfer(db *database.DB, logger *log.Logger, remoteID string,
	acc *model.LocalAccount) (*model.Transfer, *types.TransferError) {

	var trans model.Transfer
	if err := db.Get(&trans, "is_server=? AND remote_transfer_id=? AND account_id=?",
		true, remoteID, acc.ID).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil
		}
		logger.Errorf("Failed to retrieve old server transfer: %s", err)
		return nil, errDatabase
	}
	return &trans, nil
}

// NewServerTransfer inserts the given server transfer in the database.
func NewServerTransfer(db *database.DB, logger *log.Logger, trans *model.Transfer) *types.TransferError {
	if err := db.Insert(trans).Run(); err != nil {
		logger.Errorf("Failed to insert new server transfer: %s", err)
		return errDatabase
	}

	return nil
}

// NewServerPipeline returns a new ServerPipeline
func NewServerPipeline(db *database.DB, trans *model.Transfer, handlers Server,
) (*ServerPipeline, *types.TransferError) {
	logger := log.NewLogger(fmt.Sprintf("Pipeline %d", trans.ID))

	info, err := model.GetTransferInfo(db, logger, trans)
	if err != nil {
		return nil, err
	}

	pipeline, err := newPipeline(db, logger, info)
	if err != nil {
		return nil, err
	}

	s := &ServerPipeline{
		Pipeline: pipeline,
		handlers: handlers,
	}
	return s, nil
}

// Pause stops the server pipeline and pauses the transfer.
func (s *ServerPipeline) Pause() {
	if pa, ok := s.handlers.(PauseHandler); ok {
		_ = pa.Pause()
	} else {
		s.handlers.SendError(types.NewTransferError(types.TeStopped,
			"transfer paused by user"))
	}
	s.Pipeline.Pause()
}

// Interrupt stops the server pipeline and interrupts the transfer.
func (s *ServerPipeline) Interrupt() {
	s.handlers.SendError(types.NewTransferError(types.TeShuttingDown,
		"transfer interrupted by service shutdown"))
	s.Pipeline.interrupt()
}

// Cancel stops the server pipeline and cancels the transfer.
func (s *ServerPipeline) Cancel() {
	if ca, ok := s.handlers.(CancelHandler); ok {
		_ = ca.Cancel()
	} else {
		s.handlers.SendError(types.NewTransferError(types.TeCanceled,
			"transfer cancelled by user"))
	}
	s.Pipeline.Cancel()
}
