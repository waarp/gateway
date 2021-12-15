package model

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// TransferInfo represents the transfer_info database table, which contains all the
// protocol-specific information attached to a transfer.
type TransferInfo struct {
	TransferID sql.NullInt64 `xorm:"BIGINT UNIQUE(tranInfo) 'transfer_id'"`
	HistoryID  sql.NullInt64 `xorm:"BIGINT UNIQUE(histInfo) 'history_id'"`
	Name       string        `xorm:"VARCHAR(100) NOTNULL UNIQUE(tranInfo) UNIQUE(histInfo) 'name'"`
	Value      string        `xorm:"TEXT NOTNULL DEFAULT('null') 'value'"`
}

// TableName returns the name of the transfers table.
func (*TransferInfo) TableName() string {
	return TableTransferInfo
}

// Appellation returns the display name of a transfer info entry.
func (*TransferInfo) Appellation() string {
	return "transfer info"
}

func (t *TransferInfo) IsHistory() bool {
	return t.HistoryID.Valid
}

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

func (*TransferInfo) MakeExtraConstraints(db *database.Executor) database.Error {
	// add a foreign key to 'transfer_id'
	if err := redefineColumn(db, TableTransferInfo, "transfer_id", fmt.Sprintf(
		`BIGINT REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE CASCADE`, TableTransfers)); err != nil {
		return err
	}

	// add a foreign key to 'history_id'
	if err := redefineColumn(db, TableTransferInfo, "history_id", fmt.Sprintf(
		`BIGINT REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE CASCADE`, TableHistory)); err != nil {
		return err
	}

	// add a constraint to enforce that one (and only one) of 'transfer_id'
	// and 'history_id' must be defined
	return addTableConstraint(db, TableTransferInfo, utils.CheckOnlyOneNotNull(
		db.Dialect, "transfer_id", "history_id"))
}
