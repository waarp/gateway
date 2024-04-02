package model

import (
	"database/sql"
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// HistoryEntry represents one record of the 'transfers_history' table.
type HistoryEntry struct {
	ID               int64                   `xorm:"id"`
	Owner            string                  `xorm:"owner"`
	RemoteTransferID string                  `xorm:"remote_transfer_id"`
	IsServer         bool                    `xorm:"is_server"`
	IsSend           bool                    `xorm:"is_send"`
	Rule             string                  `xorm:"rule"`
	Account          string                  `xorm:"account"`
	Agent            string                  `xorm:"agent"`
	Client           string                  `xorm:"client"`
	Protocol         string                  `xorm:"protocol"`
	SrcFilename      string                  `xorm:"src_filename"`
	DestFilename     string                  `xorm:"dest_filename"`
	LocalPath        types.URL               `xorm:"local_path"`
	RemotePath       string                  `xorm:"remote_path"`
	Filesize         int64                   `xorm:"filesize"`
	Start            time.Time               `xorm:"start DATETIME(6) UTC"`
	Stop             time.Time               `xorm:"stop DATETIME(6) UTC"`
	Status           types.TransferStatus    `xorm:"status"`
	Step             types.TransferStep      `xorm:"step"`
	Progress         int64                   `xorm:"progress"`
	TaskNumber       int8                    `xorm:"task_number"`
	ErrCode          types.TransferErrorCode `xorm:"error_code"`
	ErrDetails       string                  `xorm:"error_details"`
}

func (*HistoryEntry) TableName() string   { return TableHistory }
func (*HistoryEntry) Appellation() string { return "history entry" }
func (h *HistoryEntry) GetID() int64      { return h.ID }

func (h *HistoryEntry) getTransInfoCondition() (string, int64) {
	return "history_id=?", h.ID
}

func (h *HistoryEntry) setTransInfoOwner(info *TransferInfo) {
	info.HistoryID = utils.NewNullInt64(h.ID)
}

// BeforeWrite checks if the new `HistoryEntry` entry is valid and can be
// inserted in the database.
//
//nolint:funlen,gocyclo,cyclop,gocognit // validation can be long...
func (h *HistoryEntry) BeforeWrite(db database.ReadAccess) error {
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

	if h.IsSend {
		if h.SrcFilename == "" {
			return database.NewValidationError("the source file is missing")
		}
	} else if h.IsServer && h.DestFilename == "" {
		return database.NewValidationError("the destination file is missing")
	}

	if h.RemotePath != "" && h.LocalPath.Path == "" {
		return database.NewValidationError("the local filepath cannot be empty")
	}

	if !h.IsServer && h.LocalPath.Path != "" && h.RemotePath == "" {
		return database.NewValidationError("the remote filepath cannot be empty")
	}

	if h.LocalPath.Path != "" && !filesystems.DoesFileSystemExist(h.LocalPath.Scheme) {
		return database.NewValidationError("unknown local path scheme %q", h.LocalPath.Scheme)
	}

	if h.Start.IsZero() {
		return database.NewValidationError("the transfer's start date cannot be empty")
	}

	if !h.Stop.IsZero() && h.Stop.Before(h.Start) {
		return database.NewValidationError("the transfer's end date cannot be anterior " +
			"to the start date")
	}

	if !ConfigChecker.IsValidProtocol(h.Protocol) {
		return database.NewValidationError("'%s' is not a valid protocol", h.Protocol)
	}

	if !types.ValidateStatusForHistory(h.Status) {
		return database.NewValidationError("'%s' is not a valid transfer history status", h.Status)
	}

	if !h.IsServer && h.Client == "" {
		return database.NewValidationError("the transfer's client is missing")
	} else if h.IsServer && h.Client != "" {
		return database.NewValidationError("server transfers cannot have a client")
	}

	if n, err := db.Count(&HistoryEntry{}).Where("id=?", h.ID).Run(); err != nil {
		return database.NewInternalError(err)
	} else if n != 0 {
		return database.NewValidationError("a history entry with the same ID already exist")
	}

	if n, err := db.Count(&HistoryEntry{}).Where("remote_transfer_id=? AND "+
		"is_server=? AND agent=? AND account=?", h.RemoteTransferID, h.IsServer,
		h.Agent, h.Account).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate history entries: %w", err)
	} else if n != 0 {
		//nolint:goconst //too specific
		return database.NewValidationError("a history entry from the same "+
			"requester %q to the same agent %q with the same remote ID %q "+
			"already exist", h.Account, h.Agent, h.RemoteTransferID)
	}

	return nil
}

// Restart takes a HistoryEntry entry and converts it to a Transfer entry ready
// to be executed.
func (h *HistoryEntry) Restart(db database.Access, date time.Time) (*Transfer, error) {
	rule := &Rule{}
	if err := db.Get(rule, "name=? AND is_send=?", h.Rule, h.IsSend).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve transfer rule: %w", err)
	}

	trans := &Transfer{
		RuleID:       rule.ID,
		SrcFilename:  h.SrcFilename,
		DestFilename: h.DestFilename,
		Start:        date,
		Status:       types.StatusPlanned,
		Step:         types.StepNone,
		Owner:        h.Owner,
	}

	if h.IsServer {
		agent := &LocalAgent{}
		if err := db.Get(agent, "owner=? AND name=?", h.Owner, h.Agent).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve local agent: %w", err)
		}

		account := &LocalAccount{}
		if err := db.Get(account, "local_agent_id=? AND login=?", agent.ID, h.Account).
			Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve local account: %w", err)
		}

		trans.LocalAccountID = sql.NullInt64{Valid: true, Int64: account.ID}
	} else {
		client := &Client{}
		if err := db.Get(client, "name=? AND owner=?", h.Client, h.Owner).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve client: %w", err)
		}

		agent := &RemoteAgent{}
		if err := db.Get(agent, "name=? AND owner=?", h.Agent, h.Owner).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve remote agent: %w", err)
		}

		account := &RemoteAccount{}
		if err := db.Get(account, "remote_agent_id=? AND login=?", agent.ID, h.Account).
			Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve remote account: %w", err)
		}

		trans.ClientID = utils.NewNullInt64(client.ID)
		trans.RemoteAccountID = utils.NewNullInt64(account.ID)
	}

	return trans, nil
}

// GetTransferInfo returns the list of the transfer's TransferInfo as a map of interfaces.
func (h *HistoryEntry) GetTransferInfo(db database.ReadAccess) (map[string]any, error) {
	return getTransferInfo(db, h)
}

// SetTransferInfo replaces all the TransferInfo in the database of the given
// history entry by those given in the map parameter.
func (h *HistoryEntry) SetTransferInfo(db database.Access, info map[string]any) error {
	return setTransferInfo(db, h, info)
}
