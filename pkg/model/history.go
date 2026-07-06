package model

import (
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// HistoryEntry represents one record of the 'transfers_history' table.
type HistoryEntry struct {
	Identifier
	Owner            string                  `gorm:"column:owner"`
	RemoteTransferID string                  `gorm:"column:remote_transfer_id"`
	IsServer         bool                    `gorm:"column:is_server"`
	IsSend           bool                    `gorm:"column:is_send"`
	Rule             string                  `gorm:"column:rule"`
	Account          string                  `gorm:"column:account"`
	Agent            string                  `gorm:"column:agent"`
	Client           string                  `gorm:"column:client"`
	Protocol         string                  `gorm:"column:protocol"`
	SrcFilename      string                  `gorm:"column:src_filename"`
	DestFilename     string                  `gorm:"column:dest_filename"`
	LocalPath        string                  `gorm:"column:local_path"`
	RemotePath       string                  `gorm:"column:remote_path"`
	Filesize         int64                   `gorm:"column:filesize"`
	Start            time.Time               `gorm:"column:start;type:timestamp;serializer:timestamp"`
	Stop             time.Time               `gorm:"column:stop;type:timestamp;serializer:timestamp"`
	Status           types.TransferStatus    `gorm:"column:status"`
	Step             types.TransferStep      `gorm:"column:step"`
	Progress         int64                   `gorm:"column:progress"`
	TaskNumber       int8                    `gorm:"column:task_number"`
	ErrCode          types.TransferErrorCode `gorm:"column:error_code"`
	ErrDetails       string                  `gorm:"column:error_details"`
	Infos            TransferInfos           `gorm:"foreignKey:HistoryID"`
	TransferInfo     map[string]any          `gorm:"-"`
}

func (*HistoryEntry) TableName() string   { return TableHistory }
func (*HistoryEntry) Appellation() string { return NameHistory }

//nolint:goconst //best keep separate
func (*HistoryEntry) Preloads() []string { return []string{"Infos"} }

func (h *HistoryEntry) TransferID() (int64, error) {
	id, err := snowflake.ParseString(h.RemoteTransferID)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the remote transfer ID: %w", err)
	}

	return id.Int64(), nil
}

// BeforeWrite checks if the new `HistoryEntry` entry is valid and can be
// inserted in the database.
//
//nolint:funlen,gocyclo,cyclop,gocognit // validation can be long...
func (h *HistoryEntry) BeforeWrite(_ database.Access) error {
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

	if h.RemotePath != "" && h.LocalPath == "" {
		return database.NewValidationError("the local filepath cannot be empty")
	}

	if !h.IsServer && h.LocalPath != "" && h.RemotePath == "" {
		return database.NewValidationError("the remote filepath cannot be empty")
	}

	if h.LocalPath != "" {
		if err := fs.ValidPath(h.LocalPath); err != nil {
			return database.NewValidationErrorf("invalid local path: %v", err)
		}
	}

	if h.Start.IsZero() {
		return database.NewValidationError("the transfer's start date cannot be empty")
	}

	if !h.Stop.IsZero() && h.Stop.Before(h.Start) {
		return database.NewValidationError("the transfer's end date cannot be anterior " +
			"to the start date")
	}

	if !IsValidProtocol(h.Protocol) {
		return database.NewValidationErrorf("%q is not a valid protocol", h.Protocol)
	}

	if !types.ValidateStatusForHistory(h.Status) {
		return database.NewValidationErrorf("%q is not a valid transfer history status", h.Status)
	}

	if h.TransferInfo == nil {
		h.TransferInfo = map[string]any{}
	}

	if !h.IsServer && h.Client == "" {
		return database.NewValidationError("the transfer's client is missing")
	} else if h.IsServer && h.Client != "" {
		return database.NewValidationError("server transfers cannot have a client")
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
		if err := db.Get(agent, "name=?", h.Agent).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve local agent: %w", err)
		}

		account := &LocalAccount{}
		if err := db.Get(account, "local_agent_id=? AND login=?", agent.ID, h.Account).
			Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve local account: %w", err)
		}

		trans.LocalAccountID = account.NullableID()
	} else {
		client := &Client{}
		if err := db.Get(client, "name=?", h.Client).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve client: %w", err)
		}

		agent := &RemoteAgent{}
		if err := db.Get(agent, "name=?", h.Agent).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve remote agent: %w", err)
		}

		account := &RemoteAccount{}
		if err := db.Get(account, "remote_agent_id=? AND login=?", agent.ID, h.Account).
			Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve remote account: %w", err)
		}

		trans.ClientID = client.NullableID()
		trans.RemoteAccountID = account.NullableID()
	}

	return trans, nil
}

func (h *HistoryEntry) AfterInsert(db database.Access) error {
	h.Infos = make(TransferInfos, 0, len(h.TransferInfo))
	for k, v := range h.TransferInfo {
		h.Infos = append(h.Infos, TransferInfo{HistoryID: h.NullableID(), Name: k, Value: v})
	}

	if err := database.InsertBatch[TransferInfo](db, h.Infos...); err != nil {
		return fmt.Errorf("failed to insert transfer info: %w", err)
	}

	return nil
}

func (h *HistoryEntry) AfterRead(database.ReadAccess) error {
	h.TransferInfo = h.Infos.asMap()

	return nil
}
