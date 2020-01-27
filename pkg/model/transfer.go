package model

import (
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &Transfer{})
}

// Transfer represents one record of the 'transfers' table.
type Transfer struct {
	ID         uint64         `xorm:"pk autoincr <- 'id'"`
	RuleID     uint64         `xorm:"notnull 'rule_id'"`
	IsServer   bool           `xorm:"notnull 'is_server'"`
	AgentID    uint64         `xorm:"notnull 'agent_id'"`
	AccountID  uint64         `xorm:"notnull 'account_id'"`
	SourcePath string         `xorm:"notnull 'source_path'"`
	DestPath   string         `xorm:"notnull 'dest_path'"`
	Start      time.Time      `xorm:"notnull 'start'"`
	Status     TransferStatus `xorm:"notnull 'status'"`
	Owner      string         `xorm:"notnull 'owner'"`
	Progress   uint64         `xorm:"notnull 'progression'"`
	TaskNumber uint64         `xorm:"notnull 'task_number'"`
	Error      TransferError  `xorm:"extends"`
	ExtInfo    []byte         `xorm:"'ext_info'"`
}

// TableName returns the name of the transfers table.
func (*Transfer) TableName() string {
	return "transfers"
}

// ValidateInsert checks if the new `Transfer` entry is valid and can be
// inserted in the database.
func (t *Transfer) ValidateInsert(acc database.Accessor) error {
	if t.ID != 0 {
		return database.InvalidError("The transfer's ID cannot be entered manually")
	}
	if t.RuleID == 0 {
		return database.InvalidError("The transfer's rule ID cannot be empty")
	}
	if t.AgentID == 0 {
		return database.InvalidError("The transfer's remote ID cannot be empty")
	}
	if t.AccountID == 0 {
		return database.InvalidError("The transfer's account ID cannot be empty")
	}
	if t.SourcePath == "" {
		return database.InvalidError("The transfer's source cannot be empty")
	}
	if t.DestPath == "" {
		return database.InvalidError("The transfer's destination cannot be empty")
	}
	if t.Start.IsZero() {
		return database.InvalidError("The transfer's starting date cannot be empty")
	}
	if !validateStatusForTransfer(t.Status) {
		return database.InvalidError("'%s' is not a valid transfer status", t.Status)
	}
	if t.Error.Code != TeOk {
		return database.InvalidError("The transfer's error code must be empty")
	}
	if t.Error.Details != "" {
		return database.InvalidError("The transfer's error message must be empty")
	}
	if t.Owner == "" {
		return database.InvalidError("The transfer's owner cannot be empty")
	}

	rule := Rule{ID: t.RuleID}
	if err := acc.Get(&rule); err != nil {
		if err == database.ErrNotFound {
			return database.InvalidError("The rule %d does not exist", t.RuleID)
		}
		return err
	}

	if t.IsServer {
		if err := t.validateServerTransfer(acc); err != nil {
			return err
		}
	} else {
		if err := t.validateClientTransfer(acc); err != nil {
			return err
		}
	}

	return nil
}

func (t *Transfer) validateClientTransfer(acc database.Accessor) error {
	remote := RemoteAgent{ID: t.AgentID}
	if err := acc.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return database.InvalidError("The partner %d does not exist", t.AgentID)
		}
		return err
	}
	if res, err := acc.Query("SELECT id FROM remote_accounts WHERE id=? AND remote_agent_id=?",
		t.AccountID, t.AgentID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("The agent %d does not have an account %d",
			t.AgentID, t.AccountID)
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(acc, t); err != nil {
		return err
	} else if !auth {
		return database.InvalidError("Rule %d is not authorized for this transfer", t.RuleID)
	}

	if remote.Protocol == "sftp" {
		if res, err := acc.Query("SELECT id FROM certificates WHERE owner_type=? AND owner_id=?",
			(&RemoteAgent{}).TableName(), t.AgentID); err != nil {
			return err
		} else if len(res) == 0 {
			return database.InvalidError("No certificate found for agent %d", t.AgentID)
		}
	}
	return nil
}

func (t *Transfer) validateServerTransfer(acc database.Accessor) error {
	remote := LocalAgent{ID: t.AgentID}
	if err := acc.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return database.InvalidError("The partner %d does not exist", t.AgentID)
		}
		return err
	}
	if res, err := acc.Query("SELECT id FROM local_accounts WHERE id=? AND local_agent_id=?",
		t.AccountID, t.AgentID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("The agent %d does not have an account %d",
			t.AgentID, t.AccountID)
	}
	/*
		if remote.Protocol == "sftp" {
		}
	*/
	return nil
}

// BeforeInsert is called before inserting the transfer in the database. Its
// role is to set the Owner, to force the Status and to set a Start time if none
// was entered.
func (t *Transfer) BeforeInsert(database.Accessor) error {
	t.Owner = database.Owner
	if t.Start.IsZero() {
		t.Start = time.Now().Truncate(time.Second)
	}
	if t.Status == "" {
		t.Status = StatusPlanned
	}
	return nil
}

// ValidateUpdate is called before updating an existing `Transfer` entry from
// the database. It checks whether the updated entry is valid or not.
func (t *Transfer) ValidateUpdate(database.Accessor, uint64) error {
	if t.ID != 0 {
		return database.InvalidError("The transfer's ID cannot be entered manually")
	}
	if t.Owner != "" {
		return database.InvalidError("The transfer's owner cannot be changed")
	}
	if t.RuleID != 0 {
		return database.InvalidError("The transfer's rule cannot be changed")
	}
	if t.AgentID != 0 {
		return database.InvalidError("The transfer's partner cannot be changed")
	}
	if t.AccountID != 0 {
		return database.InvalidError("The transfer's account cannot be changed")
	}
	if t.SourcePath != "" {
		return database.InvalidError("The transfer's source cannot be changed")
	}
	if t.DestPath != "" {
		return database.InvalidError("The transfer's destination cannot be changed")
	}

	if t.Status != "" {
		if !validateStatusForTransfer(t.Status) {
			return database.InvalidError("'%s' is not a valid transfer status", t.Status)
		}
	}

	return nil
}

// ToHistory converts the `Transfer` entry into an equivalent `TransferHistory`
// entry with the given time as the end date.
func (t *Transfer) ToHistory(acc database.Accessor, stop time.Time) (*TransferHistory, error) {

	rule := &Rule{ID: t.RuleID}
	if err := acc.Get(rule); err != nil {
		return nil, fmt.Errorf("rule: %s", err)
	}
	agentName := ""
	accountLogin := ""
	protocol := ""

	if t.IsServer {
		agent := &LocalAgent{ID: t.AgentID}
		if err := acc.Get(agent); err != nil {
			return nil, fmt.Errorf("local agent: %s", err)
		}
		account := &LocalAccount{ID: t.AccountID}
		if err := acc.Get(account); err != nil {
			return nil, fmt.Errorf("local account: %s", err)
		}
		agentName = agent.Name
		accountLogin = account.Login
		protocol = agent.Protocol
	} else {
		agent := &RemoteAgent{ID: t.AgentID}
		if err := acc.Get(agent); err != nil {
			return nil, fmt.Errorf("remote agent: %s", err)
		}
		account := &RemoteAccount{ID: t.AccountID}
		if err := acc.Get(account); err != nil {
			return nil, fmt.Errorf("remote account: %s", err)
		}
		agentName = agent.Name
		accountLogin = account.Login
		protocol = agent.Protocol
	}

	if !validateStatusForHistory(t.Status) {
		return nil, fmt.Errorf(
			"a transfer cannot be recorded in history with status '%s'", t.Status,
		)
	}

	hist := TransferHistory{
		ID:             t.ID,
		Owner:          t.Owner,
		IsServer:       t.IsServer,
		IsSend:         rule.IsSend,
		Account:        accountLogin,
		Agent:          agentName,
		Protocol:       protocol,
		SourceFilename: t.SourcePath,
		DestFilename:   t.DestPath,
		Rule:           rule.Name,
		Start:          t.Start,
		Stop:           stop,
		Status:         t.Status,
		Error:          t.Error,
		ExtInfo:        t.ExtInfo,
	}

	return &hist, nil
}

// Update updates the transfer start, status & error in the database.
func (t *Transfer) Update(acc database.Accessor) error {
	trans := &Transfer{
		Start:      t.Start,
		Status:     t.Status,
		Error:      t.Error,
		Progress:   t.Progress,
		TaskNumber: t.TaskNumber,
	}

	return acc.Update(trans, t.ID, false)
}
