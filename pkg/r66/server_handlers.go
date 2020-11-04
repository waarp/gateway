package r66

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-r66/r66"
	"code.waarp.fr/waarp-r66/r66/utils"
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

func (s *sessionHandler) ValidRequest(request *r66.Request) (r66.TransferHandler, error) {
	if request.Filepath == "" {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "missing filepath"}
	}
	if request.Block == 0 {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "missing block size"}
	}

	var isMD5 bool
	rule := model.Rule{Name: request.Rule}
	switch request.Mode {
	case 1:
	case 2:
		rule.IsSend = true
	case 3:
		isMD5 = true
	case 4:
		isMD5 = true
		rule.IsSend = true
	default:
		return nil, &r66.Error{Code: r66.Unimplemented, Detail: "unknown transfer mode"}
	}

	if err := s.db.Get(&rule); err != nil {
		if err == database.ErrNotFound {
			s.logger.Warningf("Requested transfer rule '%s' does not exist", request.Rule)
			return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "rule does not exist"}
		}
		s.logger.Errorf("Failed to retrieve transfer rule: %s", err)
		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to retrieve rule"}
	}

	if !rule.IsSend && s.hasFileSize && request.FileSize == 0 {
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
		Status:           model.StatusRunning,
	}

	tStream, err := pipeline.NewTransferStream(s.ctx, s.logger, s.db, s.paths, &trans)
	if err != nil {
		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to initiate transfer"}
	}

	setProgress(tStream.Transfer, request)

	//TODO: add transfer info to DB
	stream := &stream{tStream}

	handler := transferHandler{sessionHandler: s, stream: stream, isMD5: isMD5, fileSize: request.FileSize}
	return &handler, nil
}

type transferHandler struct {
	*sessionHandler
	stream   *stream
	isMD5    bool
	fileSize uint64
}

func (t *transferHandler) RunPreTask() error {
	if err := t.stream.PreTasks(); err != nil {
		if err.Kind == model.KindTransfer {
			return &r66.Error{Code: r66.ExternalOperation, Detail: err.Cause.Details}
		}
		return &r66.Error{Code: r66.Internal, Detail: "pre-tasks failed"}
	}
	return nil
}

func (t *transferHandler) GetStream() (utils.ReadWriterAt, error) {
	if err := t.stream.Start(); err != nil {
		return nil, &r66.Error{Code: r66.FileNotAllowed, Detail: "failed to open file"}
	}
	return t.stream, nil
}

func (t *transferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	if t.stream.Close() != nil {
		return &r66.Error{Code: r66.FinalOp, Detail: "failed to finalize transfer"}
	}
	if t.stream.Move() != nil {
		return &r66.Error{Code: r66.FinalOp, Detail: "failed to finalize transfer"}
	}

	if !t.stream.Rule.IsSend {
		if t.hasFileSize && t.stream.Transfer.Progress != t.fileSize {
			return &r66.Error{
				Code: r66.SizeNotAllowed,
				Detail: fmt.Sprintf("incorrect file size (expected %d, got %d)",
					t.fileSize, t.stream.Transfer.Progress),
			}
		}
		if t.hasHash {
			if len(end.Hash) != 32 {
				return &r66.Error{Code: r66.FinalOp, Detail: "invalid file hash"}
			}
			if checkHash(t.stream.Transfer.TrueFilepath, []byte(end.Hash)) != nil {
				return &r66.Error{
					Code:   r66.FinalOp,
					Detail: "file hash does not match expected value",
				}
			}
		}
	}

	return nil
}

func (t *transferHandler) RunPostTask() error {
	if err := t.stream.PostTasks(); err != nil {
		if err.Kind == model.KindTransfer {
			return &r66.Error{Code: r66.ExternalOperation, Detail: err.Cause.Details}
		}
		return &r66.Error{Code: r66.Internal, Detail: "pre-tasks failed"}
	}
	return nil
}

func (t *transferHandler) ValidEndRequest() error {
	t.stream.Transfer.Step = model.StepNone
	t.stream.Transfer.TaskNumber = 0
	t.stream.Transfer.Status = model.StatusDone
	if err := t.stream.Archive(); err != nil {
		return &r66.Error{Code: r66.Internal, Detail: "failed to archive transfer"}
	}
	return nil
}

func (t *transferHandler) RunErrorTask(protoErr error) error {
	_ = t.stream.File.Close()

	if t.stream.Transfer.Error.Code == model.TeOk {
		if r66Err, ok := protoErr.(*r66.Error); ok {
			t.stream.Transfer.Error.Code = model.FromR66Code(r66Err.Code)
			t.stream.Transfer.Error.Details = r66Err.Detail
		} else {
			t.stream.Transfer.Error.Code = model.TeUnknownRemote
			t.stream.Transfer.Error.Details = protoErr.Error()
		}
	}

	t.stream.ErrorTasks()
	t.stream.Transfer.Status = model.StatusError
	if err := t.db.Update(t.stream.Transfer); err != nil {
		t.logger.Criticalf("Failed to update transfer status to '%s': %s",
			model.StatusError, err)
		return &r66.Error{Code: r66.Internal, Detail: "failed to archive transfer"}
	}
	return nil
}
