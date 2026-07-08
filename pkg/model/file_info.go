//go:build removed_file_info
// +build removed_file_info

package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// FileInfo represents the file_info database table, which contains all the
// protocol-specific information attached to a transfer.
type FileInfo struct {
	TransferID uint64 `gorm:"column:notnull unique(infoName) 'transfer_id'"`
	IsHistory  bool   `gorm:"column:notnull 'is_history'"`
	Name       string `gorm:"column:notnull unique(infoName) 'name'"`
	Value      string `gorm:"column:notnull 'value'"`
}

// TableName returns the name of the transfers table.
func (*FileInfo) TableName() string {
	return TableFileInfo
}

// Appellation returns the display name of a transfer info entry.
func (*FileInfo) Appellation() string {
	return "transfer info"
}

// BeforeWrite checks if the FileInfo entry is valid for insertion in the database.
func (t *FileInfo) BeforeWrite(db database.ReadAccess) error {
	n, err := db.Count(&Transfer{}).Where("id=?", t.TransferID).Run()
	if err != nil {
		return fmt.Errorf("failed to retrieve transfer list: %w", err)
	}
	if n == 0 {
		return database.NewValidationErrorf("no transfer %d found", t.TransferID)
	}

	n, err = db.Count(&FileInfo{}).Where("transfer_id=? AND name=?", t.TransferID, t.Name).Run()
	if err != nil {
		return fmt.Errorf("failed to retrieve info list: %w", err)
	}
	if n > 0 {
		return database.NewValidationErrorf("transfer %d already has a property %q",
			t.TransferID, t.Name)
	}

	return nil
}

// ToMap converts and returns the FileInfoList into an equivalent map.
func (t FileInfoList) ToMap() map[string]string {
	infoMap := map[string]string{}
	for _, info := range t {
		infoMap[info.Name] = info.Value
	}
	return infoMap
}
