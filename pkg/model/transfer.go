package model

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/bwmarrin/snowflake"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
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
//
//nolint:lll //SQL tags are long, nothing we can do about it
type Transfer struct {
	ID               int64                `xorm:"<- id AUTOINCR"`
	Owner            string               `xorm:"owner"`
	RemoteTransferID string               `xorm:"remote_transfer_id"`
	RuleID           int64                `xorm:"rule_id"`
	LocalAccountID   sql.NullInt64        `xorm:"local_account_id"`
	RemoteAccountID  sql.NullInt64        `xorm:"remote_account_id"`
	LocalPath        string               `xorm:"local_path"`
	RemotePath       string               `xorm:"remote_path"`
	Filesize         int64                `xorm:"filesize"`
	Start            time.Time            `xorm:"start DATETIME(6) UTC"`
	Status           types.TransferStatus `xorm:"status"`
	Step             types.TransferStep   `xorm:"step"`
	Progress         int64                `xorm:"progress"`
	TaskNumber       int8                 `xorm:"task_number"`
	Error            types.TransferError  `xorm:"EXTENDS"`
}

func (*Transfer) TableName() string   { return TableTransfers }
func (*Transfer) Appellation() string { return "transfer" }
func (t *Transfer) GetID() int64      { return t.ID }

// IsServer returns the transfer is a server transfer (from the gateway's perspective).
func (t *Transfer) IsServer() bool { return t.LocalAccountID.Valid }

func (t *Transfer) getTransInfoCondition() (string, int64) {
	return "transfer_id=?", t.ID
}

func (t *Transfer) setTransInfoOwner(info *TransferInfo) {
	info.TransferID = utils.NewNullInt64(t.ID)
}

func (t *Transfer) checkRemoteTransferID(db database.ReadAccess) database.Error {
	accCond, accArgs := func() (string, []any) {
		if t.IsServer() {
			return "local_account_id=?", []any{t.LocalAccountID.Int64}
		} else {
			return "remote_account_id=?", []any{t.RemoteAccountID.Int64}
		}
	}()

	n1, err := db.Count(t).Where("id<>? AND remote_transfer_id=?", t.ID,
		t.RemoteTransferID).Where(accCond, accArgs...).Run()
	if err != nil {
		return err
	}

	tbl := "local_accounts"
	accID := t.LocalAccountID.Int64

	if !t.IsServer() {
		tbl = "remote_accounts"
		accID = t.RemoteAccountID.Int64
	}

	n2, err := db.Count(&HistoryEntry{}).Where(fmt.Sprintf("remote_transfer_id=? "+
		"AND is_server=? AND account=(SELECT login FROM %s WHERE id=?)", tbl),
		t.RemoteTransferID, t.IsServer(), accID).Run()
	if err != nil {
		return err
	}

	if n1 != 0 || n2 != 0 {
		return database.NewValidationError("a transfer from the same account " +
			"with the same remote ID already exists")
	}

	return nil
}

func (t *Transfer) checkMandatoryValues() database.Error {
	if t.RuleID == 0 {
		return database.NewValidationError("the transfer's rule ID cannot be empty")
	}

	if t.LocalPath == "" || t.LocalPath == "/" || t.LocalPath == "." {
		return database.NewValidationError("the local filepath is missing")
	}

	if t.RemotePath == "" || t.RemotePath == "/" || t.RemotePath == "." {
		t.RemotePath = filepath.Base(t.LocalPath)
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
		return database.NewValidationError("'%s' is not a valid transfer status", t.Status)
	}

	if !t.Step.IsValid() {
		return database.NewValidationError("'%s' is not a valid transfer step", t.Step)
	}

	if !t.Error.Code.IsValid() {
		return database.NewValidationError("'%s' is not a valid transfer error code", t.Error.Code)
	}

	t.LocalPath = utils.ToOSPath(t.LocalPath)
	t.RemotePath = utils.ToStandardPath(t.RemotePath)

	return nil
}

// BeforeWrite checks if the new `Transfer` entry is valid and can be
// inserted in the database.
func (t *Transfer) BeforeWrite(db database.ReadAccess) database.Error {
	t.Owner = conf.GlobalConfig.GatewayName

	if err := t.checkMandatoryValues(); err != nil {
		return err
	}

	switch {
	case !t.LocalAccountID.Valid && !t.RemoteAccountID.Valid:
		return database.NewValidationError("the transfer is missing an account ID")
	case !t.LocalAccountID.Valid:
		if err := db.Get(&RemoteAccount{}, "id=?", t.RemoteAccountID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationError("the remote account %d does not exist",
					t.RemoteAccountID.Int64)
			}

			return err
		}
	case !t.RemoteAccountID.Valid:
		if err := db.Get(&LocalAccount{}, "id=?", t.LocalAccountID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationError("the local account %d does not exist",
					t.LocalAccountID.Int64)
			}

			return err
		}
	default:
		return database.NewValidationError("the transfer cannot have both a local and remote account ID")
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(db, t); err != nil {
		return err
	} else if !auth {
		return database.NewValidationError("Rule %d is not authorized for this transfer", t.RuleID)
	}

	if n, err := db.Count(&Rule{}).Where("id=?", t.RuleID).Run(); err != nil {
		return err
	} else if n == 0 {
		return database.NewValidationError("the rule %d does not exist", t.RuleID)
	}

	return t.checkRemoteTransferID(db)
}

func (t *Transfer) makeAgentInfo(db database.ReadAccess) (string, string, string, database.Error) {
	if t.IsServer() {
		var account LocalAccount
		if err := db.Get(&account, "id=?", t.LocalAccountID).Run(); err != nil {
			return "", "", "", err
		}

		var agent LocalAgent
		if err := db.Get(&agent, "id=?", account.LocalAgentID).Run(); err != nil {
			return "", "", "", err
		}

		return agent.Name, account.Login, agent.Protocol, nil
	}

	var account RemoteAccount
	if err := db.Get(&account, "id=?", t.RemoteAccountID).Run(); err != nil {
		return "", "", "", err
	}

	var agent RemoteAgent
	if err := db.Get(&agent, "id=?", account.RemoteAgentID).Run(); err != nil {
		return "", "", "", err
	}

	return agent.Name, account.Login, agent.Protocol, nil
}

// makeHistoryEntry converts the `Transfer` entry into an equivalent `HistoryEntry`
// entry with the given time as the end date.
func (t *Transfer) makeHistoryEntry(db database.ReadAccess, stop time.Time) (*HistoryEntry, database.Error) {
	var rule Rule
	if err := db.Get(&rule, "id=?", t.RuleID).Run(); err != nil {
		return nil, err
	}

	agentName, accountLogin, protocol, err2 := t.makeAgentInfo(db)
	if err2 != nil {
		return nil, err2
	}

	if !types.ValidateStatusForHistory(t.Status) {
		return nil, database.NewValidationError(
			"a transfer cannot be recorded in history with status '%s'", t.Status)
	}

	hist := HistoryEntry{
		ID:               t.ID,
		Owner:            t.Owner,
		RemoteTransferID: t.RemoteTransferID,
		IsServer:         t.IsServer(),
		IsSend:           rule.IsSend,
		Account:          accountLogin,
		Agent:            agentName,
		Protocol:         protocol,
		LocalPath:        t.LocalPath,
		RemotePath:       t.RemotePath,
		Filesize:         t.Filesize,
		Rule:             rule.Name,
		Start:            t.Start,
		Stop:             stop,
		Status:           t.Status,
		Error:            t.Error,
		Step:             t.Step,
		Progress:         t.Progress,
		TaskNumber:       t.TaskNumber,
	}

	return &hist, nil
}

// ToHistory removes the transfer entry from the database, converts it into a
// history entry, and inserts the new history entry in the database.
// If any of these steps fails, the changes are reverted and an error is returned.
func (t *Transfer) ToHistory(db *database.DB, logger *log.Logger, end time.Time) database.Error {
	return db.Transaction(func(ses *database.Session) database.Error {
		hist, err := t.makeHistoryEntry(ses, end)
		if err != nil {
			logger.Error("Failed to convert transfer to history: %v", err)

			return err
		}

		if err := ses.Insert(hist).Run(); err != nil {
			logger.Error("Failed to create new history entry: %v", err)

			return err
		}

		if err := ses.Exec(`UPDATE transfer_info SET history_id=transfer_id, 
			transfer_id=null WHERE transfer_id=?`, t.ID); err != nil {
			logger.Error("Failed to update transfer info target: %s", err)

			return err
		}

		/*
			if err := ses.Exec(`UPDATE file_info SET history_id=transfer_id, transfer_id=null`); err != nil {
				logger.Errorf("Failed to update file info target: %s", err)

				return err
			}
		*/

		if err := ses.Delete(t).Run(); err != nil {
			logger.Error("Failed to delete transfer for archival: %s", err)

			return err
		}

		return nil
	})
}

// GetTransferInfo returns the list of the transfer's TransferInfo as a map of interfaces.
func (t *Transfer) GetTransferInfo(db database.ReadAccess) (map[string]any, database.Error) {
	return getTransferInfo(db, t)
}

// SetTransferInfo replaces all the TransferInfo in the database of the given
// history entry by those given in the map parameter.
func (t *Transfer) SetTransferInfo(db database.Access, info map[string]any) database.Error {
	return setTransferInfo(db, t, info)
}

func (t *Transfer) TransferID() (int64, error) {
	id, err := snowflake.ParseString(t.RemoteTransferID)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the remote transfer ID: %w", err)
	}

	return id.Int64(), nil
}
