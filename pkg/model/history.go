package model

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&TransferHistory{})
}

// TransferHistory represents one record of the 'transfers_history' table.
type TransferHistory struct {
	ID               uint64               `xorm:"pk 'id'"`
	Owner            string               `xorm:"notnull 'owner'"`
	RemoteTransferID string               `xorm:"'remote_transfer_id'"`
	IsServer         bool                 `xorm:"notnull 'is_server'"`
	IsSend           bool                 `xorm:"notnull 'is_send'"`
	Account          string               `xorm:"notnull 'account'"`
	Agent            string               `xorm:"notnull 'agent'"`
	Protocol         string               `xorm:"notnull 'protocol'"`
	SourceFilename   string               `xorm:"notnull 'source_filename'"`
	DestFilename     string               `xorm:"notnull 'dest_filename'"`
	Rule             string               `xorm:"notnull 'rule'"`
	Start            time.Time            `xorm:"notnull timestampz 'start'"`
	Stop             time.Time            `xorm:"timestampz 'stop'"`
	Status           types.TransferStatus `xorm:"notnull varchar(50) 'status'"`
	Error            types.TransferError  `xorm:"extends"`
	Step             types.TransferStep   `xorm:"notnull varchar(50) 'step'"`
	Progress         uint64               `xorm:"notnull 'progression'"`
	TaskNumber       uint64               `xorm:"notnull 'task_number'"`
}

// TableName returns the name of the transfer history table.
func (*TransferHistory) TableName() string {
	return TableHistory
}

// Appellation returns the name of 1 element of the transfer history table.
func (*TransferHistory) Appellation() string {
	return "history entry"
}

// GetID returns the transfer's ID.
func (h *TransferHistory) GetID() uint64 {
	return h.ID
}

// BeforeWrite checks if the new `TransferHistory` entry is valid and can be
// inserted in the database.
//nolint:funlen,gocyclo,cyclop // validation can be long...
func (h *TransferHistory) BeforeWrite(db database.ReadAccess) database.Error {
	h.Owner = database.Owner

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

	if h.IsServer {
		if h.IsSend && h.DestFilename == "" {
			return database.NewValidationError("the transfer's destination filename cannot be empty")
		} else if !h.IsSend && h.SourceFilename == "" {
			return database.NewValidationError("the transfer's destination filename cannot be empty")
		}
	} else {
		if h.SourceFilename == "" {
			return database.NewValidationError("the transfer's source filename cannot be empty")
		}
		if h.DestFilename == "" {
			return database.NewValidationError("the transfer's destination filename cannot be empty")
		}
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
		if n, err := db.Count(&TransferHistory{}).Where("remote_transfer_id=? AND agent=? AND account=?",
			h.RemoteTransferID, h.Agent, h.Account).Run(); err != nil {
			return err
		} else if n != 0 {
			return database.NewValidationError("a history entry from the same " +
				"partner with the same remote ID already exist")
		}
	}

	return nil
}

// Restart takes a History entry and converts it to a Transfer entry ready
// to be executed.
func (h *TransferHistory) Restart(db database.Access, date time.Time) (*Transfer, database.Error) {
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
		SourceFile:       h.SourceFilename,
		DestFile:         h.DestFilename,
		Start:            date,
		Status:           types.StatusPlanned,
		Step:             types.StepNone,
		Owner:            h.Owner,
	}, nil
}
