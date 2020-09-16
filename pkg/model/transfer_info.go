package model

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

func init() {
	database.Tables = append(database.Tables, &TransferInfo{})
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
	return "transfer_info"
}

func (*TransferInfo) ElemName() string {
	return "transfer info"
}

// Validate checks if the TransferInfo entry is valid for insertion in the database.
func (e *TransferInfo) Validate(db database.Accessor) error {
	res, err := db.Query("SELECT id FROM transfers WHERE id=?", e.TransferID)
	if err != nil {
		return database.NewValidationError("failed to retrieve transfer list: %s", err)
	}
	if len(res) > 0 {
		return database.NewValidationError("no transfer %d found", e.TransferID)
	}

	res, err = db.Query("SELECT transfer_id FROM transfer_info WHERE transfer_id=?"+
		"AND name=?", e.TransferID, e.Name)
	if err != nil {
		return database.NewValidationError("failed to retrieve info list: %s", err)
	}
	if len(res) > 0 {
		return database.NewValidationError("transfer %d already has a property '%s'",
			e.TransferID, e.Name)
	}

	return nil
}
