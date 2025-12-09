package model

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
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

type transferInfoOwner interface {
	getTransInfoCondition() (string, int64)
	setTransInfoOwner(info *TransferInfo)
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
func (*TransferInfo) Appellation() string { return NameTransferInfo }
func (t *TransferInfo) IsHistory() bool   { return t.HistoryID.Valid }

// BeforeWrite checks if the TransferInfo entry is valid for insertion in the database.
func (t *TransferInfo) BeforeWrite(db database.Access) error {
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
		return fmt.Errorf("failed to retrieve transfer list: %w", err)
	} else if n == 0 {
		return database.NewValidationErrorf("no transfer %d found", transID)
	}

	if n, err = db.Count(&TransferInfo{}).Where("transfer_id=? AND name=?",
		transID, t.Name).Run(); err != nil {
		return fmt.Errorf("failed to retrieve info list: %w", err)
	} else if n > 0 {
		return database.NewValidationErrorf("transfer %d already has a property %q",
			transID, t.Name)
	}

	return nil
}
