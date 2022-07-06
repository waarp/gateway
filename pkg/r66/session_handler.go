package r66

import (
	"fmt"
	"path"
	"strings"
	"time"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

type sessionHandler struct {
	*authHandler

	account *model.LocalAccount
	conf    *r66.Authent
}

func (s *sessionHandler) ValidRequest(req *r66.Request) (r66.TransferHandler, error) {
	if err := s.checkRequest(req); err != nil {
		return nil, err
	}

	rule, err := s.getRule(req.Rule, req.IsRecv)
	if err != nil {
		return nil, err
	}

	if !rule.IsSend {
		s.logger.Infof("Upload of file %s was requested by %s, using rule %s",
			path.Base(req.Filepath), s.account.Login, req.Rule)
	} else {
		s.logger.Infof("Download of file %s was requested by %s, using rule %s",
			path.Base(req.Filepath), s.account.Login, req.Rule)
	}

	trans, err := s.getTransfer(req, rule)
	if err != nil {
		return nil, err
	}

	if progErr := s.setProgress(req, trans); progErr != nil {
		return nil, progErr
	}

	pip, pErr := pipeline.NewServerPipeline(s.db, trans)
	if pErr != nil {
		return nil, internal.ToR66Error(pErr)
	}

	if err := s.getSize(req, rule, trans); err != nil {
		pip.SetError(err)

		return nil, internal.ToR66Error(err)
	}

	if err := internal.UpdateTransferInfo(req.Infos, pip); err != nil {
		pip.SetError(err)

		return nil, internal.ToR66Error(err)
	}

	r66Pip := &serverTransfer{
		r66Conf: s.r66Conf,
		conf:    s.conf,
		pip:     pip,
		store:   utils.NewErrorStorage(),
	}

	s.runningTransfers.Add(trans.ID, r66Pip)

	handler := transferHandler{
		sessionHandler: s,
		trans:          r66Pip,
	}

	return &handler, nil
}

func (s *sessionHandler) checkRequest(req *r66.Request) *r66.Error {
	if req.Filepath == "" {
		return internal.NewR66Error(r66.IncorrectCommand, "missing filepath")
	}

	if req.Block == 0 {
		return internal.NewR66Error(r66.IncorrectCommand, "missing block size")
	}

	if req.Rule == "" {
		return internal.NewR66Error(r66.IncorrectCommand, "missing transfer rule")
	}

	/*
		if !req.IsRecv && s.conf.Filesize && req.FileSize < 0 {
			return internal.NewR66Error(r66.IncorrectCommand, "missing file size")
		}
	*/

	return nil
}

func (s *sessionHandler) getRule(ruleName string, isSend bool) (*model.Rule, *r66.Error) {
	var rule model.Rule
	if err := s.db.Get(&rule, "name=? AND send=?", ruleName, isSend).Run(); err != nil {
		if database.IsNotFound(err) {
			rule.IsSend = isSend
			s.logger.Warningf("Requested %s transfer rule '%s' does not exist",
				rule.Direction(), ruleName)

			return nil, internal.NewR66Error(r66.IncorrectCommand, "rule does not exist")
		}

		s.logger.Errorf("Failed to retrieve transfer rule: %s", err)

		return nil, internal.NewR66Error(r66.Internal, "database error")
	}

	ok, err := rule.IsAuthorized(s.db, s.account)
	if err != nil {
		s.logger.Errorf("Failed to check rule permissions: %s", err)

		return nil, internal.NewR66Error(r66.Internal, "database error")
	}

	if !ok {
		return nil, internal.NewR66Error(r66.FileNotAllowed, "you do not have the rights to use this transfer rule")
	}

	return &rule, nil
}

func (s *sessionHandler) getTransfer(req *r66.Request, rule *model.Rule) (*model.Transfer, *r66.Error) {
	trans := &model.Transfer{
		RemoteTransferID: fmt.Sprint(req.ID),
		RuleID:           rule.ID,
		IsServer:         true,
		AgentID:          s.agent.ID,
		AccountID:        s.account.ID,
		LocalPath:        strings.TrimPrefix(req.Filepath, "/"),
		RemotePath:       path.Base(req.Filepath),
		Start:            time.Now(),
		Status:           types.StatusPlanned,
	}

	trans, tErr := pipeline.GetOldTransfer(s.db, s.logger, trans)
	if tErr != nil {
		return nil, internal.ToR66Error(tErr)
	}

	return trans, nil
}

func (s *sessionHandler) getSize(req *r66.Request, rule *model.Rule, trans *model.Transfer) *types.TransferError {
	if rule.IsSend {
		req.FileSize = trans.Filesize

		return nil
	}

	if req.FileSize < 0 {
		return nil
	}

	trans.Filesize = req.FileSize
	if err := s.db.Update(trans).Cols("filesize").Run(); err != nil {
		s.logger.Errorf("Failed to set file size: %s", err)

		return types.NewTransferError(types.TeInternal, "database error")
	}

	return nil
}

func (s *sessionHandler) setProgress(req *r66.Request, trans *model.Transfer) *r66.Error {
	if trans.Step > types.StepData {
		return nil
	}

	prog := uint64(req.Rank) * uint64(req.Block)
	if trans.Progress <= prog {
		req.Rank = uint32(trans.Progress / uint64(req.Block))

		return nil
	}

	if prog == trans.Progress {
		return nil
	}

	trans.Progress = prog
	if err := s.db.Update(trans).Cols("progression").Run(); err != nil {
		s.logger.Errorf("Failed to update R66 transfer progress: %s", err)

		return internal.NewR66Error(r66.Internal, "database error")
	}

	return nil
}
