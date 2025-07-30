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

//nolint:gochecknoglobals //yes, this is very ugly, but there is really no other way to avoid an import cycle
var GetDefaultTransferClient func(db *database.DB, protocol string) (*model.Client, error)

var (
	ErrTransferNoFile       = errors.New(`missing transfer source file`)
	ErrTransferNoPartner    = errors.New(`missing transfer remote partner`)
	ErrTransferBothPartners = errors.New(`cannot have both "to" and "from" arguments`)
	ErrTransferNoAccount    = errors.New(`missing transfer account`)
	ErrTransferNoRule       = errors.New(`missing transfer rule`)

	ErrTransferPartnerNotFound = errors.New("transfer partner not found")
	ErrTransferAccountNotFound = errors.New("transfer account not found")
	ErrTransferRuleNotFound    = errors.New("transfer rule not found")
	ErrTransferClientNotFound  = errors.New("transfer client not found")
)

// TransferTask is a task which schedules a new transfer.
type TransferTask struct {
	File                 string       `json:"file"`
	Output               string       `json:"output"`
	To                   string       `json:"to"`
	From                 string       `json:"from"`
	As                   string       `json:"as"`
	Using                string       `json:"using"`
	Rule                 string       `json:"rule"`
	Info                 jsonObject   `json:"info"`
	CopyInfo             jsonBool     `json:"copyInfo"`
	NbOfAttempts         jsonInt      `json:"nbOfAttempts"`
	FirstRetryDelay      jsonDuration `json:"firstRetryDelay"`
	RetryIncrementFactor jsonFloat    `json:"retryIncrementFactor"`

	partner model.RemoteAgent
	account model.RemoteAccount
	rule    model.Rule
	client  model.Client
}

func (t *TransferTask) parseArgs(db database.ReadAccess, args map[string]string) error {
	*t = TransferTask{}

	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse the transfer task arguments: %w", err)
	}

	if t.File == "" {
		return ErrTransferNoFile
	}

	var partner string

	switch {
	case t.To != "" && t.From != "":
		return ErrTransferBothPartners
	case t.To != "":
		partner = t.To
	case t.From != "":
		partner = t.From
	default:
		return ErrTransferNoPartner
	}

	if err := db.Get(&t.partner, "name=?", partner).Owner().
		Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w: %q", ErrTransferPartnerNotFound, partner)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve partner %q: %w", partner, err)
	}

	if t.As == "" {
		return ErrTransferNoAccount
	}

	if err := db.Get(&t.account, "login=?", t.As).
		And("remote_agent_id=?", t.partner.ID).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w: %q", ErrTransferAccountNotFound, t.As)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve account %q: %w", t.As, err)
	}

	if t.Using == "" {
		//nolint:forcetypeassert //assertion always succeeds here
		client, err := GetDefaultTransferClient(db.(*database.DB), t.partner.Protocol)
		if err != nil {
			return fmt.Errorf("failed to retrieve default transfer client: %w", err)
		}

		t.client = *client
	} else {
		if err := db.Get(&t.client, "name=?", t.Using).Owner().Run(); database.IsNotFound(err) {
			return fmt.Errorf("%w: %q", ErrTransferClientNotFound, t.Using)
		} else if err != nil {
			return fmt.Errorf("failed to retrieve client %q: %w", t.Using, err)
		}
	}

	if t.Rule == "" {
		return ErrTransferNoRule
	}

	if err := db.Get(&t.rule, "name=? AND is_send=?", t.Rule, t.To != "").
		Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w: %q", ErrTransferRuleNotFound, t.Rule)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve rule %q: %w", t.Rule, err)
	}

	return nil
}

// ValidateDB checks if the task has all the required arguments.
func (t *TransferTask) ValidateDB(db database.ReadAccess, args map[string]string) error {
	return t.parseArgs(db, args)
}

// Run executes the task by scheduling a new transfer with the given parameters.
func (t *TransferTask) Run(_ context.Context, args map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := t.parseArgs(db, args); err != nil {
		logger.Error(err.Error())

		return err
	}

	transferInfo := map[string]any{}
	if t.CopyInfo {
		transferInfo = transCtx.TransInfo
	}

	maps.Copy(transferInfo, t.Info)

	output := t.Output
	if output == "" {
		output = filepath.Base(t.File)
	}

	trans := &model.Transfer{
		RuleID:               t.rule.ID,
		ClientID:             utils.NewNullInt64(t.client.ID),
		RemoteAccountID:      utils.NewNullInt64(t.client.ID),
		SrcFilename:          t.File,
		DestFilename:         output,
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

	logger.Debugf("Programmed new transfer n°%d of file %q, %s as %q using rule %q",
		trans.ID, t.File, t.partner.Name, t.account.Login, t.rule.Name)

	return nil
}
