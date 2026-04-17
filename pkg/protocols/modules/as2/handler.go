package as2

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"code.waarp.fr/lib/as2"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type accountCtxKey struct{}

func setUserCtxVal(r *http.Request, acc *model.LocalAccount) {
	*r = *r.WithContext(context.WithValue(r.Context(), accountCtxKey{}, acc))
}

func getUserCtxVal(ctx context.Context) *model.LocalAccount {
	val := ctx.Value(accountCtxKey{})
	if val == nil {
		return nil
	}

	//nolint:forcetypeassert //assertion is guaranteed to succeed
	return val.(*model.LocalAccount)
}

func (s *server) auth(r *http.Request) bool {
	//nolint:canonicalheader //spec says "AS2-From"
	as2from := r.Header.Get("AS2-From")
	if as2from == "" {
		s.logger.Warning(`Missing "AS2-From" header`)

		return false
	}

	// Retrieve account
	acc, err := s.newPartnerStore().getAccount(as2from)
	if err != nil {
		return false
	}

	setUserCtxVal(r, acc)

	// Retrieve account password from db
	creds, err := acc.GetCredentials(s.db, auth.Password)
	if err != nil {
		s.logger.Errorf("Failed to retrieve password for account %q: %v", acc.Login, err)

		return false
	}

	// If no password in DB, skip HTTP authent
	if len(creds) == 0 {
		return true
	}

	// Retrieve HTTP request authent
	login, pswd, ok := r.BasicAuth()
	if !ok {
		s.logger.Warning("HTTP authentication failed: missing Authorization header")

		return false
	}

	// If HTTP login does not match AS2-From, fail authent
	if login != as2from {
		s.logger.Warningf("HTTP authentication failed: header login mismatch (AS2-From: %q, Authorization: %q",
			as2from, login)

		return false
	}

	// Check password
	for _, cred := range creds {
		if utils.IsHashOf(cred.Value, pswd) {
			return true
		}
	}

	// No match found, fail authent
	return false
}

func (s *server) handle(ctx context.Context, filename string, payload []byte) error {
	acc := getUserCtxVal(ctx)
	if acc == nil {
		return as2.ErrMissingAS2Identity
	}

	rule, err := protoutils.GetClosestRule(s.db, s.logger, s.agent, acc, filename, false)
	if err != nil {
		return err
	}

	if rule.IsSend {
		return ErrTransferPull
	}

	filename = strings.TrimLeft(filename, "/")
	filename = strings.TrimPrefix(filename, rule.Path)
	filename = strings.TrimLeft(filename, "/")

	trans, err := s.getTransfer(filename, rule, acc)
	if err != nil {
		return err
	}

	return s.runTransfer(ctx, trans, payload)
}

func (s *server) getTransfer(filename string, rule *model.Rule, acc *model.LocalAccount,
) (*model.Transfer, error) {
	trans, err := pipeline.GetOldTransferByFilename(s.db, filename, 0, acc, rule)
	if err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		s.logger.Errorf("Failed to retrieve transfer by name: %v", err)

		return nil, ErrDatabase
	}

	trans, err = pipeline.GetAvailableTransferByFilename(s.db, filename, "", acc, rule)
	if err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		s.logger.Errorf("Failed to retrieve transfer by name: %v", err)

		return nil, ErrDatabase
	}

	return pipeline.MakeServerTransfer("", filename, acc, rule), nil
}

func (s *server) runTransfer(parent context.Context, trans *model.Transfer, payload []byte) error {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)

	pip, pipErr := s.initPipeline(cancel, trans)
	if pipErr != nil {
		return pipErr
	}

	return utils.RunWithCtx(ctx, func() error {
		if err := pip.PreTasks(); err != nil {
			return err
		}

		file, fErr := pip.StartData()
		if fErr != nil {
			return fErr
		}

		if _, err := file.Write(payload); err != nil {
			return fmt.Errorf("failed to write payload: %w", err)
		}

		if err := pip.EndData(); err != nil {
			return err
		}

		if err := pip.PostTasks(); err != nil {
			return err
		}

		//nolint:revive //can't just return the error because it's not of type error
		if err := pip.EndTransfer(); err != nil {
			return err
		}

		return nil
	})
}

func (s *server) initPipeline(cancel context.CancelCauseFunc, trans *model.Transfer,
) (*pipeline.Pipeline, *pipeline.Error) {
	pip, err := pipeline.NewServerPipeline(s.db, s.logger, trans, snmp.GlobalService)
	if err != nil {
		s.logger.Errorf("Failed to initialize transfer pipeline: %v", err)

		return nil, err
	}

	sigPause := pipeline.NewError(types.TeStopped, "transfer paused by user")
	sigShutdown := pipeline.NewError(types.TeShuttingDown, "service is shutting down")
	sigCancel := pipeline.NewError(types.TeCanceled, "transfer canceled by user")

	sendSig := func(err error) func(context.Context) error {
		return func(context.Context) error {
			cancel(err)

			return nil
		}
	}

	pip.Trace = s.tracer()
	pip.SetInterruptionHandlers(
		sendSig(sigPause),
		sendSig(sigShutdown),
		sendSig(sigCancel),
	)

	return pip, nil
}
