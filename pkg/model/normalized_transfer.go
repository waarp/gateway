package model

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type NormalizedTransferView struct {
	HistoryEntry `xorm:"extends"`

	IsTransfer           bool      `xorm:"BOOL 'is_transfer'"`
	RemainingTries       int8      `xorm:"remaining_tries"`
	NextRetryDelay       int32     `xorm:"next_retry_delay"`
	RetryIncrementFactor float32   `xorm:"retry_increment_factor"`
	NextRetry            time.Time `xorm:"next_retry DATETIME(6) UTC"`
}

func (*NormalizedTransferView) TableName() string   { return ViewNormalizedTransfers }
func (*NormalizedTransferView) Appellation() string { return "normalized transfer" }
func (n *NormalizedTransferView) GetID() int64      { return n.ID }

// BeforeWrite always returns an error because writing is not allowed on views.
func (n *NormalizedTransferView) BeforeWrite(database.Access) error {
	return database.NewInternalError(errWriteOnView)
}

// BeforeDelete always returns an error because deleting is not allowed on views.
func (n *NormalizedTransferView) BeforeDelete(database.Access) error {
	return database.NewInternalError(errWriteOnView)
}

// GetTransferInfo returns the list of the transfer's TransferInfo as a map of interfaces.
func (n *NormalizedTransferView) GetTransferInfo(db database.ReadAccess) (map[string]any, error) {
	return getTransferInfo(db, n)
}

func (n *NormalizedTransferView) getTransInfoCondition() (string, int64) {
	if n.IsTransfer {
		return (&Transfer{ID: n.ID}).getTransInfoCondition()
	}

	return n.HistoryEntry.getTransInfoCondition()
}

func (n *NormalizedTransferView) setTransInfoOwner(info *TransferInfo) {
	if n.IsTransfer {
		(&Transfer{ID: n.ID}).setTransInfoOwner(info)
	} else {
		(&HistoryEntry{ID: n.ID}).setTransInfoOwner(info)
	}
}
