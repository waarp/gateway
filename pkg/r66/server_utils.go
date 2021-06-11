package r66

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"time"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"

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
)

func (a *authHandler) certAuth(auth *r66.Authent) (*model.LocalAccount, *r66.Error) {
	if auth.TLS == nil || len(auth.TLS.PeerCertificates) == 0 {
		return nil, nil
	}

	sign := utils.MakeSignature(auth.TLS.PeerCertificates[0])
	var crypto model.Crypto
	if err := a.db.Get(&crypto, "owner_type=? AND signature=?", "local_accounts",
		sign).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, &r66.Error{Code: r66.BadAuthent, Detail: "unknown certificate"}
		}
		a.logger.Errorf("Failed to retrieve client certificate: %s", err)
		return nil, r66ErrDatabase
	}

	var acc model.LocalAccount
	if err := a.db.Get(&acc, "id=?", crypto.OwnerID).Run(); err != nil {
		a.logger.Errorf("Failed to retrieve client account: %s", err)
		return nil, r66ErrDatabase
	}
	return &acc, nil
}

func (a *authHandler) passwordAuth(auth *r66.Authent) (*model.LocalAccount, *r66.Error) {
	if auth.Login == "" || len(auth.Password) == 0 {
		return nil, nil
	}

	var acc model.LocalAccount
	if err := a.db.Get(&acc, "login=? AND local_agent_id=?", auth.Login,
		a.agent.ID).Run(); err != nil {
		if !database.IsNotFound(err) {
			a.logger.Errorf("Failed to retrieve credentials from database: %s", err)
			return nil, r66ErrDatabase
		}
	}

	if bcrypt.CompareHashAndPassword(acc.PasswordHash, auth.Password) != nil {
		if acc.Login == "" {
			a.logger.Warningf("Authentication failed with unknown account '%s'", auth.Login)
		} else {
			a.logger.Warningf("Account '%s' authenticated with wrong password %s",
				auth.Login, string(auth.Password))
		}
		return nil, &r66.Error{Code: r66.BadAuthent, Detail: "incorrect credentials"}
	}
	return &acc, nil
}

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
	if !req.IsRecv && s.conf.Filesize && req.FileSize < 0 {
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
		return errDatabase
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
		if stat.Size() != t.pip.TransCtx.Transfer.Filesize {
			msg := fmt.Sprintf("incorrect file size (expected %d, got %d)",
				t.pip.TransCtx.Transfer.Filesize, stat.Size())
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
