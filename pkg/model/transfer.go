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
	ID               int64                `xorm:"BIGINT PK AUTOINCR <- 'id'"`
	Owner            string               `xorm:"VARCHAR(100) NOTNULL 'owner'"`
	RemoteTransferID string               `xorm:"VARCHAR(100) NOTNULL UNIQUE(remID) UNIQUE(locID) 'remote_transfer_id'"`
	RuleID           int64                `xorm:"BIGINT NOTNULL 'rule_id'"`                 // foreign key (rules.id)
	LocalAccountID   sql.NullInt64        `xorm:"BIGINT UNIQUE(locID) 'local_account_id'"`  // foreign_key (local_accounts.id)
	RemoteAccountID  sql.NullInt64        `xorm:"BIGINT UNIQUE(remID) 'remote_account_id'"` // foreign_key (remote_accounts.id)
	LocalPath        string               `xorm:"TEXT NOTNULL 'local_path'"`
	RemotePath       string               `xorm:"TEXT NOTNULL 'remote_path'"`
	Filesize         int64                `xorm:"BIGINT NOTNULL DEFAULT(-1) 'filesize'"`
	Start            time.Time            `xorm:"DATETIME(6) UTC NOTNULL DEFAULT(CURRENT_TIMESTAMP) 'start'"`
	Status           types.TransferStatus `xorm:"VARCHAR(50) NOTNULL DEFAULT('PLANNED') 'status'"`
	Step             types.TransferStep   `xorm:"VARCHAR(50) NOTNULL DEFAULT('StepNone') 'step'"`
	Progress         int64                `xorm:"BIGINT NOTNULL DEFAULT(0) 'progress'"`
	TaskNumber       int16                `xorm:"SMALLINT NOTNULL DEFAULT(0) 'task_number'"`
	Error            types.TransferError  `xorm:"extends"`
}

// TableName returns the name of the transfers table.
func (*Transfer) TableName() string { return TableTransfers }

// Appellation returns the name of 1 element of the transfers table.
func (*Transfer) Appellation() string { return "transfer" }

// GetID returns the transfer's ID.
func (t *Transfer) GetID() int64 { return t.ID }

// IsServer returns the transfer is a server transfer (from the gateway's perspective).
func (t *Transfer) IsServer() bool { return t.LocalAccountID.Valid }

// SetTransferInfo replaces all the TransferInfo in the database of the given
// transfer by those given in the map parameter.
func (t *Transfer) SetTransferInfo(db *database.DB, info map[string]interface{}) database.Error {
	return setTransferInfo(db, info, t.ID, false)
}

/*
// SetFileInfo replaces all the FileInfo in the database of the given transfer
// by those given in the map parameter.
func (t *Transfer) SetFileInfo(db *database.DB, info map[string]interface{}) database.Error {
	return db.Transaction(func(ses *database.Session) database.Error {
		if err := ses.DeleteAll(&FileInfo{}).Where("transfer_id=?", t.ID).Run(); err != nil {
			return err
		}
		for name, val := range info {
			str, err := json.Marshal(val)
			if err != nil {
				return database.NewValidationError("invalid file info value '%v': %s", val, err)
			}
			i := &FileInfo{TransferID: t.ID, Name: name, Value: string(str)}
			if err := ses.Insert(i).Run(); err != nil {
				return err
			}
		}
		return nil
	})
}
*/

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
func (t *Transfer) GetTransferInfo(db database.ReadAccess) (map[string]interface{}, database.Error) {
	return getTransferInfo(db, t.ID)
}

/*
// GetFileInfo returns the list of the transfer's FileInfo as a map of interfaces.
func (t *Transfer) GetFileInfo(db database.ReadAccess) (map[string]interface{}, database.Error) {
	var infoList FileInfoList
	if err := db.Select(&infoList).Where("transfer_id=?", t.ID).Run(); err != nil {
		return nil, err
	}
	infoMap := map[string]interface{}{}
	for _, info := range infoList {
		var val interface{}
		if err := json.Unmarshal([]byte(info.Value), &val); err != nil {
			return nil, database.NewValidationError("invalid transfer info value '%s': %s", info.Value, err)
		}
		infoMap[info.Name] = val
	}

	return infoMap, nil
}
*/

func (t *Transfer) TransferID() (int64, error) {
	id, err := snowflake.ParseString(t.RemoteTransferID)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the remote transfer ID: %w", err)
	}

	return id.Int64(), nil
}

func (*Transfer) MakeExtraConstraints(db *database.Executor) database.Error {
	// add a not null foreign key to 'rule_id'
	if err := redefineColumn(db, TableTransfers, "rule_id", fmt.Sprintf(
		`BIGINT NOT NULL REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE RESTRICT`,
		TableRules)); err != nil {
		return err
	}

	// add a foreign key to 'local_account_id'
	if err := redefineColumn(db, TableTransfers, "local_account_id", fmt.Sprintf(
		`BIGINT REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE RESTRICT`,
		TableLocAccounts)); err != nil {
		return err
	}

	// add a foreign key to 'remote_account_id'
	if err := redefineColumn(db, TableTransfers, "remote_account_id", fmt.Sprintf(
		`BIGINT REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE RESTRICT`,
		TableRemAccounts)); err != nil {
		return err
	}

	// add a constraint to enforce that one (and only one) of 'local_account_id'
	// and 'remote_account_id' must be defined
	return addTableConstraint(db, TableTransfers, utils.CheckOnlyOneNotNull(
		db.Dialect, "local_account_id", "remote_account_id"))
}
