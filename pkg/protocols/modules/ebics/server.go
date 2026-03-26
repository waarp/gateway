package ebics

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type Server struct {
	db     *database.DB
	logger *log.Logger
	server *model.LocalAgent
	config *serverConfig
	state  utils.State
}

func NewServer(db *database.DB, dbServer *model.LocalAgent) *Server {
	return &Server{db: db, server: dbServer}
}

func (s *Server) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.server.Name)
	cfg := defaultServerConfig()
	if err := utils.JSONConvert(s.server.ProtoConfig, cfg); err != nil {
		err = wrapConfigError(err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	if err := cfg.ValidServer(); err != nil {
		err = wrapConfigError(err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.config = cfg
	err := fmt.Errorf("%w: server bootstrap is not implemented yet", ErrNotImplemented)
	s.state.Set(utils.StateError, err.Error())
	s.logger.Warning(err.Error())

	return err
}

func (s *Server) Stop(context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *Server) State() (utils.StateCode, string) {
	return s.state.Get()
}
