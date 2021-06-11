package r66

import (
	"path"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-r66/r66"
	r66utils "code.waarp.fr/waarp-r66/r66/utils"
)

type authHandler struct {
	*Service
}

func (a *authHandler) ValidAuth(auth *r66.Authent) (r66.SessionHandler, error) {
	if auth.FinalHash && !strings.EqualFold(auth.Digest, "SHA-256") {
		a.logger.Warningf("Unknown hash digest '%s'", auth.Digest)
		return nil, &r66.Error{Code: r66.Unimplemented, Detail: "unknown final hash digest"}
	}

	var certAcc, pwdAcc *model.LocalAccount
	var err *r66.Error
	if certAcc, err = a.certAuth(auth); err != nil {
		return nil, err
	}
	if pwdAcc, err = a.passwordAuth(auth); err != nil {
		return nil, err
	}
	if certAcc == nil && pwdAcc == nil {
		return nil, &r66.Error{Code: r66.BadAuthent, Detail: "missing credentials"}
	}

	acc := certAcc
	if certAcc == nil {
		acc = pwdAcc
	} else if pwdAcc != nil && certAcc.ID != pwdAcc.ID {
		return nil, &r66.Error{Code: r66.BadAuthent,
			Detail: "the given certificate does not match the given login"}
	}

	ses := sessionHandler{
		authHandler: a,
		account:     acc,
		conf:        auth,
	}
	return &ses, nil
}

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

	if req.IsRecv {
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
	if err := s.setProgress(req, trans); err != nil {
		return nil, err
	}

	inter := &interruptionHandler{c: make(chan *r66.Error)}
	pip, pErr := pipeline.NewServerPipeline(s.db, trans, inter)
	if pErr != nil {
		return nil, internal.ToR66Error(err)
	}

	if err := s.getSize(req, rule, trans); err != nil {
		pip.SetError(err)
		return nil, internal.ToR66Error(err)
	}
	//TODO: add transfer info to DB

	s.runningTransfers.Add(trans.ID, pip)

	handler := transferHandler{
		sessionHandler: s,
		pip:            pip,
		inter:          inter,
	}
	return &handler, nil
}

type transferHandler struct {
	*sessionHandler
	pip   *pipeline.ServerPipeline
	inter *interruptionHandler
}

func (t *transferHandler) GetHash() ([]byte, error) {
	hash, err := internal.MakeHash(t.pip.Logger, t.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		t.pip.SetError(err)
		return nil, internal.ToR66Error(err)
	}
	return hash, nil
}

func (t *transferHandler) UpdateTransferInfo(info *r66.UpdateInfo) error {
	return internal.UpdateServerInfo(info, t.pip.Pipeline)
}

func (t *transferHandler) RunPreTask() (*r66.UpdateInfo, error) {
	if err := t.pip.PreTasks(); err != nil {
		return nil, internal.ToR66Error(err)
	}
	var info *r66.UpdateInfo
	if t.pip.TransCtx.Rule.IsSend {
		info = &r66.UpdateInfo{
			Filename: strings.TrimPrefix(t.pip.TransCtx.Transfer.RemotePath, "/"),
			FileSize: t.pip.TransCtx.Transfer.Filesize,
			FileInfo: &r66.TransferData{},
		}
	}
	return info, nil
}

func (t *transferHandler) GetStream() (r66utils.ReadWriterAt, error) {
	file, err := t.pip.StartData()
	if err != nil {
		return nil, internal.ToR66Error(err)
	}
	return file, nil
}

func (t *transferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	if t.pip.Stream == nil {
		_, err := t.pip.StartData()
		if err != nil {
			return internal.ToR66Error(err)
		}
	}
	if err := t.pip.EndData(); err != nil {
		return internal.ToR66Error(err)
	}

	if !t.pip.TransCtx.Rule.IsSend && t.pip.TransCtx.Transfer.Step == types.StepData {
		if err := t.checkSize(); err != nil {
			return err
		}
		if err := t.checkHash(end.Hash); err != nil {
			return err
		}
	}

	return nil
}

func (t *transferHandler) RunPostTask() error {
	if err := t.pip.PostTasks(); err != nil {
		return internal.ToR66Error(err)
	}
	return nil
}

func (t *transferHandler) ValidEndRequest() error {
	defer t.runningTransfers.Delete(t.pip.TransCtx.Transfer.ID)
	t.pip.TransCtx.Transfer.Step = types.StepNone
	t.pip.TransCtx.Transfer.TaskNumber = 0
	t.pip.TransCtx.Transfer.Status = types.StatusDone
	if err := t.pip.EndTransfer(); err != nil {
		return internal.ToR66Error(err)
	}
	return nil
}

func (t *transferHandler) RunErrorTask(err error) error {
	defer t.runningTransfers.Delete(t.pip.TransCtx.Transfer.ID)
	tErr := internal.FromR66Error(err, t.pip.Pipeline)
	if tErr != nil {
		t.pip.SetError(tErr)
	}
	return nil
}
