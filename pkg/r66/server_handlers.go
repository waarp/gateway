package r66

import (
	"path"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-r66/r66"
	r66utils "code.waarp.fr/waarp-r66/r66/utils"
)

type authHandler struct {
	*Service
}

func (a *authHandler) ValidAuth(auth *r66.Authent) (r66.SessionHandler, error) {
	select {
	case <-a.shutdown:
		return nil, sigShutdown
	default:
	}

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

	pip, pErr := pipeline.NewServerPipeline(s.db, trans)
	if pErr != nil {
		return nil, internal.ToR66Error(err)
	}

	if err := s.getSize(req, rule, trans); err != nil {
		pip.SetError(err)
		return nil, internal.ToR66Error(err)
	}
	//TODO: add transfer info to DB

	r66Pip := &serverTransfer{
		conf:  s.conf,
		pip:   pip,
		store: utils.NewErrorStorage(),
	}
	s.runningTransfers.Add(trans.ID, r66Pip)

	handler := transferHandler{
		sessionHandler: s,
		trans:          r66Pip,
	}
	return &handler, nil
}

type transferHandler struct {
	*sessionHandler
	trans *serverTransfer
}

func (t *transferHandler) GetHash() ([]byte, error) {
	return t.trans.getHash()
}

func (t *transferHandler) UpdateTransferInfo(info *r66.UpdateInfo) error {
	return t.trans.updTransInfo(info)
}

func (t *transferHandler) RunPreTask() (*r66.UpdateInfo, error) {
	return t.trans.runPreTask()
}

func (t *transferHandler) GetStream() (r66utils.ReadWriterAt, error) {
	return t.trans.getStream()
}

func (t *transferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	return t.trans.validEndTransfer(end)
}

func (t *transferHandler) RunPostTask() error {
	return t.trans.runPostTask()
}

func (t *transferHandler) ValidEndRequest() error {
	defer t.runningTransfers.Delete(t.trans.pip.TransCtx.Transfer.ID)
	return t.trans.validEndRequest()
}

func (t *transferHandler) RunErrorTask(err error) error {
	defer t.runningTransfers.Delete(t.trans.pip.TransCtx.Transfer.ID)
	return t.trans.runErrorTasks(err)
}
