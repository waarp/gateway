package model

import (
	"database/sql"
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// HistoryEntry represents one record of the 'transfers_history' table.
type HistoryEntry struct {
	ID               int64                `xorm:"BIGINT PK 'id'"`
	Owner            string               `xorm:"VARCHAR(100) NOTNULL 'owner'"`
	RemoteTransferID string               `xorm:"VARCHAR(100) NOTNULL 'remote_transfer_id'"`
	IsServer         bool                 `xorm:"BOOL NOTNULL 'is_server'"`
	IsSend           bool                 `xorm:"BOOL NOTNULL 'is_send'"`
	Rule             string               `xorm:"VARCHAR(100) NOTNULL 'rule'"`
	Account          string               `xorm:"VARCHAR(100) NOTNULL 'account'"`
	Agent            string               `xorm:"VARCHAR(100) NOTNULL 'agent'"`
	Protocol         string               `xorm:"VARCHAR(50) NOTNULL 'protocol'"`
	LocalPath        string               `xorm:"TEXT NOTNULL 'local_path'"`
	RemotePath       string               `xorm:"TEXT NOTNULL 'remote_path'"`
	Filesize         int64                `xorm:"BIGINT NOTNULL DEFAULT(-1) 'filesize'"`
	Start            time.Time            `xorm:"DATETIME(6) UTC NOTNULL 'start'"`
	Stop             time.Time            `xorm:"DATETIME(6) UTC 'stop'"`
	Status           types.TransferStatus `xorm:"VARCHAR(50) NOTNULL 'status'"`
	Step             types.TransferStep   `xorm:"VARCHAR(50) NOTNULL 'step'"`
	Progress         int64                `xorm:"BIGINT NOTNULL DEFAULT(0) 'progress'"`
	TaskNumber       int16                `xorm:"SMALLINT NOTNULL DEFAULT(0) 'task_number'"`
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
func (h *HistoryEntry) GetID() int64 {
	return h.ID
}

// BeforeWrite checks if the new `HistoryEntry` entry is valid and can be
// inserted in the database.
//
//nolint:funlen,gocyclo,cyclop // validation can be long...
func (h *HistoryEntry) BeforeWrite(db database.ReadAccess) database.Error {
	h.Owner = conf.GlobalConfig.GatewayName

	if h.Owner == "" {
		return database.NewValidationError("the transfer's owner cannot be empty")
	}

	if h.ID == 0 {
		return database.NewValidationError("the transfer's ID cannot be empty")
	}

	if h.RemoteTransferID == "" {
		return database.NewValidationError("the transfer's remote ID is missing")
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

	if h.RemoteTransferID != "" {
		if n, err := db.Count(&HistoryEntry{}).Where("remote_transfer_id=? AND agent=? AND account=?",
			h.RemoteTransferID, h.Agent, h.Account).Run(); err != nil {
			return err
		} else if n != 0 {
			return database.NewValidationError("a history entry from the same " +
				"partner with the same remote ID already exist")
		}
	}

	return nil
}

// Restart takes a HistoryEntry entry and converts it to a Transfer entry ready
// to be executed.
func (h *HistoryEntry) Restart(db database.Access, date time.Time) (*Transfer, database.Error) {
	rule := &Rule{}
	if err := db.Get(rule, "name=? AND is_send=?", h.Rule, h.IsSend).Run(); err != nil {
		return nil, err
	}

	trans := &Transfer{
		RuleID:     rule.ID,
		LocalPath:  path.Base(h.LocalPath),
		RemotePath: path.Base(h.RemotePath),
		Start:      date,
		Status:     types.StatusPlanned,
		Step:       types.StepNone,
		Owner:      h.Owner,
	}

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

		trans.LocalAccountID = sql.NullInt64{Valid: true, Int64: account.ID}
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

		trans.RemoteAccountID = sql.NullInt64{Valid: true, Int64: account.ID}
	}

	return trans, nil
}

// GetTransferInfo returns the list of the transfer's TransferInfo as a map of interfaces.
func (h *HistoryEntry) GetTransferInfo(db database.ReadAccess) (map[string]interface{}, database.Error) {
	return getTransferInfo(db, h.ID)
}

// SetTransferInfo replaces all the TransferInfo in the database of the given
// history entry by those given in the map parameter.
func (h *HistoryEntry) SetTransferInfo(db *database.DB, info map[string]interface{}) database.Error {
	return setTransferInfo(db, info, h.ID, true)
}
