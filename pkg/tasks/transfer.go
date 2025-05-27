package tasks

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrTransferNoSource             = errors.New(`missing transfer source file parameter ("file")`)
	ErrTransferNoPartner            = errors.New(`missing transfer remote partner parameter ("from" or "to")`)
	ErrTransferNoAccount            = errors.New(`missing transfer account parameter ("as")`)
	ErrTransferNoRule               = errors.New(`missing transfer rule parameter ("rule")`)
	ErrTransferConflictingDirection = errors.New(`transfer cannot have both "to" and "from" parameters`)
)

//nolint:gochecknoglobals //yes, this is very ugly, but there is really no other way to avoid an import cycle
var GetDefaultTransferClient func(db *database.DB, accID int64) (*model.Client, error)

// TransferTask is a task which schedules a new transfer.
type TransferTask struct {
	File     string   `json:"file"`
	To       string   `json:"to"`
	From     string   `json:"from"`
	As       string   `json:"as"`
	Using    string   `json:"using"`
	Rule     string   `json:"rule"`
	Info     jsonMap  `json:"info"`
	CopyInfo jsonBool `json:"copyInfo"`

	NbOfAttempts         jsonInt      `json:"nbOfAttempts"`
	FirstRetryDelay      jsonDuration `json:"firstRetryDelay"`
	RetryIncrementFactor jsonFloat    `json:"retryIncrementFactor"`
}

// Validate checks if the tasks has all the required arguments.
func (t *TransferTask) Validate(args map[string]string) error {
	*t = TransferTask{}

	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse transfer task arguments: %w", err)
	}

	if t.File == "" {
		return ErrTransferNoSource
	}

	if t.To == "" && t.From == "" {
		return ErrTransferNoPartner
	}

	if t.To != "" && t.From != "" {
		return ErrTransferConflictingDirection
	}

	if t.As == "" {
		return ErrTransferNoAccount
	}

	if t.Rule == "" {
		return ErrTransferNoRule
	}

	return nil
}

//nolint:funlen,revive //function is perfectly readable despite being long
func (t *TransferTask) getTransferInfo(db *database.DB,
) (*model.Rule, *model.Client, *model.RemoteAccount, error) {
	var rule model.Rule
	if err := db.Get(&rule, "name=? AND is_send=?", t.Rule, t.To != "").Run(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to retrieve rule %q: %w", t.Rule, err)
	}

	agentName := t.To
	if t.To == "" {
		agentName = t.From
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", agentName).Owner().Run(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to retrieve partner %q: %w", agentName, err)
	}

	var account model.RemoteAccount
	if err := db.Get(&account, "remote_agent_id=? AND login=?", partner.ID, t.As).Run(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to retrieve account %q: %w", t.As, err)
	}

	client := &model.Client{}

	if t.Using == "" {
		var err error
		if client, err = GetDefaultTransferClient(db, account.ID); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to retrieve default transfer client: %w", err)
		}
	} else {
		if err := db.Get(client, "name=?", t.Using).Owner().Run(); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to retrieve client %q: %w", t.Using, err)
		}
	}

	return &rule, client, &account, nil
}

// Run executes the task by scheduling a new transfer with the given parameters.
func (t *TransferTask) Run(_ context.Context, args map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := t.Validate(args); err != nil {
		logger.Errorf("%v", err)

		return err
	}

	rule, client, account, infoErr := t.getTransferInfo(db)
	if infoErr != nil {
		logger.Errorf("%v", infoErr)

		return infoErr
	}

	transferInfo := map[string]any{}
	if t.CopyInfo {
		transferInfo = transCtx.TransInfo
	}

	maps.Copy(transferInfo, t.Info)

	trans := &model.Transfer{
		RuleID:               rule.ID,
		ClientID:             utils.NewNullInt64(client.ID),
		RemoteAccountID:      utils.NewNullInt64(account.ID),
		SrcFilename:          t.File,
		DestFilename:         filepath.Base(t.File),
		RemainingTries:       int8(t.NbOfAttempts),
		NextRetryDelay:       int32(t.FirstRetryDelay.Seconds()),
		RetryIncrementFactor: float32(t.RetryIncrementFactor),
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
		return fmt.Errorf("failed to create transfer: %w", err)
	}

	recipient := fmt.Sprintf("from %q", args["from"])
	if rule.IsSend {
		recipient = fmt.Sprintf("to %q", args["to"])
	}

	logger.Debugf("Programmed new transfer n°%d of file %q, %s as %q using rule %q",
		trans.ID, t.File, recipient, args["as"], args["rule"])

	return nil
}
