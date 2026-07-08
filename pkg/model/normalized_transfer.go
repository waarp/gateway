package model

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var (
	ErrResumeDone    = errors.New("cannot resume a transfer that is already done")
	ErrResumeRunning = errors.New("cannot resume a transfer that is already running")
	ErrResumeServer  = errors.New("cannot resume server transfers")
)

type NormalizedTransferView struct {
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

	IsTransfer           bool      `gorm:"column:is_transfer"`
	RemainingTries       int8      `gorm:"column:remaining_tries"`
	NextRetryDelay       int32     `gorm:"column:next_retry_delay"`
	RetryIncrementFactor float32   `gorm:"column:retry_increment_factor"`
	NextRetry            time.Time `gorm:"column:next_retry;type:timestamp;serializer:timestamp"`

	TransferInfo map[string]any          `gorm:"-"`
	Infos        NormalizedTransferInfos `gorm:"foreignKey:OwnerID"`
}

func (n *NormalizedTransferView) asHistoryEntry() *HistoryEntry {
	hist := &HistoryEntry{
		Identifier:       n.Identifier,
		RemoteTransferID: n.RemoteTransferID,
		IsServer:         n.IsServer,
		IsSend:           n.IsSend,
		Rule:             n.Rule,
		Account:          n.Account,
		Agent:            n.Agent,
		Client:           n.Client,
		Protocol:         n.Protocol,
		SrcFilename:      n.SrcFilename,
		DestFilename:     n.DestFilename,
		LocalPath:        n.LocalPath,
		RemotePath:       n.RemotePath,
		Filesize:         n.Filesize,
		Start:            n.Start,
		Stop:             n.Stop,
		Status:           n.Status,
		Step:             n.Step,
		Progress:         n.Progress,
		TaskNumber:       n.TaskNumber,
		ErrCode:          n.ErrCode,
		ErrDetails:       n.ErrDetails,
		TransferInfo:     n.TransferInfo,
	}

	return hist
}

func (*NormalizedTransferView) TableName() string   { return ViewNormalizedTransfers }
func (*NormalizedTransferView) Appellation() string { return "normalized transfer" }

//nolint:goconst //best keep separate
func (*NormalizedTransferView) Preloads() []string { return []string{"Infos"} }

// BeforeWrite always returns an error because writing is not allowed on views.
func (n *NormalizedTransferView) BeforeWrite(database.Access) error {
	return database.NewInternalError(errWriteOnView)
}

// BeforeDelete always returns an error because deleting is not allowed on views.
func (n *NormalizedTransferView) BeforeDelete(database.Access) error {
	return database.NewInternalError(errWriteOnView)
}

func (n *NormalizedTransferView) AfterRead(database.ReadAccess) error {
	n.TransferInfo = n.Infos.AsMap()
	return nil
}

func (n *NormalizedTransferView) CheckResumable() error {
	if !n.IsTransfer {
		return ErrResumeDone
	}

	trans := &Transfer{
		Identifier:   n.Identifier,
		Status:       n.Status,
		TransferInfo: n.TransferInfo,
	}

	if n.IsServer {
		trans.LocalAccountID = sql.NullInt64{Int64: -1, Valid: true}
	} else {
		trans.ClientID = sql.NullInt64{Int64: -1, Valid: true}
		trans.RemoteAccountID = sql.NullInt64{Int64: -1, Valid: true}
	}

	return trans.CheckResumable()
}

func (n *NormalizedTransferView) Resume(db database.Access, when time.Time) error {
	if !n.IsTransfer {
		return ErrResumeDone
	}

	var dbTrans Transfer
	if err := db.Get(&dbTrans, "id=?", n.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve transfer: %w", err)
	}

	if err := dbTrans.Resume(db, when); err != nil {
		return err
	}

	n.Status = dbTrans.Status
	n.NextRetry = dbTrans.NextRetry
	n.ErrCode = dbTrans.ErrCode
	n.ErrDetails = dbTrans.ErrDetails

	return nil
}

func (n *NormalizedTransferView) Restart(db database.Access, date time.Time) (*Transfer, error) {
	return n.asHistoryEntry().Restart(db, date)
}

type NormalizedTransferInfo struct {
	OwnerID int64  `gorm:"column:owner_id"`
	Name    string `gorm:"column:name"`
	Value   any    `gorm:"column:value;serializer:json"`
}

func (NormalizedTransferInfo) TableName() string   { return ViewNormalizedTransferInfo }
func (NormalizedTransferInfo) Appellation() string { return NameNormalizedTransferInfo }

type NormalizedTransferInfos []NormalizedTransferInfo

func (n NormalizedTransferInfos) AsMap() map[string]any {
	m := make(map[string]any, len(n))
	for _, info := range n {
		m[info.Name] = info.Value
	}

	return m
}
