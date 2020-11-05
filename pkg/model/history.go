package model

import (
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

func init() {
	database.Tables = append(database.Tables, &TransferHistory{})
}

// TransferHistory represents one record of the 'transfers_history' table.
type TransferHistory struct {
	ID               uint64               `xorm:"pk 'id'"`
	Owner            string               `xorm:"notnull 'owner'"`
	RemoteTransferID string               `xorm:"unique(histRemID) 'remote_transfer_id'"`
	IsServer         bool                 `xorm:"notnull 'is_server'"`
	IsSend           bool                 `xorm:"notnull 'is_send'"`
	Account          string               `xorm:"notnull unique(histRemID) 'account'"`
	Agent            string               `xorm:"notnull unique(histRemID) 'agent'"`
	Protocol         string               `xorm:"notnull 'protocol'"`
	SourceFilename   string               `xorm:"notnull 'source_filename'"`
	DestFilename     string               `xorm:"notnull 'dest_filename'"`
	Rule             string               `xorm:"notnull 'rule'"`
	Start            time.Time            `xorm:"notnull 'start'"`
	Stop             time.Time            `xorm:"notnull 'stop'"`
	Status           types.TransferStatus `xorm:"notnull 'status'"`
	Error            types.TransferError  `xorm:"extends"`
	Step             types.TransferStep   `xorm:"notnull varchar(50) 'step'"`
	Progress         uint64               `xorm:"notnull 'progression'"`
	TaskNumber       uint64               `xorm:"notnull 'task_number'"`
}

// TableName returns the name of the transfer history table.
func (*TransferHistory) TableName() string {
	return "transfer_history"
}

// GetID returns the transfer's ID.
func (h *TransferHistory) GetID() uint64 {
	return h.ID
}

// Validate checks if the new `TransferHistory` entry is valid and can be
// inserted in the database.
func (h *TransferHistory) Validate(database.Accessor) error {
	h.Owner = database.Owner

	if h.Owner == "" {
		return database.InvalidError("the transfer's owner cannot be empty")
	}
	if h.ID == 0 {
		return database.InvalidError("the transfer's ID cannot be empty")
	}
	if h.Rule == "" {
		return database.InvalidError("the transfer's rule cannot be empty")
	}
	if h.Account == "" {
		return database.InvalidError("the transfer's account cannot be empty")
	}
	if h.Agent == "" {
		return database.InvalidError("the transfer's agent cannot be empty")
	}
	if h.IsServer {
		if h.IsSend && h.DestFilename == "" {
			return database.InvalidError("the transfer's destination filename cannot be empty")
		} else if !h.IsSend && h.SourceFilename == "" {
			return database.InvalidError("the transfer's destination filename cannot be empty")
		}
	} else {
		if h.SourceFilename == "" {
			return database.InvalidError("the transfer's source filename cannot be empty")
		}
		if h.DestFilename == "" {
			return database.InvalidError("the transfer's destination filename cannot be empty")
		}
	}
	if h.Start.IsZero() {
		return database.InvalidError("the transfer's start date cannot be empty")
	}
	if h.Stop.IsZero() {
		return database.InvalidError("the transfer's end date cannot be empty")
	}

	if h.Stop.Before(h.Start) {
		return database.InvalidError("the transfer's end date cannot be anterior " +
			"to the start date")
	}

	if _, ok := config.ProtoConfigs[h.Protocol]; !ok {
		return database.InvalidError("'%s' is not a valid protocol", h.Protocol)
	}

	if !types.ValidateStatusForHistory(h.Status) {
		return database.InvalidError("'%s' is not a valid transfer history status", h.Status)
	}

	return nil
}

// Restart takes a History entry and converts it to a Transfer entry ready
// to be executed.
func (h *TransferHistory) Restart(acc database.Accessor, date time.Time) (*Transfer, error) {
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
		RuleID:           rule.ID,
		RemoteTransferID: h.RemoteTransferID,
		IsServer:         h.IsServer,
		AgentID:          agentID,
		AccountID:        accountID,
		SourceFile:       h.SourceFilename,
		DestFile:         h.DestFilename,
		Start:            date.UTC(),
		Status:           types.StatusPlanned,
		Step:             types.StepNone,
		Owner:            h.Owner,
	}, nil
}
