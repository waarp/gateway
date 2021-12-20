package model

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type NormalizedTransferView struct {
	HistoryEntry `xorm:"extends"`
	IsTransfer   bool `xorm:"BOOL 'is_transfer'"`
}

func (*NormalizedTransferView) TableName() string   { return ViewNormalizedTransfers }
func (*NormalizedTransferView) Appellation() string { return "normalized transfer" }
func (n *NormalizedTransferView) GetID() int64      { return n.ID }

// BeforeWrite always returns an error because writing is not allowed on views.
func (n *NormalizedTransferView) BeforeWrite(database.ReadAccess) database.Error {
	return database.NewInternalError(errWriteOnView)
}

// BeforeDelete always returns an error because deleting is not allowed on views.
func (n *NormalizedTransferView) BeforeDelete(database.Access) database.Error {
	return database.NewInternalError(errWriteOnView)
}

// GetTransferInfo returns the list of the transfer's TransferInfo as a map of interfaces.
func (n *NormalizedTransferView) GetTransferInfo(db database.ReadAccess) (map[string]interface{}, database.Error) {
	return getTransferInfo(db, n.ID, n.IsTransfer)
}
