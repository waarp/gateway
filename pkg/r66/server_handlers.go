package r66

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-r66/r66"
	r66utils "code.waarp.fr/waarp-r66/r66/utils"
	"golang.org/x/crypto/bcrypt"
)

type authHandler struct {
	*Service
	ctx context.Context
}

func (a *authHandler) ValidAuth(authent *r66.Authent) (r66.SessionHandler, error) {
	if authent.FinalHash && !strings.EqualFold(authent.Digest, "SHA-256") {
		a.logger.Warningf("Unknown hash digest '%s'", authent.Digest)
		return nil, &r66.Error{Code: r66.Unimplemented, Detail: "unknown final hash digest"}
	}

	acc := model.LocalAccount{Login: authent.Login, LocalAgentID: a.agent.ID}
	if err := a.db.Get(&acc); err != nil {
		if err == database.ErrNotFound {
			a.logger.Warningf("Unknown account '%s'", authent.Login)
			return nil, &r66.Error{Code: r66.BadAuthent, Detail: "incorrect credentials"}
		}
		a.logger.Errorf("Failed to retrieve credentials from database: %s", err)
		return nil, &r66.Error{Code: r66.Internal, Detail: "authentication failed: " +
			fmt.Sprintf("%#v", err)}
	}

	if bcrypt.CompareHashAndPassword(acc.Password, authent.Password) != nil {
		a.logger.Warningf("Account '%s' authenticated with wrong password", authent.Login)
		return nil, &r66.Error{Code: r66.BadAuthent, Detail: "incorrect credentials"}
	}

	ses := sessionHandler{
		authHandler: a,
		account:     &acc,
		hasHash:     authent.FinalHash,
		hasFileSize: authent.Filesize,
	}
	return &ses, nil
}

type sessionHandler struct {
	*authHandler

	account              *model.LocalAccount
	hasFileSize, hasHash bool
}

func (s *sessionHandler) GetTransferInfo(ID int64, isClient bool) (*r66.TransferInfo, error) {
	panic("implement me")
}

func (s *sessionHandler) GetFileInfo(rule string, filename string) ([]r66.FileInfo, error) {
	panic("implement me")
}

func (s *sessionHandler) GetBandwidth() (*r66.Bandwidth, error) {
	panic("implement me")
}

func (s *sessionHandler) SetBandwidth(*r66.Bandwidth) (*r66.Bandwidth, error) {
	panic("implement me")
}

func (s *sessionHandler) parseRuleMode(r *r66.Request) (bool, *model.Rule, error) {
	var isMD5 bool
	rule := model.Rule{Name: r.Rule}
	switch r.Mode {
	case 1:
	case 2:
		rule.IsSend = true
	case 3:
		isMD5 = true
	case 4:
		isMD5 = true
		rule.IsSend = true
	default:
		return false, nil, &r66.Error{Code: r66.Unimplemented, Detail: "unknown transfer mode"}
	}
	return isMD5, &rule, nil
}

func (s *sessionHandler) getRule(rule *model.Rule) error {
	if err := s.db.Get(rule); err != nil {
		if err == database.ErrNotFound {
			s.logger.Warningf("Requested transfer rule '%s' does not exist", rule.Name)
			return &r66.Error{Code: r66.IncorrectCommand, Detail: "rule does not exist"}
		}
		s.logger.Errorf("Failed to retrieve transfer rule: %s", err)
		return &r66.Error{Code: r66.Internal, Detail: "failed to retrieve rule"}
	}
	return nil
}

func (s *sessionHandler) ValidRequest(request *r66.Request) (r66.TransferHandler, error) {
	if request.Filepath == "" {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "missing filepath"}
	}
	if request.Block == 0 {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "missing block size"}
	}

	isMD5, rule, err := s.parseRuleMode(request)
	if err != nil {
		return nil, err
	}

	if err := s.getRule(rule); err != nil {
		return nil, err
	}

	if !rule.IsSend && s.hasFileSize && request.FileSize < 0 {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "missing file size"}
	}

	trans := model.Transfer{
		RemoteTransferID: fmt.Sprint(request.ID),
		RuleID:           rule.ID,
		IsServer:         true,
		AgentID:          s.agent.ID,
		AccountID:        s.account.ID,
		SourceFile:       path.Base(request.Filepath),
		DestFile:         path.Base(request.Filepath),
		Start:            time.Now(),
		Status:           types.StatusRunning,
	}

	tStream, err := pipeline.NewTransferStream(s.ctx, s.logger, s.db, s.paths, &trans)
	if err != nil {
		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to initiate transfer"}
	}

	if rule.IsSend && s.hasFileSize {
		stats, err := os.Stat(utils.DenormalizePath(tStream.Transfer.TrueFilepath))
		if err != nil {
			return nil, &r66.Error{Code: r66.Internal, Detail: err.Error()}
		}
		request.FileSize = stats.Size()
	}
	setProgress(tStream.Transfer, request)

	file := &stream{tStream}

	handler := transferHandler{
		sessionHandler: s,
		file:           file,
		isMD5:          isMD5,
		fileSize:       request.FileSize,
	}
	return &handler, nil
}

type transferHandler struct {
	*sessionHandler
	file     *stream
	isMD5    bool
	fileSize int64
}

func (t *transferHandler) UpdateTransferInfo(info *r66.UpdateInfo) error {
	if t.file.Transfer.Step >= types.StepData {
		return &r66.Error{
			Code:   r66.IncorrectCommand,
			Detail: "cannot update transfer info after data transfer started",
		}
	}

	if info.Filename != "" {
		filename := path.Base(info.Filename)
		newPath := path.Join(path.Dir(t.file.Transfer.TrueFilepath), filename)

		if t.file.Rule.IsSend {
			if err := os.Rename(t.file.Transfer.TrueFilepath, newPath); err != nil {
				t.file.Logger.Errorf("Failed to rename R66 file: %s", err)
				return &r66.Error{
					Code:   r66.FileNotAllowed,
					Detail: "failed to rename file",
				}
			}
		}

		t.file.Transfer.TrueFilepath = newPath
		t.file.Transfer.SourceFile = filename
		t.file.Transfer.DestFile = filename
	}

	if info.FileSize != 0 {
		t.fileSize = info.FileSize
	}

	if err := t.db.Update(t.file.Transfer); err != nil {
		t.logger.Errorf("Failed to update transfer: %s", err)
		return &r66.Error{Code: r66.Internal, Detail: "database error"}
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
	if err := t.file.PreTasks(); err != nil {
		if err.Kind == model.KindTransfer {
			return &r66.Error{Code: r66.ExternalOperation, Detail: err.Cause.Details}
		}
		return &r66.Error{Code: r66.Internal, Detail: "pre-tasks failed"}
	}
	return nil
}

func (t *transferHandler) GetStream() (r66utils.ReadWriterAt, error) {
	if err := t.file.Start(); err != nil {
		return nil, &r66.Error{Code: r66.FileNotAllowed, Detail: "failed to open file"}
	}
	return t.file, nil
}

func (t *transferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	if t.file.Close() != nil {
		return &r66.Error{Code: r66.FinalOp, Detail: "failed to finalize transfer"}
	}
	if t.file.Transfer.Step > types.StepData {
		return nil
	}

	if !t.file.Rule.IsSend {
		if t.hasFileSize && int64(t.file.Transfer.Progress) != t.fileSize {
			return &r66.Error{
				Code: r66.SizeNotAllowed,
				Detail: fmt.Sprintf("incorrect file size (expected %d, got %d)",
					t.fileSize, t.file.Transfer.Progress),
			}
		}
		if t.hasHash {
			if len(end.Hash) != 32 {
				return &r66.Error{Code: r66.FinalOp, Detail: "invalid file hash"}
			}
			if err := checkHash(t.file.Transfer.TrueFilepath, end.Hash); err != nil {
				return &r66.Error{
					Code:   r66.FinalOp,
					Detail: err.Error(),
				}
			}
		}
	} else {
		if t.hasHash {
			hash, err := makeHash(t.file.Transfer.TrueFilepath)
			if err != nil {
				return &r66.Error{Code: r66.FinalOp, Detail: "failed to calculate file hash"}
			}
			end.Hash = hash
		}
	}

	if t.file.Move() != nil {
		return &r66.Error{Code: r66.FinalOp, Detail: "failed to finalize transfer"}
	}

	return nil
}

func (t *transferHandler) RunPostTask() error {
	if err := t.file.PostTasks(); err != nil {
		if err.Kind == model.KindTransfer {
			return &r66.Error{Code: r66.ExternalOperation, Detail: err.Cause.Details}
		}
		return &r66.Error{Code: r66.Internal, Detail: "pre-tasks failed"}
	}
	return nil
}

func (t *transferHandler) ValidEndRequest() error {
	t.file.Transfer.Step = types.StepNone
	t.file.Transfer.TaskNumber = 0
	t.file.Transfer.Status = types.StatusDone
	if err := t.file.Archive(); err != nil {
		return &r66.Error{Code: r66.Internal, Detail: "failed to archive transfer"}
	}
	return nil
}

func (t *transferHandler) RunErrorTask(protoErr error) error {
	_ = t.file.File.Close()

	if t.file.Transfer.Error.Code == types.TeOk {
		if r66Err, ok := protoErr.(*r66.Error); ok {
			t.file.Transfer.Error.Code = types.FromR66Code(r66Err.Code)
			t.file.Transfer.Error.Details = r66Err.Detail
		} else {
			t.file.Transfer.Error.Code = types.TeUnknownRemote
			t.file.Transfer.Error.Details = protoErr.Error()
		}
	}

	t.file.ErrorTasks()
	t.file.Transfer.Status = types.StatusError
	if err := t.db.Update(t.file.Transfer); err != nil {
		t.logger.Criticalf("Failed to update transfer status to '%s': %s",
			types.StatusError, err)
		return &r66.Error{Code: r66.Internal, Detail: "failed to archive transfer"}
	}
	return nil
}
