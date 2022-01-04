package model

import (
	"database/sql"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type transferInfoOwner interface {
	getTransInfoCondition() (string, int64)
	setTransInfoOwner(*TransferInfo)
}

// TransferInfo represents the transfer_info database table, which contains all the
// protocol-specific information attached to a transfer.
type TransferInfo struct {
	// The owner of the info pair. Only one can be valid at a time.
	TransferID sql.NullInt64 `xorm:"transfer_id"`
	HistoryID  sql.NullInt64 `xorm:"history_id"`

	Name  string `xorm:"name"`  // The info's key.
	Value string `xorm:"value"` // The info's value.
}

func (*TransferInfo) TableName() string   { return TableTransferInfo }
func (*TransferInfo) Appellation() string { return "transfer info" }
func (t *TransferInfo) IsHistory() bool   { return t.HistoryID.Valid }

// BeforeWrite checks if the TransferInfo entry is valid for insertion in the database.
func (t *TransferInfo) BeforeWrite(db database.ReadAccess) database.Error {
	var (
		owner   database.IterateBean
		transID int64
	)

	switch {
	case t.TransferID.Valid && t.HistoryID.Valid:
		return database.NewValidationError("the transfer info cannot belong to " +
			"both a transfer and a history entry")
	case t.TransferID.Valid:
		owner = &Transfer{}
		transID = t.TransferID.Int64
	case t.HistoryID.Valid:
		owner = &HistoryEntry{}
		transID = t.HistoryID.Int64
	default:
		return database.NewValidationError("the transfer info is missing a transfer")
	}

	n, err := db.Count(owner).Where("id=?", transID).Run()
	if err != nil {
		return database.NewValidationError("failed to retrieve transfer list: %s", err)
	} else if n == 0 {
		return database.NewValidationError("no transfer %d found", transID)
	}

	if n, err = db.Count(&TransferInfo{}).Where("transfer_id=? AND name=?",
		transID, t.Name).Run(); err != nil {
		return database.NewValidationError("failed to retrieve info list: %s", err)
	} else if n > 0 {
		return database.NewValidationError("transfer %d already has a property '%s'",
			transID, t.Name)
	}

	return nil
}
