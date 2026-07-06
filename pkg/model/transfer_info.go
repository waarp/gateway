package model

import (
	"database/sql"
	"fmt"
	"slices"
)

type ResumeSyncError struct {
	ID       int64
	ParentID any
}

func (e *ResumeSyncError) Error() string {
	return fmt.Sprintf(
		"cannot resume transfer %d: it is a child of transfer %v, resume the parent instead",
		e.ID, e.ParentID)
}

const (
	// FollowID defines the name of the transfer info value containing the R66
	// follow ID.
	FollowID = "__followID__"

	SyncTransferID   = "__syncTransferID__"
	SyncTransferRank = "__syncTransferRank__"
)

// TransferInfo represents the transfer_info database table, which contains all the
// protocol-specific information attached to a transfer.
type TransferInfo struct {
	// The owner of the info pair. Only one can be valid at a time.
	TransferID sql.NullInt64 `gorm:"column:transfer_id"`
	HistoryID  sql.NullInt64 `gorm:"column:history_id"`

	Name  string `gorm:"column:name"`                  // The info's key.
	Value any    `gorm:"column:value;serializer:json"` // The info's value.
}

func (TransferInfo) TableName() string   { return TableTransferInfo }
func (TransferInfo) Appellation() string { return NameTransferInfo }

type TransferInfos []TransferInfo

func (TransferInfos) TableName() string { return TableTransferInfo }
func (TransferInfos) Elem() string      { return NameTransferInfo }

func (t TransferInfos) asMap() map[string]any {
	m := make(map[string]any, len(t))
	for _, info := range t {
		m[info.Name] = info.Value
	}

	return m
}

func (t TransferInfos) ToHist() TransferInfos {
	newInfos := slices.Clone(t)
	for i := range newInfos {
		newInfos[i].HistoryID = newInfos[i].TransferID
		newInfos[i].TransferID = sql.NullInt64{}
	}

	return newInfos
}
