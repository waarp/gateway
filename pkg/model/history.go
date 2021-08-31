package model

import (
	"path"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

func init() {
	database.AddTable(&HistoryEntry{})
}

// HistoryEntry represents one record of the 'transfers_history' table.
type HistoryEntry struct {
	ID               uint64               `xorm:"pk 'id'"`
	Owner            string               `xorm:"notnull 'owner'"`
	RemoteTransferID string               `xorm:"unique(histRemID) 'remote_transfer_id'"`
	IsServer         bool                 `xorm:"notnull 'is_server'"`
	IsSend           bool                 `xorm:"notnull 'is_send'"`
	Rule             string               `xorm:"notnull 'rule'"`
	Agent            string               `xorm:"notnull unique(histRemID) 'agent'"`
	Account          string               `xorm:"notnull unique(histRemID) 'account'"`
	Protocol         string               `xorm:"notnull 'protocol'"`
	LocalPath        string               `xorm:"notnull 'local_path'"`
	RemotePath       string               `xorm:"notnull 'remote_path'"`
	Filesize         int64                `xorm:"notnull 'filesize'"`
	Start            time.Time            `xorm:"notnull timestampz 'start'"`
	Stop             time.Time            `xorm:"timestampz 'stop'"`
	Status           types.TransferStatus `xorm:"notnull varchar(50) 'status'"`
	Step             types.TransferStep   `xorm:"notnull varchar(50) 'step'"`
	Progress         uint64               `xorm:"notnull 'progression'"`
	TaskNumber       uint64               `xorm:"notnull 'task_number'"`
	Error            types.TransferError  `xorm:"extends"`
}

// TableName returns the name of the transfer history table.
func (*HistoryEntry) TableName() string {
	return TableHistory
}

// Appellation returns the name of 1 element of the transfer history table.
func (*HistoryEntry) Appellation() string {
	return "history entry"
}

// GetID returns the transfer's ID.
func (h *HistoryEntry) GetID() uint64 {
	return h.ID
}

// BeforeWrite checks if the new `HistoryEntry` entry is valid and can be
// inserted in the database.
func (h *HistoryEntry) BeforeWrite(database.ReadAccess) database.Error {
	h.Owner = conf.GlobalConfig.GatewayName

	if h.Owner == "" {
		return database.NewValidationError("the transfer's owner cannot be empty")
	}
	if h.ID == 0 {
		return database.NewValidationError("the transfer's ID cannot be empty")
	}
	if h.Rule == "" {
		return database.NewValidationError("the transfer's rule cannot be empty")
	}
	if h.Account == "" {
		return database.NewValidationError("the transfer's account cannot be empty")
	}
	if h.Agent == "" {
		return database.NewValidationError("the transfer's agent cannot be empty")
	}
	if h.LocalPath == "" {
		return database.NewValidationError("the local filepath cannot be empty")
	}
	if h.RemotePath == "" {
		return database.NewValidationError("the remote filepath cannot be empty")
	}
	if h.Start.IsZero() {
		return database.NewValidationError("the transfer's start date cannot be empty")
	}

	if !h.Stop.IsZero() && h.Stop.Before(h.Start) {
		return database.NewValidationError("the transfer's end date cannot be anterior " +
			"to the start date")
	}

	if _, ok := config.ProtoConfigs[h.Protocol]; !ok {
		return database.NewValidationError("'%s' is not a valid protocol", h.Protocol)
	}

	if !types.ValidateStatusForHistory(h.Status) {
		return database.NewValidationError("'%s' is not a valid transfer history status", h.Status)
	}

	return nil
}

// Restart takes a HistoryEntry entry and converts it to a Transfer entry ready
// to be executed.
func (h *HistoryEntry) Restart(db database.Access, date time.Time) (*Transfer, database.Error) {
	rule := &Rule{}
	if err := db.Get(rule, "name=? AND send=?", h.Rule, h.IsSend).Run(); err != nil {
		return nil, err
	}
	var agentID, accountID uint64
	if h.IsServer {
		agent := &LocalAgent{}
		if err := db.Get(agent, "owner=? AND name=?", h.Owner, h.Agent).Run(); err != nil {
			return nil, err
		}
		account := &LocalAccount{}
		if err := db.Get(account, "local_agent_id=? AND login=?", agent.ID, h.Account).
			Run(); err != nil {
			return nil, err
		}
		agentID = agent.ID
		accountID = account.ID
	} else {
		agent := &RemoteAgent{}
		if err := db.Get(agent, "name=?", h.Agent).Run(); err != nil {
			return nil, err
		}
		account := &RemoteAccount{}
		if err := db.Get(account, "remote_agent_id=? AND login=?", agent.ID, h.Account).
			Run(); err != nil {
			return nil, err
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
		LocalPath:        path.Base(h.LocalPath),
		RemotePath:       path.Base(h.RemotePath),
		Start:            date,
		Status:           types.StatusPlanned,
		Step:             types.StepNone,
		Owner:            h.Owner,
	}, nil
}
