package tasks

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrTransferPreregisterNoFile    = errors.New("missing file name")
	ErrTransferPreregisterNoRule    = errors.New("missing rule name")
	ErrTransferPreregisterNoServer  = errors.New("missing server name")
	ErrTransferPreregisterNoAccount = errors.New("missing account name")
	ErrTransferPreregisterNoDueDate = errors.New("missing due date")
)

type TransferPreregister struct {
	File     string     `json:"file"`
	Rule     string     `json:"rule"`
	IsSend   jsonBool   `json:"isSend"`
	Server   string     `json:"server"`
	Account  string     `json:"account"`
	DueDate  time.Time  `json:"dueDate"`
	Info     jsonObject `json:"info"`
	CopyInfo jsonBool   `json:"copyInfo"`

	rule    model.Rule
	server  model.LocalAgent
	account model.LocalAccount
}

func (t *TransferPreregister) ValidateDB(db database.ReadAccess, params map[string]string) error {
	*t = TransferPreregister{}

	if err := utils.JSONConvert(params, t); err != nil {
		return fmt.Errorf("failed to parse the transfer parameters: %w", err)
	}

	if t.File == "" {
		return ErrTransferPreregisterNoFile
	}

	if t.Rule == "" {
		return ErrTransferPreregisterNoRule
	}

	if t.Server == "" {
		return ErrTransferPreregisterNoServer
	}

	if t.Account == "" {
		return ErrTransferPreregisterNoAccount
	}

	if t.DueDate.IsZero() {
		return ErrTransferPreregisterNoDueDate
	}

	if err := db.Get(&t.rule, "name=? AND is_send=?", t.Rule, t.IsSend).Run(); err != nil {
		return fmt.Errorf("failed to retrieve rule %q: %w", t.Rule, err)
	}

	if err := db.Get(&t.server, "name=?", t.Server).Owner().Run(); err != nil {
		return fmt.Errorf("failed to retrieve server %q: %w", t.Server, err)
	}

	if err := db.Get(&t.account, "login=? AND local_agent_id=?", t.Account,
		t.server.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve account %q: %w", t.Account, err)
	}

	return nil
}

func (t *TransferPreregister) Run(_ context.Context, args map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := t.ValidateDB(db, args); err != nil {
		logger.Error(err.Error())

		return err
	}

	transferInfo := map[string]any{}
	if t.CopyInfo {
		transferInfo = transCtx.TransInfo
	}

	maps.Copy(transferInfo, t.Info)

	trans := &model.Transfer{
		Status:         types.StatusAvailable,
		RuleID:         t.rule.ID,
		LocalAccountID: utils.NewNullInt64(t.account.ID),
		Start:          t.DueDate,
	}

	var kind string

	if t.IsSend {
		trans.SrcFilename = t.File
		kind = "download"
	} else {
		trans.DestFilename = t.File
		kind = "upload"
	}

	if err := db.Transaction(func(ses *database.Session) error {
		if err := ses.Insert(trans).Run(); err != nil {
			return fmt.Errorf("failed to insert transfer: %w", err)
		}

		if err := trans.SetTransferInfo(ses, transferInfo); err != nil {
			return fmt.Errorf("failed to set transfer info: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to preregister transfer: %w", err)
	}

	logger.Debugf("Preregistered %s transfer n°%d of file %q, from %q to %q using rule %q",
		kind, trans.ID, t.File, t.Server, t.Account, t.Rule)

	return nil
}
