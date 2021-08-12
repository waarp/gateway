package r66

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
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

	acc := model.LocalAccount{}
	if err := a.db.Get(&acc, "login=? AND local_agent_id=?", authent.Login,
		a.agent.ID).Run(); err != nil {
		if database.IsNotFound(err) {
			a.logger.Warningf("Unknown account '%s'", authent.Login)
			return nil, &r66.Error{Code: r66.BadAuthent, Detail: "incorrect credentials"}
		}
		a.logger.Errorf("Failed to retrieve credentials from database: %s", err)
		return nil, &r66.Error{Code: r66.Internal, Detail: "authentication failed: " +
			fmt.Sprintf("%#v", err)}
	}

	if bcrypt.CompareHashAndPassword(acc.PasswordHash, authent.Password) != nil {
		a.logger.Warningf("Account '%s' authenticated with wrong password %s", authent.Login, string(authent.Password))
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

func (s *sessionHandler) GetTransferInfo(int64, bool) (*r66.TransferInfo, error) {
	return nil, &r66.Error{Code: r66.Unimplemented, Detail: "command not implemented"}
}

func (s *sessionHandler) GetFileInfo(string, string) ([]r66.FileInfo, error) {
	return nil, &r66.Error{Code: r66.Unimplemented, Detail: "command not implemented"}
}

func (s *sessionHandler) GetBandwidth() (*r66.Bandwidth, error) {
	return nil, &r66.Error{Code: r66.Unimplemented, Detail: "command not implemented"}
}

func (s *sessionHandler) SetBandwidth(*r66.Bandwidth) (*r66.Bandwidth, error) {
	return nil, &r66.Error{Code: r66.Unimplemented, Detail: "command not implemented"}
}

func (s *sessionHandler) parseRuleMode(r *r66.Request) (isMD5, isSend bool, err error) {
	switch r.Mode {
	case 1:
	case 2:
		isSend = true
	case 3:
		isMD5 = true
	case 4:
		isMD5 = true
		isSend = true
	default:
		return false, false, &r66.Error{Code: r66.Unimplemented, Detail: "unknown transfer mode"}
	}
	return
}

func (s *sessionHandler) getRule(ruleName string, isSend bool) (*model.Rule, error) {
	rule := &model.Rule{}
	if err := s.db.Get(rule, "name=? AND send=?", ruleName, isSend).Run(); err != nil {
		if database.IsNotFound(err) {
			s.logger.Warningf("Requested transfer rule '%s' does not exist", rule.Name)
			return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "rule does not exist"}
		}
		s.logger.Errorf("Failed to retrieve transfer rule: %s", err)
		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to retrieve rule"}
	}
	return rule, nil
}

func (s *sessionHandler) ValidRequest(request *r66.Request) (r66.TransferHandler, error) {
	if request.Filepath == "" {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "missing filepath"}
	}
	if request.Block == 0 {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "missing block size"}
	}

	isMD5, isSend, err := s.parseRuleMode(request)
	if err != nil {
		return nil, err
	}

	rule, err := s.getRule(request.Rule, isSend)
	if err != nil {
		return nil, err
	}

	if !isSend && s.hasFileSize && request.FileSize < 0 {
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

	s.logger.Infof("Transfer of file %s was requested by %s, using rule %s",
		trans.SourceFile, s.account.Login, rule.Name)

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

	//TODO: add transfer info to DB
	stream := &stream{tStream}

	handler := transferHandler{
		sessionHandler: s,
		file:           stream,
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

	if !t.file.Rule.IsSend {
		if info.Filename != "" {
			filename := path.Base(info.Filename)
			newPath := path.Join(path.Dir(t.file.Transfer.TrueFilepath), filename)

			t.file.Transfer.TrueFilepath = newPath
			t.file.Transfer.SourceFile = filename
			t.file.Transfer.DestFile = filename
		}
		if info.FileSize != 0 {
			t.fileSize = info.FileSize
		}

		if err := t.db.Update(t.file.Transfer).Run(); err != nil {
			t.logger.Errorf("Failed to update transfer: %s", err)
			return &r66.Error{Code: r66.Internal, Detail: "database error"}
		}
	} else {
		info.Filename = t.file.Transfer.SourceFile

		fileInfo, err := os.Stat(utils.DenormalizePath(t.file.Transfer.TrueFilepath))
		if err != nil {
			t.logger.Errorf("Failed to retrieve file size: %s", err)
			return &r66.Error{Code: r66.Internal, Detail: "failed to retrieve file size"}
		}
		t.fileSize = fileInfo.Size()
		info.FileSize = fileInfo.Size()
	}

	return nil
}

func (t *transferHandler) RunPreTask() error {
	if err := t.file.PreTasks(); err != nil {
		return toR66Error(err)
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
		if t.hasFileSize {
			stat, err := os.Stat(utils.DenormalizePath(t.file.Transfer.TrueFilepath))
			if err != nil {
				t.logger.Errorf("Failed to retrieve file info: %s", err)
				return &r66.Error{
					Code:   r66.Internal,
					Detail: "failed to retrieve file info",
				}
			}
			if stat.Size() != t.fileSize {
				t.logger.Errorf("Incorrect file size (expected %d, got %d)",
					t.fileSize, stat.Size())
				return &r66.Error{
					Code: r66.SizeNotAllowed,
					Detail: fmt.Sprintf("incorrect file size (expected %d, got %d)",
						t.fileSize, stat.Size()),
				}
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
		return toR66Error(err)
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
	_ = t.file.Close()

	if t.file.Transfer.Error.Code == types.TeOk {
		if r66Err, ok := protoErr.(*r66.Error); ok {
			t.file.Transfer.Error.Code = types.FromR66Code(r66Err.Code)
			t.file.Transfer.Error.Details = r66Err.Detail
		} else {
			t.file.Transfer.Error.Code = types.TeUnknownRemote
			t.file.Transfer.Error.Details = protoErr.Error()
		}
	}
	if err := t.db.Update(t.file.Transfer).Cols("error_code", "error_details").Run(); err != nil {
		t.logger.Criticalf("Failed to update transfer error to '%s': %s",
			t.file.Transfer.Error.Code.String(), err)
		return &r66.Error{Code: r66.Internal, Detail: "failed to archive transfer"}
	}

	t.file.ErrorTasks()
	t.file.Transfer.Status = types.StatusError
	if err := t.db.Update(t.file.Transfer).Cols("status").Run(); err != nil {
		t.logger.Criticalf("Failed to update transfer status to '%s': %s",
			types.StatusError, err)
		return &r66.Error{Code: r66.Internal, Detail: "failed to archive transfer"}
	}
	return nil
}
