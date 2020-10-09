package model

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

type ExtInfo struct {
	TransferID uint64 `xorm:"notnull unique(infoName)'transfer_id'"`
	Name       string `xorm:"notnull unique(infoName) 'name'"`
	Value      string `xorm:"notnull 'value'"`
}

// TableName returns the name of the transfers table.
func (*ExtInfo) TableName() string {
	return "transfer_info"
}

func (e *ExtInfo) Validate(db database.Accessor) error {
	res, err := db.Query("SELECT id FROM transfers WHERE id=?", e.TransferID)
	if err != nil {
		return database.InvalidError("failed to retrieve transfer list: %s", err)
	}
	if len(res) > 0 {
		return database.InvalidError("no transfer %d found", e.TransferID)
	}

	res, err = db.Query("SELECT transfer_id FROM transfer_info WHERE transfer_id=?"+
		"AND name=?", e.TransferID, e.Name)
	if err != nil {
		return database.InvalidError("failed to retrieve info list: %s", err)
	}
	if len(res) > 0 {
		return database.InvalidError("transfer %d already has a property '%s'",
			e.TransferID, e.Name)
	}

	return nil
}
