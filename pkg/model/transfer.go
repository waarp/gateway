package model

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &Transfer{})
}

// Transfer represents one record of the 'transfers' table.
type Transfer struct {
	ID           uint64         `xorm:"pk autoincr <- 'id'"`
	RuleID       uint64         `xorm:"notnull 'rule_id'"`
	IsServer     bool           `xorm:"notnull 'is_server'"`
	AgentID      uint64         `xorm:"notnull 'agent_id'"`
	AccountID    uint64         `xorm:"notnull 'account_id'"`
	TrueFilepath string         `xorm:"notnull 'true_filepath'"`
	SourceFile   string         `xorm:"notnull 'source_file'"`
	DestFile     string         `xorm:"notnull 'dest_file'"`
	Start        time.Time      `xorm:"notnull 'start'"`
	Step         TransferStep   `xorm:"notnull 'step'"`
	Status       TransferStatus `xorm:"notnull 'status'"`
	Owner        string         `xorm:"notnull 'owner'"`
	Progress     uint64         `xorm:"notnull 'progression'"`
	TaskNumber   uint64         `xorm:"notnull 'task_number'"`
	Error        TransferError  `xorm:"extends"`
	ExtInfo      []byte         `xorm:"'ext_info'"`
}

// TableName returns the name of the transfers table.
func (*Transfer) TableName() string {
	return "transfers"
}

func (t *Transfer) Id() uint64 {
	return t.ID
}

func (t *Transfer) validateClientTransfer(db database.Accessor) error {
	remote := RemoteAgent{ID: t.AgentID}
	if err := db.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return database.InvalidError("the partner %d does not exist", t.AgentID)
		}
		return err
	}
	if res, err := db.Query("SELECT id FROM remote_accounts WHERE id=? AND remote_agent_id=?",
		t.AccountID, t.AgentID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("the agent %d does not have an account %d",
			t.AgentID, t.AccountID)
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(db, t); err != nil {
		return err
	} else if !auth {
		return database.InvalidError("Rule %d is not authorized for this transfer", t.RuleID)
	}

	if remote.Protocol == "sftp" {
		if res, err := db.Query("SELECT id FROM certificates WHERE owner_type=? AND owner_id=?",
			(&RemoteAgent{}).TableName(), t.AgentID); err != nil {
			return err
		} else if len(res) == 0 {
			return database.InvalidError("the partner is missing an SFTP host key")
		}
	}
	return nil
}

func (t *Transfer) validateServerTransfer(db database.Accessor) error {
	remote := LocalAgent{ID: t.AgentID}
	if err := db.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return database.InvalidError("the partner %d does not exist", t.AgentID)
		}
		return err
	}
	if res, err := db.Query("SELECT id FROM local_accounts WHERE id=? AND local_agent_id=?",
		t.AccountID, t.AgentID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("the agent %d does not have an account %d",
			t.AgentID, t.AccountID)
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(db, t); err != nil {
		return err
	} else if !auth {
		return database.InvalidError("Rule %d is not authorized for this transfer", t.RuleID)
	}
	return nil
}

// Validate checks if the new `Transfer` entry is valid and can be
// inserted in the database.
//nolint:funlen
func (t *Transfer) Validate(db database.Accessor) error {
	t.Owner = database.Owner

	if t.ID != 0 {
		return database.InvalidError("the transfer's ID cannot be entered manually")
	}
	if t.RuleID == 0 {
		return database.InvalidError("the transfer's rule ID cannot be empty")
	}
	if t.AgentID == 0 {
		return database.InvalidError("the transfer's remote ID cannot be empty")
	}
	if t.AccountID == 0 {
		return database.InvalidError("the transfer's account ID cannot be empty")
	}
	if t.SourceFile == "" {
		return database.InvalidError("the transfer's source cannot be empty")
	}
	if t.DestFile == "" {
		return database.InvalidError("the transfer's destination cannot be empty")
	}
	if t.Start.IsZero() {
		t.Start = time.Now().Truncate(time.Second)
	}
	if t.Status == "" {
		t.Status = StatusPlanned
	}
	if !validateStatusForTransfer(t.Status) {
		return database.InvalidError("'%s' is not a valid transfer status", t.Status)
	}
	if t.Error.Code != TeOk {
		return database.InvalidError("the transfer's error code must be empty")
	}
	if t.Error.Details != "" {
		return database.InvalidError("the transfer's error message must be empty")
	}
	if t.Owner == "" {
		return database.InvalidError("the transfer's owner cannot be empty")
	}
	if t.SourceFile != filepath.Base(t.SourceFile) {
		return database.InvalidError("the source file cannot contain subdirectories")
	}
	if t.DestFile != filepath.Base(t.DestFile) {
		return database.InvalidError("the destination file cannot contain subdirectories")
	}
	if t.TrueFilepath != "" {
		t.TrueFilepath = utils.NormalizePath(t.TrueFilepath)
		if !path.IsAbs(t.TrueFilepath) {
			return database.InvalidError("the filepath must be an absolute path")
		}
	}
	rule := Rule{ID: t.RuleID}
	if err := db.Get(&rule); err != nil {
		if err == database.ErrNotFound {
			return database.InvalidError("the rule %d does not exist", t.RuleID)
		}
		return err
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
		SourceFilename: t.SourceFile,
		DestFilename:   t.DestFile,
		Rule:           rule.Name,
		Start:          t.Start,
		Stop:           stop,
		Status:         t.Status,
		Error:          t.Error,
		Step:           t.Step,
		Progress:       t.Progress,
		TaskNumber:     t.TaskNumber,
		ExtInfo:        t.ExtInfo,
	}
	return &hist, nil
}

// Update updates the transfer start, status & error in the database.
func (t *Transfer) Update(acc database.Accessor) error {
	trans := &Transfer{
		Start:      t.Start,
		Status:     t.Status,
		Step:       t.Step,
		Error:      t.Error,
		Progress:   t.Progress,
		TaskNumber: t.TaskNumber,
	}

	return acc.Update(trans)
}
