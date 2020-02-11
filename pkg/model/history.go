package model

import (
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

func init() {
	database.Tables = append(database.Tables, &TransferHistory{})
}

// TransferHistory represents one record of the 'transfers_history' table.
type TransferHistory struct {
	ID             uint64         `xorm:"pk autoincr <- 'id'"`
	Owner          string         `xorm:"notnull 'owner'"`
	IsServer       bool           `xorm:"notnull 'is_server'"`
	IsSend         bool           `xorm:"notnull 'is_send'"`
	Account        string         `xorm:"notnull 'account'"`
	Agent          string         `xorm:"notnull 'agent'"`
	Protocol       string         `xorm:"notnull 'protocol'"`
	SourceFilename string         `xorm:"notnull 'source_filename'"`
	DestFilename   string         `xorm:"notnull 'dest_filename'"`
	Rule           string         `xorm:"notnull 'rule'"`
	Start          time.Time      `xorm:"notnull 'start'"`
	Stop           time.Time      `xorm:"notnull 'stop'"`
	Status         TransferStatus `xorm:"notnull 'status'"`
	Error          TransferError  `xorm:"extends"`
	Step           TransferStep   `xorm:"notnull 'step'"`
	Progress       uint64         `xorm:"notnull 'progression'"`
	TaskNumber     uint64         `xorm:"notnull 'task_number'"`
	ExtInfo        []byte         `xorm:"ext_info"`
}

// TableName returns the name of the transfer history table.
func (*TransferHistory) TableName() string {
	return "transfer_history"
}

// BeforeInsert is called before inserting the transfer in the database. Its
// role is to set the Owner, to force the Status and to set a Start time if none
// was entered.
func (h *TransferHistory) BeforeInsert(database.Accessor) error {
	h.Owner = database.Owner
	return nil
}

// ValidateInsert checks if the new `TransferHistory` entry is valid and can be
// inserted in the database.
func (h *TransferHistory) ValidateInsert(database.Accessor) error {
	if h.Owner == "" {
		return database.InvalidError("The transfer's owner cannot be empty")
	}
	if h.ID == 0 {
		return database.InvalidError("The transfer's ID cannot be empty")
	}
	if h.Rule == "" {
		return database.InvalidError("The transfer's rule cannot be empty")
	}
	if h.Account == "" {
		return database.InvalidError("The transfer's account cannot be empty")
	}
	if h.Agent == "" {
		return database.InvalidError("The transfer's agent cannot be empty")
	}
	if h.IsServer {
		if h.IsSend && h.DestFilename == "" {
			return database.InvalidError("The transfer's destination filename cannot be empty")
		} else if !h.IsSend && h.SourceFilename == "" {
			return database.InvalidError("The transfer's destination filename cannot be empty")
		}
	} else {
		if h.SourceFilename == "" {
			return database.InvalidError("The transfer's source filename cannot be empty")
		}
		if h.DestFilename == "" {
			return database.InvalidError("The transfer's destination filename cannot be empty")
		}
	}
	if h.Start.IsZero() {
		return database.InvalidError("The transfer's start date cannot be empty")
	}
	if h.Stop.IsZero() {
		return database.InvalidError("The transfer's end date cannot be empty")
	}

	if h.Stop.Before(h.Start) {
		return database.InvalidError("The transfer's end date cannot be anterior " +
			"to the start date")
	}

	if _, ok := config.ProtoConfigs[h.Protocol]; !ok {
		return database.InvalidError("'%s' is not a valid protocol", h.Protocol)
	}

	if !validateStatusForHistory(h.Status) {
		return database.InvalidError("'%s' is not a valid transfer history status", h.Status)
	}

	return nil
}

// ValidateUpdate is called before updating an existing `TransferHistory` entry
// from the database. It checks whether the updated entry is valid or not.
func (h *TransferHistory) ValidateUpdate(database.Accessor, uint64) error {
	if h.ID != 0 {
		return database.InvalidError("The transfer's ID cannot be changed")
	}
	if h.Owner != "" {
		return database.InvalidError("The transfer's owner cannot be changed")
	}
	if h.Rule != "" {
		return database.InvalidError("The transfer's rule cannot be changed")
	}
	if h.Account != "" {
		return database.InvalidError("The transfer's account cannot be changed")
	}
	if h.Agent != "" {
		return database.InvalidError("The transfer's agent cannot be changed")
	}
	if h.SourceFilename != "" {
		return database.InvalidError("The transfer's source filename cannot be changed")
	}
	if h.DestFilename != "" {
		return database.InvalidError("The transfer's destination filename cannot be changed")
	}
	if h.Protocol != "" {
		return database.InvalidError("The transfer's protocol cannot be changed")
	}

	if h.Start.IsZero() {
		return database.InvalidError("The transfer's start cannot be empty")
	}
	if h.Stop.IsZero() {
		return database.InvalidError("The transfer's stop cannot be empty")
	}

	if h.Stop.Before(h.Start) {
		return database.InvalidError("The transfer's end date cannot be anterior " +
			"to the start date")
	}

	if !validateStatusForHistory(h.Status) {
		return database.InvalidError("'%s' is not a valid transfer history status", h.Status)
	}

	return nil
}

// Reprogram takes a History entry and converts it to a Transfer entry ready
// to be executed.
func (h *TransferHistory) Reprogram(acc database.Accessor, date time.Time) (*Transfer, error) {
	rule := &Rule{Name: h.Rule, IsSend: h.IsSend}
	if err := acc.Get(rule); err != nil {
		return nil, fmt.Errorf("failed to retrieve rule: %s", err)
	}
	var agentID, accountID uint64
	if h.IsServer {
		agent := &LocalAgent{Owner: h.Owner, Name: h.Agent}
		if err := acc.Get(agent); err != nil {
			return nil, fmt.Errorf("failed to retrieve local agent: %s", err)
		}
		account := &LocalAccount{LocalAgentID: agentID, Login: h.Account}
		if err := acc.Get(account); err != nil {
			return nil, fmt.Errorf("failed to retrieve local account: %s", err)
		}
		agentID = agent.ID
		accountID = account.ID
	} else {
		agent := &RemoteAgent{Name: h.Agent}
		if err := acc.Get(agent); err != nil {
			return nil, fmt.Errorf("failed to retrieve remote agent: %s", err)
		}
		account := &RemoteAccount{RemoteAgentID: agentID, Login: h.Account}
		if err := acc.Get(account); err != nil {
			return nil, fmt.Errorf("failed to retrieve remote account: %s", err)
		}
		agentID = agent.ID
		accountID = account.ID
	}

	return &Transfer{
		RuleID:     rule.ID,
		IsServer:   h.IsServer,
		AgentID:    agentID,
		AccountID:  accountID,
		SourcePath: h.SourceFilename,
		DestPath:   h.DestFilename,
		Start:      date.UTC(),
		Status:     StatusPlanned,
		Step:       h.Step,
		Owner:      h.Owner,
		Progress:   h.Progress,
		TaskNumber: h.TaskNumber,
		ExtInfo:    h.ExtInfo,
	}, nil
}
