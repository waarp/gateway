package r66

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-r66/r66"
)

var (
	errDatabase    = types.NewTransferError(types.TeInternal, "database error")
	r66ErrDatabase = &r66.Error{Code: r66.Internal, Detail: "database error"}

	testEndSignal func(*pipeline.ServerPipeline)
)

func (s *sessionHandler) checkRequest(req *r66.Request) *r66.Error {
	if req.Filepath == "" {
		return &r66.Error{Code: r66.IncorrectCommand, Detail: "missing filepath"}
	}
	if req.Block == 0 {
		return &r66.Error{Code: r66.IncorrectCommand, Detail: "missing block size"}
	}
	if req.Rule == "" {
		return &r66.Error{Code: r66.IncorrectCommand, Detail: "missing transfer rule"}
	}
	if req.Mode == 0 {
		return &r66.Error{Code: r66.IncorrectCommand, Detail: "missing transfer mode"}
	}
	isSend := r66.IsRecv(req.Mode)
	if !isSend && s.conf.Filesize && req.FileSize < 0 {
		return &r66.Error{Code: r66.IncorrectCommand, Detail: "missing file size"}
	}

	return nil
}

func (s *sessionHandler) getRule(ruleName string, isSend bool) (*model.Rule, *r66.Error) {
	var rule model.Rule
	if err := s.db.Get(&rule, "name=? AND send=?", ruleName, isSend).Run(); err != nil {
		if database.IsNotFound(err) {
			s.logger.Warningf("Requested transfer rule '%s' does not exist", rule.Name)
			return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "rule does not exist"}
		}
		s.logger.Errorf("Failed to retrieve transfer rule: %s", err)
		return nil, r66ErrDatabase
	}
	return &rule, nil
}

func (s *sessionHandler) getTransfer(req *r66.Request, rule *model.Rule) (*model.Transfer, *r66.Error) {
	trans, err := pipeline.GetOldServerTransfer(s.db, s.logger, fmt.Sprint(req.ID), s.account)
	if err != nil {
		return nil, r66ErrDatabase
	}
	if trans == nil {
		trans = &model.Transfer{
			RemoteTransferID: fmt.Sprint(req.ID),
			RuleID:           rule.ID,
			IsServer:         true,
			AgentID:          s.agent.ID,
			AccountID:        s.account.ID,
			LocalPath:        path.Base(req.Filepath),
			RemotePath:       path.Base(req.Filepath),
			Start:            time.Now(),
			Status:           types.StatusPlanned,
		}
		if pipeline.NewServerTransfer(s.db, s.logger, trans) != nil {
			return nil, r66ErrDatabase
		}
	}
	return trans, nil
}

func (s *sessionHandler) getSize(req *r66.Request, rule *model.Rule, trans *model.Transfer) *types.TransferError {
	if !rule.IsSend {
		return nil
	}

	if s.conf.Filesize {
		stats, err := os.Stat(trans.LocalPath)
		if err != nil {
			s.logger.Errorf("Failed to retrieve file size: %s", err)
			return &types.TransferError{Code: types.TeInternal, Details: "failed to retrieve file size"}
		}
		req.FileSize = stats.Size()
	} else {
		req.FileSize = -1
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
		s.logger.Errorf("Failed to update transfer progress: %s", err)
		return r66ErrDatabase
	}
	return nil
}

func (t *transferHandler) checkSize() *r66.Error {
	if t.conf.Filesize {
		stat, err := os.Stat(t.pip.TransCtx.Transfer.LocalPath)
		if err != nil {
			t.pip.Logger.Errorf("Failed to retrieve file info: %s", err)
			return &r66.Error{
				Code:   r66.Internal,
				Detail: "failed to retrieve file info",
			}
		}
		if stat.Size() != t.req.FileSize {
			msg := fmt.Sprintf("incorrect file size (expected %d, got %d)",
				t.req.FileSize, stat.Size())
			t.pip.Logger.Error(msg)
			return &r66.Error{Code: r66.SizeNotAllowed, Detail: msg}
		}
	}
	return nil
}

func (t *transferHandler) checkHash(exp []byte) *r66.Error {
	hash, err := internal.MakeHash(t.pip.Logger, t.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		return &r66.Error{Code: r66.Internal, Detail: "failed to compute file hash"}
	}
	if !bytes.Equal(hash, exp) {
		t.pip.Logger.Errorf("File hash verification failed: hashes do not match")
		return &r66.Error{Code: r66.FinalOp, Detail: "file hash does not match expected value"}
	}
	return nil
}
