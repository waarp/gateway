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
	ID         uint64         `xorm:"pk autoincr <- 'id'" json:"id"`
	RuleID     uint64         `xorm:"notnull 'rule_id'" json:"ruleID"`
	IsServer   bool           `xorm:"notnull 'is_server'" json:"isServer'"`
	RemoteID   uint64         `xorm:"notnull 'remote_id'" json:"remoteID"`
	AccountID  uint64         `xorm:"notnull 'account_id'" json:"accountID"`
	SourcePath string         `xorm:"notnull 'source_path'" json:"sourcePath"`
	DestPath   string         `xorm:"notnull 'dest_path'" json:"destPath"`
	Start      time.Time      `xorm:"notnull 'start'" json:"start"`
	Status     TransferStatus `xorm:"notnull 'status'" json:"status"`
	Owner      string         `xorm:"notnull 'owner'"`
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
	if t.RemoteID == 0 {
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
	if t.Status != StatusPlanned {
		return database.InvalidError("The transfer's status must be 'planned'")
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
	remote := RemoteAgent{ID: t.RemoteID}
	if err := acc.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return database.InvalidError("The partner %d does not exist", t.RemoteID)
		}
		return err
	}
	if res, err := acc.Query("SELECT id FROM remote_accounts WHERE id=? AND remote_agent_id=?",
		t.AccountID, t.RemoteID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("The agent %d does not have an account %d",
			t.RemoteID, t.AccountID)
	}

	if remote.Protocol == "sftp" {
		if res, err := acc.Query("SELECT id FROM certificates WHERE owner_type=? AND owner_id=?",
			(&RemoteAgent{}).TableName(), t.RemoteID); err != nil {
			return err
		} else if len(res) == 0 {
			return database.InvalidError("No certificate found for agent %d", t.RemoteID)
		}
	}
	return nil
}

func (t *Transfer) validateServerTransfer(acc database.Accessor) error {
	remote := LocalAgent{ID: t.RemoteID}
	if err := acc.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return database.InvalidError("The partner %d does not exist", t.RemoteID)
		}
		return err
	}
	if res, err := acc.Query("SELECT id FROM local_accounts WHERE id=? AND local_agent_id=?",
		t.AccountID, t.RemoteID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("The agent %d does not have an account %d",
			t.RemoteID, t.AccountID)
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
	t.Status = StatusPlanned
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
	if t.RemoteID != 0 {
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

	if !validateStatusForTransfer(t.Status) {
		return database.InvalidError("'%s' is not a valid transfer status", t.Status)
	}

	return nil
}

// ToHistory converts the `Transfer` entry into an equivalent `TransferHistory`
// entry with the given time as the end date.
func (t *Transfer) ToHistory(acc database.Accessor, stop time.Time) (*TransferHistory, error) {

	rule := &Rule{ID: t.RuleID}
	if err := acc.Get(rule); err != nil {
		return nil, err
	}
	remote := &RemoteAgent{ID: t.RemoteID}
	if err := acc.Get(remote); err != nil {
		return nil, err
	}
	account := &RemoteAccount{ID: t.AccountID}
	if err := acc.Get(account); err != nil {
		return nil, err
	}

	if !validateStatusForHistory(t.Status) {
		return nil, fmt.Errorf(
			"a transfer cannot be recorded in history with status '%s'",
			t.Status,
		)
	}

	hist := TransferHistory{
		ID:             t.ID,
		Owner:          t.Owner,
		Account:        account.Login,
		Remote:         remote.Name,
		Protocol:       remote.Protocol,
		SourceFilename: t.SourcePath,
		DestFilename:   t.DestPath,
		Rule:           rule.Name,
		IsSend:         !rule.IsGet,
		Start:          t.Start,
		Stop:           stop,
		Status:         t.Status,
	}

	return &hist, nil
}
