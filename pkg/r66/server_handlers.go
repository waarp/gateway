package r66

import (
	"path"
	"path/filepath"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-r66/r66"
	r66utils "code.waarp.fr/waarp-r66/r66/utils"
	"golang.org/x/crypto/bcrypt"
)

type authHandler struct {
	*Service
}

func (a *authHandler) ValidAuth(authent *r66.Authent) (r66.SessionHandler, error) {
	if authent.FinalHash && !strings.EqualFold(authent.Digest, "SHA-256") {
		a.logger.Warningf("Unknown hash digest '%s'", authent.Digest)
		return nil, &r66.Error{Code: r66.Unimplemented, Detail: "unknown final hash digest"}
	}

	var acc model.LocalAccount
	if err := a.db.Get(&acc, "login=? AND local_agent_id=?", authent.Login,
		a.agent.ID).Run(); err != nil {
		if database.IsNotFound(err) {
			a.logger.Warningf("Unknown account '%s'", authent.Login)
			return nil, &r66.Error{Code: r66.BadAuthent, Detail: "incorrect credentials"}
		}
		a.logger.Errorf("Failed to retrieve credentials from database: %s", err)
		return nil, r66ErrDatabase
	}

	if bcrypt.CompareHashAndPassword(acc.PasswordHash, authent.Password) != nil {
		a.logger.Warningf("Account '%s' authenticated with wrong password %s", authent.Login, string(authent.Password))
		return nil, &r66.Error{Code: r66.BadAuthent, Detail: "incorrect credentials"}
	}

	ses := sessionHandler{
		authHandler: a,
		account:     &acc,
		conf:        authent,
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
	isSend := r66.IsRecv(req.Mode)

	rule, err := s.getRule(req.Rule, isSend)
	if err != nil {
		return nil, err
	}

	if isSend {
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
		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to initiate transfer"}
	}

	if err := s.getSize(req, rule, trans); err != nil {
		pip.SetError(err)
		return nil, internal.ToR66Error(err)
	}
	//TODO: add transfer info to DB

	handler := transferHandler{
		pip:   pip,
		inter: inter,
		conf:  s.conf,
		req:   req,
	}
	return &handler, nil
}

type transferHandler struct {
	pip   *pipeline.ServerPipeline
	inter *interruptionHandler
	conf  *r66.Authent
	req   *r66.Request
}

func (t *transferHandler) GetHash() ([]byte, error) {
	hash, err := internal.MakeHash(t.pip.Logger, t.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		t.pip.SetError(err)
		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to compute file hash"}
	}
	return hash, nil
}

func (t *transferHandler) UpdateTransferInfo(info *r66.UpdateInfo) error {
	if t.pip.TransCtx.Transfer.Step >= types.StepData {
		return nil //cannot update transfer info after data transfer started
	}

	var cols []string
	if info.Filename != "" {
		old := t.pip.TransCtx.Transfer.LocalPath
		filename := path.Base(info.Filename)
		newPath := filepath.Join(filepath.Dir(old), filename)

		t.pip.TransCtx.Transfer.LocalPath = newPath
		cols = append(cols, "local_path", "remote_path")
	}

	if info.FileSize != 0 {
		t.req.FileSize = info.FileSize
	}

	if err := t.pip.DB.Update(t.pip.TransCtx.Transfer).Cols(cols...).Run(); err != nil {
		t.pip.Logger.Errorf("Failed to update transfer info: %s", err)
		t.pip.SetError(errDatabase)
		return r66ErrDatabase
	}

	/* TODO: de-comment once TransferInfo are used
	tid := t.file.Transfer.ID
	if info.FileInfo != nil {
		oldInfo, err := t.file.Transfer.GetTransferInfo(t.db)
		if err != nil {
			t.logger.Errorf("Failed to retrieve transfer info: %s", err)
			return &r66.Error{Code: r66.Internal, Detail: "database error"}
		}
		for key, val := range info.FileInfo {
			ti := &model.TransferInfo{TransferID: tid, Name: key, Value: fmt.Sprint(val)}
			var dbErr error
			if _, ok := oldInfo[key]; ok {
				dbErr = t.db.Execute("UPDATE transfer_info SET value=? WHERE transfer_id=? AND name=?",
					ti.Value, ti.TransferID, ti.Name)
			} else {
				dbErr = t.db.Create(ti)
			}
			if dbErr != nil {
				t.logger.Errorf("Failed to update transfer info: %s", err)
				return &r66.Error{Code: r66.Internal, Detail: "database error"}
			}
		}
	} */

	return nil
}

func (t *transferHandler) RunPreTask() error {
	if err := t.pip.PreTasks(); err != nil {
		return internal.ToR66Error(err)
	}
	return nil
}

func (t *transferHandler) GetStream() (r66utils.ReadWriterAt, error) {
	file, err := t.pip.StartData()
	if err != nil {
		return nil, &r66.Error{Code: r66.FileNotAllowed, Detail: "failed to open file"}
	}
	return file, nil
}

func (t *transferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	if t.pip.EndData() != nil {
		return &r66.Error{Code: r66.FinalOp, Detail: "failed to finalize transfer"}
	}

	if !t.pip.TransCtx.Rule.IsSend {
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
	t.pip.TransCtx.Transfer.Step = types.StepNone
	t.pip.TransCtx.Transfer.TaskNumber = 0
	t.pip.TransCtx.Transfer.Status = types.StatusDone
	if err := t.pip.EndTransfer(); err != nil {
		return &r66.Error{Code: r66.Internal, Detail: "failed to archive transfer"}
	}
	if testEndSignal != nil {
		testEndSignal(t.pip)
	}
	return nil
}

func (t *transferHandler) RunErrorTask(err error) error {
	tErr := internal.FromR66Error(err, t.pip.Pipeline)
	if tErr != nil {
		t.pip.SetError(tErr)
	}
	if testEndSignal != nil {
		testEndSignal(t.pip)
	}
	return nil
}
