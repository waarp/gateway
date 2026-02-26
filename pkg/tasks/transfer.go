package tasks

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const transferCancelTimeout = 5 * time.Second

type ClientPipeline interface {
	Run() error
	Interrupt(context.Context) error
}

//nolint:gochecknoglobals //yes, this is very ugly, but there is really no other way to avoid an import cycle
var (
	GetDefaultTransferClient func(db database.Access, protocol string) (*model.Client, error)
	NewClientPipeline        func(db *database.DB, trans *model.Transfer) (ClientPipeline, error)
)

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

	errTransferNeedDBWrite = errors.New("transfer task requires database write access")
)

// TransferTask is a task which schedules a new transfer.
type TransferTask struct {
	Synchronous          jsonBool     `json:"synchronous"`
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
	Timeout              jsonDuration `json:"timeout"`

	partner model.RemoteAgent
	account model.RemoteAccount
	rule    model.Rule
	client  model.Client
}

func (t *TransferTask) parseArgs(db database.Access, args map[string]string) error {
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
		client, err := GetDefaultTransferClient(db, t.partner.Protocol)
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
func (t *TransferTask) ValidateDB(rd database.ReadAccess, args map[string]string) error {
	rw, ok := rd.(database.Access)
	if !ok {
		return errTransferNeedDBWrite
	}

	return t.parseArgs(rw, args)
}

// Run executes the task by scheduling a new transfer with the given parameters.
func (t *TransferTask) Run(ctx context.Context, args map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := t.parseArgs(db, args); err != nil {
		logger.Error(err.Error())

		return err
	}

	trans, err := t.makeTransfer(db, transCtx)
	if err != nil {
		logger.Errorf("Failed to create transfer: %v", err)

		return err
	}

	if !t.Synchronous {
		logger.Debugf("Programmed new transfer n°%d of file %q, %s as %q using rule %q",
			trans.ID, t.File, t.partner.Name, t.account.Login, t.rule.Name)

		return nil
	}

	return t.runSynchronousTransfer(ctx, db, logger, trans)
}

func (t *TransferTask) makeTransfer(db *database.DB, transCtx *model.TransferContext,
) (*model.Transfer, error) {
	if t.Synchronous {
		trans, dbErr := model.GetTransferFromParentID(db, transCtx.Transfer)
		if dbErr == nil {
			trans.Status = types.StatusRunning
			trans.ErrCode = types.TeOk
			trans.ErrDetails = ""
			trans.NextRetry = time.Now()

			if err := db.Update(trans).Run(); err != nil {
				return nil, fmt.Errorf("failed to update transfer: %w", err)
			}

			return trans, nil
		} else if !database.IsNotFound(dbErr) {
			return nil, fmt.Errorf("failed to retrieve transfer: %w", dbErr)
		}
	}

	transferInfo := map[string]any{}
	if t.CopyInfo {
		transferInfo = transCtx.Transfer.CopyInfo()
	}

	maps.Copy(transferInfo, t.Info)

	output := t.Output
	if output == "" {
		output = filepath.Base(t.File)
	}

	trans := &model.Transfer{
		RuleID:               t.rule.ID,
		ClientID:             utils.NewNullInt64(t.client.ID),
		RemoteAccountID:      utils.NewNullInt64(t.account.ID),
		SrcFilename:          t.File,
		DestFilename:         output,
		RemainingTries:       int8(t.NbOfAttempts),
		NextRetryDelay:       int32(t.FirstRetryDelay.Seconds()),
		RetryIncrementFactor: float32(t.RetryIncrementFactor),
		TransferInfo:         transferInfo,
	}

	if t.Synchronous {
		trans.Status = types.StatusRunning
		transferInfo[model.SyncTransferID] = transCtx.Transfer.ID
		transferInfo[model.SyncTransferRank] = transCtx.Transfer.TaskNumber
	}

	if err := db.Insert(trans).Run(); err != nil {
		return nil, fmt.Errorf("failed to insert transfer: %w", err)
	}

	return trans, nil
}

func (t *TransferTask) runSynchronousTransfer(ctx context.Context,
	db *database.DB, logger *log.Logger, trans *model.Transfer,
) error {
	logger.Debugf("Executing new transfer n°%d of file %q, %s as %q using rule %q",
		trans.ID, t.File, t.partner.Name, t.account.Login, t.rule.Name)

	pip, pipErr := NewClientPipeline(db, trans)
	if pipErr != nil {
		return fmt.Errorf("failed to initialize the client transfer pipeline: %w", pipErr)
	}

	if !t.Timeout.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.Timeout.Duration)
		defer cancel()
	}

	result := make(chan error)
	go func() {
		defer close(result)
		result <- pip.Run()
	}()

	select {
	case err := <-result:
		if err != nil {
			return fmt.Errorf("failed to run the client transfer pipeline: %w", err)
		}

		return nil
	case <-ctx.Done():
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, transferCancelTimeout)
		defer cancel()

		if err := pip.Interrupt(ctx); err != nil {
			logger.Warningf("Failed to cancel transfer: %v", err)
		}

		return fmt.Errorf("transfer cancelled: %w", ctx.Err())
	}
}
