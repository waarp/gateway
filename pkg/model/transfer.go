package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/bwmarrin/snowflake"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// UnknownSize defines the value given to Transfer.Filesize when the file size is
// unknown.
const UnknownSize int64 = -1

//nolint:gochecknoglobals //global var is required here
var idGenerator *snowflake.Node

//nolint:gochecknoinits // init is used by design
func init() {
	var err error
	if idGenerator, err = makeIDGenerator(); err != nil {
		panic(err)
	}
}

// Transfer represents one record of the 'transfers' table.
type Transfer struct {
	ID                   int64                   `xorm:"<- id AUTOINCR"`
	Owner                string                  `xorm:"owner"`
	RemoteTransferID     string                  `xorm:"remote_transfer_id"`
	RuleID               int64                   `xorm:"rule_id"`
	ClientID             sql.NullInt64           `xorm:"client_id"`
	LocalAccountID       sql.NullInt64           `xorm:"local_account_id"`
	RemoteAccountID      sql.NullInt64           `xorm:"remote_account_id"`
	SrcFilename          string                  `xorm:"src_filename"`
	DestFilename         string                  `xorm:"dest_filename"`
	LocalPath            string                  `xorm:"local_path"`
	RemotePath           string                  `xorm:"remote_path"`
	Filesize             int64                   `xorm:"filesize"`
	Start                time.Time               `xorm:"start DATETIME(6) UTC"`
	Stop                 time.Time               `xorm:"stop DATETIME(6) UTC"`
	Status               types.TransferStatus    `xorm:"status"`
	Step                 types.TransferStep      `xorm:"step"`
	Progress             int64                   `xorm:"progress"`
	TaskNumber           int8                    `xorm:"task_number"`
	ErrCode              types.TransferErrorCode `xorm:"error_code"`
	ErrDetails           string                  `xorm:"error_details"`
	RemainingTries       int8                    `xorm:"remaining_tries"`
	NextRetryDelay       int32                   `xorm:"next_retry_delay"`
	RetryIncrementFactor float32                 `xorm:"retry_increment_factor"`
	NextRetry            time.Time               `xorm:"next_retry DATETIME(6) UTC"`
	TransferInfo         map[string]any          `xorm:"-"`
}

func (*Transfer) TableName() string   { return TableTransfers }
func (*Transfer) Appellation() string { return NameTransfer }
func (t *Transfer) GetID() int64      { return t.ID }
func (t *Transfer) IsServer() bool    { return t.LocalAccountID.Valid }

func (t *Transfer) getTransInfoCondition() (string, int64) {
	return "transfer_id=?", t.ID
}

func (t *Transfer) setTransInfoOwner(info *TransferInfo) {
	info.TransferID = utils.NewNullInt64(t.ID)
}

func (t *Transfer) Init(db database.Access) error {
	return initPesitCounter(db)
}

//nolint:funlen,gocyclo,cyclop //function is fine for now
func (t *Transfer) checkMandatoryValues(rule *Rule) error {
	if t.IsServer() {
		if rule.IsSend {
			if t.SrcFilename == "" {
				return database.NewValidationError("the source file is missing")
			}

			// For server transfers, we force the filepath to be relative for
			// security reasons.
			t.SrcFilename = path.Clean("./" + t.SrcFilename)
			t.RemotePath, t.DestFilename = "", ""
		} else {
			if t.DestFilename == "" {
				return database.NewValidationError("the destination file is missing")
			}

			// For server transfers, we force the filepath to be relative for
			// security reasons.
			t.DestFilename = path.Clean("./" + t.DestFilename)
			t.RemotePath, t.SrcFilename = "", ""
		}
	} else if t.SrcFilename == "" {
		return database.NewValidationError("the source file is missing")
	}

	if t.RemotePath != "" && t.LocalPath == "" {
		return database.NewValidationError("the local path is missing")
	}

	if t.Start.IsZero() {
		t.Start = time.Now()
	}

	if t.Status == "" {
		t.Status = types.StatusPlanned
	}

	if strings.TrimSpace(t.RemoteTransferID) == "" {
		remoteID := idGenerator.Generate()
		t.RemoteTransferID = remoteID.String()
	}

	if !types.ValidateStatusForTransfer(t.Status) {
		return database.NewValidationErrorf("%q is not a valid transfer status", t.Status)
	}

	if !t.Step.IsValid() {
		return database.NewValidationErrorf("%q is not a valid transfer step", t.Step)
	}

	if !t.ErrCode.IsValid() {
		return database.NewValidationErrorf("%q is not a valid transfer error code", t.ErrCode)
	}

	if t.LocalPath != "" {
		if err := fs.ValidPath(t.LocalPath); err != nil {
			return database.NewValidationErrorf("invalid local path: %v", err)
		}
	}

	t.RemotePath = filepath.ToSlash(t.RemotePath)

	return nil
}

// BeforeWrite checks if the new `Transfer` entry is valid and can be
// inserted in the database.
//
//nolint:funlen,cyclop //no easy way to split the function
func (t *Transfer) BeforeWrite(db database.Access) error {
	t.Owner = conf.GlobalConfig.GatewayName

	if t.RuleID == 0 {
		return database.NewValidationError("the transfer's rule ID cannot be empty")
	}

	var rule Rule
	if err := db.Get(&rule, "id=?", t.RuleID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the rule %d does not exist", t.RuleID)
		}

		return fmt.Errorf("failed to retrieve rule: %w", err)
	}

	if err := t.checkMandatoryValues(&rule); err != nil {
		return err
	}

	switch {
	case t.LocalAccountID.Valid && t.RemoteAccountID.Valid:
		return database.NewValidationError("the transfer cannot have both a local and remote account ID")
	case t.RemoteAccountID.Valid:
		if err := db.Get(&RemoteAccount{}, "id=?", t.RemoteAccountID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf("the remote account %d does not exist",
					t.RemoteAccountID.Int64)
			}

			return fmt.Errorf("failed to retrieve remote account: %w", err)
		}

		if !t.ClientID.Valid {
			return database.NewValidationError("the transfer is missing a client ID")
		}

		var client Client
		if err := db.Get(&client, "id=?", t.ClientID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf("the client %d does not exist",
					t.LocalAccountID.Int64)
			}

			return fmt.Errorf("failed to retrieve client: %w", err)
		}

		t.setRetryParameters(&client)
	case t.LocalAccountID.Valid:
		if err := db.Get(&LocalAccount{}, "id=?", t.LocalAccountID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf("the local account %d does not exist",
					t.LocalAccountID.Int64)
			}

			return fmt.Errorf("failed to retrieve local account: %w", err)
		}

		t.clearRetryParameters()
	default:
		return database.NewValidationError("the transfer is missing an account ID")
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(db, t); err != nil {
		return err
	} else if !auth {
		return database.NewValidationErrorf("rule %d is not authorized for this transfer", t.RuleID)
	}

	return nil
}

func (t *Transfer) UpdateInfo(db database.Access) error {
	return setTransferInfo(db, t, t.TransferInfo)
}

func (t *Transfer) AfterInsert(db database.Access) error {
	if t.TransferInfo == nil {
		t.TransferInfo = map[string]any{}
	}

	if err := mkPesitID(db, t); err != nil {
		return err
	}

	if t.TransferInfo[FollowID] == nil {
		t.TransferInfo[FollowID] = json.Number(t.RemoteTransferID)
	}

	return t.UpdateInfo(db)
}

func (t *Transfer) AfterUpdate(db database.Access) error {
	return t.UpdateInfo(db)
}

func (t *Transfer) AfterRead(db database.ReadAccess) error {
	infos, err := getTransferInfo(db, t)
	if err != nil {
		return err
	}

	t.TransferInfo = infos

	return nil
}

func (t *Transfer) makeAgentInfo(db database.ReadAccess,
) (proto, client, agent, account string, err error) {
	if t.IsServer() {
		var locAcc LocalAccount
		if err = db.Get(&locAcc, "id=?", t.LocalAccountID.Int64).Run(); err != nil {
			return "", "", "", "", fmt.Errorf("failed to retrieve local account: %w", err)
		}

		var locAg LocalAgent
		if err = db.Get(&locAg, "id=?", locAcc.LocalAgentID).Run(); err != nil {
			return "", "", "", "", fmt.Errorf("failed to retrieve local agent: %w", err)
		}

		proto = locAg.Protocol
		agent = locAg.Name
		account = locAcc.Login

		return proto, "", agent, account, nil
	}

	var remAcc RemoteAccount
	if err = db.Get(&remAcc, "id=?", t.RemoteAccountID.Int64).Run(); err != nil {
		return "", "", "", "", fmt.Errorf("failed to retrieve remote account: %w", err)
	}

	var remAg RemoteAgent
	if err = db.Get(&remAg, "id=?", remAcc.RemoteAgentID).Run(); err != nil {
		return "", "", "", "", fmt.Errorf("failed to retrieve remote agent: %w", err)
	}

	var cli Client
	if err = db.Get(&cli, "id=?", t.ClientID).Run(); err != nil {
		return "", "", "", "", fmt.Errorf("failed to retrieve client: %w", err)
	}

	proto = cli.Protocol
	client = cli.Name
	agent = remAg.Name
	account = remAcc.Login

	return proto, client, agent, account, nil
}

// makeHistoryEntry converts the `Transfer` entry into an equivalent `HistoryEntry`
// entry with the given time as the end date.
func (t *Transfer) makeHistoryEntry(db database.ReadAccess, stop time.Time) (*HistoryEntry, error) {
	var rule Rule
	if err := db.Get(&rule, "id=?", t.RuleID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve rule name: %w", err)
	}

	protocol, clientName, agentName, accountLogin, err := t.makeAgentInfo(db)
	if err != nil {
		return nil, err
	}

	if !types.ValidateStatusForHistory(t.Status) {
		return nil, database.NewValidationErrorf(
			"a transfer cannot be recorded in history with status %q", t.Status)
	}

	hist := HistoryEntry{
		ID:               t.ID,
		Owner:            t.Owner,
		RemoteTransferID: t.RemoteTransferID,
		IsServer:         t.IsServer(),
		IsSend:           rule.IsSend,
		Account:          accountLogin,
		Agent:            agentName,
		Client:           clientName,
		Protocol:         protocol,
		SrcFilename:      t.SrcFilename,
		DestFilename:     t.DestFilename,
		LocalPath:        t.LocalPath,
		RemotePath:       t.RemotePath,
		Filesize:         t.Filesize,
		Rule:             rule.Name,
		Start:            t.Start,
		Stop:             stop,
		Status:           t.Status,
		ErrCode:          t.ErrCode,
		ErrDetails:       t.ErrDetails,
		Step:             t.Step,
		Progress:         t.Progress,
		TaskNumber:       t.TaskNumber,
	}

	return &hist, nil
}

func (t *Transfer) CopyToHistory(db database.Access, logger *log.Logger, end time.Time) error {
	hist, hErr := t.makeHistoryEntry(db, end)
	if hErr != nil {
		logger.Errorf("Failed to convert transfer to history: %v", hErr)

		return hErr
	}

	if err := db.Insert(hist).Run(); err != nil {
		logger.Errorf("Failed to create new history entry: %v", err)

		return fmt.Errorf("failed to create new history entry: %w", err)
	}

	if err := db.Exec(`UPDATE transfer_info SET history_id=transfer_id, 
			transfer_id=null WHERE transfer_id=?`, t.ID); err != nil {
		logger.Errorf("Failed to update transfer info target: %v", err)

		return fmt.Errorf("failed to update transfer info target: %w", err)
	}

	/*
		if err := ses.Exec(`UPDATE file_info SET history_id=transfer_id, transfer_id=null`); err != nil {
			logger.Errorf("Failed to update file info target: %v", err)

			return err
		}
	*/

	return nil
}

// MoveToHistory removes the transfer entry from the database, converts it into a
// history entry, and inserts the new history entry in the database.
// If any of these steps fails, the changes are reverted and an error is returned.
func (t *Transfer) MoveToHistory(db *database.DB, logger *log.Logger, end time.Time) error {
	if err := db.Transaction(func(ses *database.Session) error {
		if err := t.CopyToHistory(ses, logger, end); err != nil {
			return err
		}

		if err := ses.Delete(t).Run(); err != nil {
			logger.Errorf("Failed to delete transfer for archival: %v", err)

			return fmt.Errorf("failed to delete transfer for archival: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to move transfer to history: %w", err)
	}

	return nil
}

func (t *Transfer) TransferID() (int64, error) {
	id, err := snowflake.ParseString(t.RemoteTransferID)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the remote transfer ID: %w", err)
	}

	return id.Int64(), nil
}

func (t *Transfer) setRetryParameters(client *Client) {
	if t.ID == 0 {
		if t.RemainingTries == 0 {
			t.RemainingTries = client.NbOfAttempts
		}

		if t.NextRetryDelay == 0 {
			t.NextRetryDelay = client.FirstRetryDelay
		}

		if t.RetryIncrementFactor == 0 {
			t.RetryIncrementFactor = client.RetryIncrementFactor
		}
	}

	if t.RemainingTries != 0 && t.NextRetryDelay == 0 {
		t.NextRetryDelay = 1
	}

	if t.NextRetryDelay != 0 && t.RetryIncrementFactor == 0 {
		t.RetryIncrementFactor = 1.0
	}

	if t.Status == types.StatusPlanned && t.NextRetry.IsZero() {
		t.NextRetry = t.Start
	}
}

func (t *Transfer) clearRetryParameters() {
	t.RemainingTries = 0
	t.NextRetryDelay = 0
	t.RetryIncrementFactor = 0
}

func GetTransferFromParentID(db database.ReadAccess, parent *Transfer) (*Transfer, error) {
	id := utils.FormatInt(parent.ID)
	rank := utils.FormatInt(parent.TaskNumber)

	var transfer Transfer
	if err := db.Get(&transfer,
		"id=(SELECT transfer_id FROM transfer_info WHERE name=? AND value=? AND "+
			"transfer_id IN (SELECT transfer_id FROM transfer_info WHERE name=? AND value=?))",
		SyncTransferID, id, SyncTransferRank, rank).
		Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve transfer: %w", err)
	}

	return &transfer, nil
}

func (t *Transfer) CheckResumable() error {
	if t.IsServer() {
		return ErrResumeServer
	}

	switch t.Status {
	case types.StatusPaused, types.StatusInterrupted, types.StatusError:
	default:
		return ErrResumeRunning
	}

	if parentID := t.TransferInfo[SyncTransferID]; parentID != nil {
		return &ResumeSyncError{ID: t.ID, ParentID: parentID}
	}

	return nil
}

func (t *Transfer) Resume(db database.Access, when time.Time) error {
	if err := t.CheckResumable(); err != nil {
		return err
	}

	t.Status = types.StatusPlanned
	t.ErrCode = types.TeOk
	t.ErrDetails = ""
	t.NextRetry = when
	t.Stop = time.Time{}

	if err := db.Update(t).Run(); err != nil {
		return fmt.Errorf("failed to update transfer: %w", err)
	}

	return nil
}

func (t *Transfer) CopyInfo() map[string]any {
	clone := maps.Clone(t.TransferInfo)
	delete(clone, SyncTransferID)
	delete(clone, SyncTransferRank)

	return clone
}
