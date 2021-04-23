package pipeline

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

type ServerPipeline struct {
	pip      *Pipeline
	handlers Server
}

func NewServerTransfer(db *database.DB, logger *log.Logger, trans *model.Transfer) error {
	if err := db.Insert(trans).Run(); err != nil {
		logger.Errorf("Failed to insert new transfer: %s", err)
		return errDatabase
	}

	return nil
}

func NewServerPipeline(db *database.DB, trans *model.Transfer, handlers Server) (*Pipeline, error) {

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
		pip:      pipeline,
		handlers: handlers,
	}
	RunningTransfers.Store(trans.ID, s)
	return pipeline, nil
}

func (s *ServerPipeline) Pause() {
	if pa, ok := s.handlers.(PauseHandler); ok {
		_ = pa.Pause()
	} else {
		s.handlers.SendError(types.NewTransferError(types.TeStopped,
			"transfer paused by user"))
	}
	s.pip.Pause()
}

func (s *ServerPipeline) Interrupt() {
	s.handlers.SendError(types.NewTransferError(types.TeShuttingDown,
		"transfer interrupted by service shutdown"))
	s.pip.interrupt()
}

func (s *ServerPipeline) Cancel() {
	if ca, ok := s.handlers.(CancelHandler); ok {
		_ = ca.Cancel()
	} else {
		s.handlers.SendError(types.NewTransferError(types.TeCanceled,
			"transfer cancelled by user"))
	}
	s.pip.Cancel()
}
