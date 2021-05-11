package model

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

func init() {
	database.AddTable(&TransferInfo{})
}

// TransferInfo represents the transfer_info database table, which contains all the
// protocol-specific information attached to a transfer.
type TransferInfo struct {
	TransferID uint64 `xorm:"notnull unique(infoName) 'transfer_id'"`
	Name       string `xorm:"notnull unique(infoName) 'name'"`
	Value      string `xorm:"notnull 'value'"`
}

// TableName returns the name of the transfers table.
func (*TransferInfo) TableName() string {
	return TableTransferInfo
}

// Appellation returns the display name of a transfer info entry.
func (*TransferInfo) Appellation() string {
	return "transfer info"
}

// BeforeWrite checks if the TransferInfo entry is valid for insertion in the database.
func (e *TransferInfo) BeforeWrite(db database.ReadAccess) database.Error {
	n, err := db.Count(&Transfer{}).Where("id=?", e.TransferID).Run()
	if err != nil {
		return database.NewValidationError("failed to retrieve transfer list: %s", err)
	}
	if n > 0 {
		return database.NewValidationError("no transfer %d found", e.TransferID)
	}

	n, err = db.Count(&TransferInfo{}).Where("transfer_id=? AND name=?", e.TransferID, e.Name).Run()
	if err != nil {
		return database.NewValidationError("failed to retrieve info list: %s", err)
	}
	if n > 0 {
		return database.NewValidationError("transfer %d already has a property '%s'",
			e.TransferID, e.Name)
	}

	return nil
}
