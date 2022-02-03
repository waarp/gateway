package model

import (
	"fmt"
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// UnknownSize defines the value given to Transfer.Filesize when the file size is
// unknown.
const UnknownSize int64 = -1

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&Transfer{})
}

// Transfer represents one record of the 'transfers' table.
type Transfer struct {
	ID               uint64               `xorm:"pk autoincr <- 'id'"`
	Owner            string               `xorm:"notnull 'owner'"`
	RemoteTransferID string               `xorm:"notnull 'remote_transfer_id'"`
	IsServer         bool                 `xorm:"notnull 'is_server'"`
	RuleID           uint64               `xorm:"notnull 'rule_id'"`
	AgentID          uint64               `xorm:"notnull 'agent_id'"`
	AccountID        uint64               `xorm:"notnull 'account_id'"`
	LocalPath        string               `xorm:"notnull 'local_path'"`
	RemotePath       string               `xorm:"notnull 'remote_path'"`
	Filesize         int64                `xorm:"bigint notnull default(-1) 'filesize'"`
	Start            time.Time            `xorm:"notnull timestampz 'start'"`
	Status           types.TransferStatus `xorm:"notnull 'status'"`
	Step             types.TransferStep   `xorm:"notnull varchar(50) 'step'"`
	Progress         uint64               `xorm:"notnull 'progression'"`
	TaskNumber       uint64               `xorm:"notnull 'task_number'"`
	Error            types.TransferError  `xorm:"extends"`
}

// TableName returns the name of the transfers table.
func (*Transfer) TableName() string {
	return TableTransfers
}

// Appellation returns the name of 1 element of the transfers table.
func (*Transfer) Appellation() string {
	return "transfer"
}

// GetID returns the transfer's ID.
func (t *Transfer) GetID() uint64 {
	return t.ID
}

// SetTransferInfo replaces all the TransferInfo in the database of the given transfer
// by those given in the map parameter.
func (t *Transfer) SetTransferInfo(db database.Access, info map[string]interface{}) error {
	if err := db.DeleteAll(&TransferInfo{TransferID: t.ID}).Run(); err != nil {
		return err
	}

	for name, val := range info {
		i := &TransferInfo{TransferID: t.ID, Name: name, Value: fmt.Sprint(val)}
		if err := db.Insert(i).Run(); err != nil {
			return err
		}
	}

	return nil
}

func (t *Transfer) validateClientTransfer(db database.ReadAccess) database.Error {
	t.RemoteTransferID = ""

	remote := &RemoteAgent{}
	if err := db.Get(remote, "id=?", t.AgentID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationError("the partner %d does not exist", t.AgentID)
		}

		return err
	}

	n, err := db.Count(&RemoteAccount{}).Where("id=? AND remote_agent_id=?",
		t.AccountID, t.AgentID).Run()
	if err != nil {
		return err
	} else if n == 0 {
		return database.NewValidationError("the agent %d does not have an account %d",
			t.AgentID, t.AccountID)
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(db, t); err != nil {
		return err
	} else if !auth {
		return database.NewValidationError("rule %d is not authorized for this transfer",
			t.RuleID)
	}

	return nil
}

func (t *Transfer) validateServerTransfer(db database.ReadAccess) database.Error {
	n, err := db.Count(&LocalAgent{}).Where("id=?", t.AgentID).Run()
	if err != nil {
		return err
	}

	if n == 0 {
		return database.NewValidationError("the server %d does not exist", t.AgentID)
	}

	n, err = db.Count(&LocalAccount{}).Where("id=? AND local_agent_id=?",
		t.AccountID, t.AgentID).Run()
	if err != nil {
		return err
	}

	if n == 0 {
		return database.NewValidationError("the server %d does not have an account %d",
			t.AgentID, t.AccountID)
	}

	if t.RemoteTransferID != "" {
		n1, err := db.Count(t).Where("id<>? AND remote_transfer_id=? AND account_id=?",
			t.ID, t.RemoteTransferID, t.AccountID).Run()
		if err != nil {
			return err
		}

		n2, err := db.Count(&HistoryEntry{}).Where("id<>? AND remote_transfer_id=? AND "+
			"account=(SELECT login FROM local_accounts WHERE id=?)", t.ID,
			t.RemoteTransferID, t.AccountID).Run()
		if err != nil {
			return err
		}

		if n1 != 0 || n2 != 0 {
			return database.NewValidationError("a transfer from the same account " +
				"with the same remote ID already exists")
		}
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(db, t); err != nil {
		return err
	} else if !auth {
		return database.NewValidationError("Rule %d is not authorized for this transfer", t.RuleID)
	}

	return nil
}

func (t *Transfer) checkMandatoryValues() database.Error {
	if t.RuleID == 0 {
		return database.NewValidationError("the transfer's rule ID cannot be empty")
	}

	if t.AgentID == 0 {
		return database.NewValidationError("the transfer's remote ID cannot be empty")
	}

	if t.AccountID == 0 {
		return database.NewValidationError("the transfer's account ID cannot be empty")
	}

	if t.LocalPath == "" || t.LocalPath == "/" || t.LocalPath == "." {
		return database.NewValidationError("the local filepath is missing")
	}

	if t.Start.IsZero() {
		t.Start = time.Now()
	}

	return nil
}

// BeforeWrite checks if the new `Transfer` entry is valid and can be
// inserted in the database.
func (t *Transfer) BeforeWrite(db database.ReadAccess) database.Error {
	t.Owner = database.Owner

	if err := t.checkMandatoryValues(); err != nil {
		return err
	}

	if t.Status == "" {
		t.Status = types.StatusPlanned
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

	if t.LocalPath != "" {
		t.LocalPath = utils.ToOSPath(t.LocalPath)
		if !path.IsAbs(t.LocalPath) && t.LocalPath != path.Base(t.LocalPath) {
			return database.NewValidationError("the local file cannot contain subdirectories")
		}
	} else {
		return database.NewValidationError("the local file cannot be empty")
	}

	if t.RemotePath != "" {
		t.RemotePath = utils.ToStandardPath(t.RemotePath)
		if !path.IsAbs(t.RemotePath) && t.RemotePath != path.Base(t.RemotePath) {
			return database.NewValidationError("the remote file cannot contain subdirectories")
		}
	} else {
		return database.NewValidationError("the remote file cannot be empty")
	}

	n, err := db.Count(&Rule{}).Where("id=?", t.RuleID).Run()
	if err != nil {
		return err
	}

	if n == 0 {
		return database.NewValidationError("the rule %d does not exist", t.RuleID)
	}

	if t.IsServer {
		if err := t.validateServerTransfer(db); err != nil {
			return err
		}
	} else {
		if err := t.validateClientTransfer(db); err != nil {
			return err
		}
	}

	return nil
}

func (t *Transfer) makeAgentInfo(db database.ReadAccess) (string, string, string, database.Error) {
	if t.IsServer {
		var agent LocalAgent
		if err := db.Get(&agent, "id=?", t.AgentID).Run(); err != nil {
			return "", "", "", err
		}

		var account LocalAccount
		if err := db.Get(&account, "id=?", t.AccountID).Run(); err != nil {
			return "", "", "", err
		}

		return agent.Name, account.Login, agent.Protocol, nil
	}

	var agent RemoteAgent
	if err := db.Get(&agent, "id=?", t.AgentID).Run(); err != nil {
		return "", "", "", err
	}

	var account RemoteAccount
	if err := db.Get(&account, "id=?", t.AccountID).Run(); err != nil {
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
		IsServer:         t.IsServer,
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
		if err := ses.Delete(t).Run(); err != nil {
			logger.Errorf("Failed to delete transfer for archival: %s", err)

			return err
		}

		hist, err := t.makeHistoryEntry(ses, end)
		if err != nil {
			logger.Errorf("Failed to convert transfer to history: %s", err)

			return err
		}

		if err := ses.Insert(hist).Run(); err != nil {
			logger.Errorf("Failed to create new history entry: %s", err)

			return err
		}

		return nil
	})
}

// GetTransferInfo returns the list of the transfer's TransferInfo as a map[string]string.
func (t *Transfer) GetTransferInfo(db database.ReadAccess, tID uint64) (map[string]string, error) {
	var infoList TransferInfoList
	if err := db.Select(&infoList).Where("transfer_id=?", tID).Run(); err != nil {
		return nil, err
	}

	infoMap := make(map[string]string, len(infoList))

	for i := range infoList {
		infoMap[infoList[i].Name] = infoList[i].Value
	}

	return infoMap, nil
}
