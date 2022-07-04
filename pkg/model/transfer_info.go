package model

import "code.waarp.fr/apps/gateway/gateway/pkg/database"

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&TransferInfo{})
}

// TransferInfo represents the transfer_info database table, which contains all the
// protocol-specific information attached to a transfer.
type TransferInfo struct {
	TransferID uint64 `xorm:"notnull unique(infoName) 'transfer_id'"`
	IsHistory  bool   `xorm:"notnull 'is_history'"`
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
func (t *TransferInfo) BeforeWrite(db database.ReadAccess) database.Error {
	var (
		n   uint64
		err error
	)

	if t.IsHistory {
		n, err = db.Count(&HistoryEntry{}).Where("id=?", t.TransferID).Run()
	} else {
		n, err = db.Count(&Transfer{}).Where("id=?", t.TransferID).Run()
	}

	if n == 0 {
		return database.NewValidationError("no transfer %d found", t.TransferID)
	}

	if err != nil {
		return database.NewValidationError("failed to retrieve transfer list: %s", err)
	}

	if n, err = db.Count(&TransferInfo{}).Where("transfer_id=? AND name=?",
		t.TransferID, t.Name).Run(); err != nil {
		return database.NewValidationError("failed to retrieve info list: %s", err)
	} else if n > 0 {
		return database.NewValidationError("transfer %d already has a property '%s'",
			t.TransferID, t.Name)
	}

	return nil
}

// ToMap converts and returns the TransferInfoList into an equivalent map.
func (t TransferInfoList) ToMap() map[string]string {
	infoMap := map[string]string{}
	for _, info := range t {
		infoMap[info.Name] = info.Value
	}

	return infoMap
}
