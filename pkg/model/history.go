package model

import (
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &TransferHistory{})
}

// TransferHistory represents one record of the 'transfers_history' table.
type TransferHistory struct {
	ID       uint64         `xorm:"pk 'id'" json:"id"`
	Owner    string         `xorm:"notnull" json:"-"`
	From     string         `xorm:"notnull" json:"from"`
	To       string         `xorm:"notnull" json:"to"`
	Protocol string         `xorm:"notnull" json:"protocol"`
	Filename string         `xorm:"notnull" json:"filename"`
	Rule     string         `xorm:"notnull" json:"rule"`
	Start    time.Time      `xorm:"notnull" json:"start"`
	Stop     time.Time      `xorm:"notnull" json:"stop"`
	Status   TransferStatus `xorm:"notnull" json:"status"`
}

// TableName returns the name of the transfer history table.
func (*TransferHistory) TableName() string {
	return "transfer_history"
}

// BeforeInsert is called before inserting the transfer in the database. Its
// role is to set the Owner, to force the Status and to set a Start time if none
// was entered.
func (t *TransferHistory) BeforeInsert(database.Accessor) error {
	t.Owner = database.Owner
	return nil
}

// ValidateInsert checks if the new `TransferHistory` entry is valid and can be
// inserted in the database.
func (t *TransferHistory) ValidateInsert(database.Accessor) error {
	if t.Owner == "" {
		return database.InvalidError("The transfer's owner cannot be empty")
	}
	if t.ID == 0 {
		return database.InvalidError("The transfer's ID cannot be empty")
	}
	if t.Rule == "" {
		return database.InvalidError("The transfer's rule cannot be empty")
	}
	if t.From == "" {
		return database.InvalidError("The transfer's source cannot be empty")
	}
	if t.To == "" {
		return database.InvalidError("The transfer's destination cannot be empty")
	}
	if t.Filename == "" {
		return database.InvalidError("The transfer's filename cannot be empty")
	}
	if t.Start.IsZero() {
		return database.InvalidError("The transfer's start date cannot be empty")
	}
	if t.Stop.IsZero() {
		return database.InvalidError("The transfer's end date cannot be empty")
	}

	if t.Stop.Before(t.Start) {
		return database.InvalidError("The transfer's end date cannot be anterior " +
			"to the start date")
	}

	if !IsValidProtocol(t.Protocol) {
		return database.InvalidError("'%s' is not a valid protocol", t.Protocol)
	}

	if !validateStatusForHistory(t.Status) {
		return database.InvalidError("'%s' is not a valid transfer history status", t.Status)
	}

	return nil
}

// ValidateUpdate is called before updating an existing `TransferHistory` entry
// from the database. It checks whether the updated entry is valid or not.
func (t *TransferHistory) ValidateUpdate(database.Accessor, uint64) error {
	if t.ID != 0 {
		return database.InvalidError("The transfer's ID cannot be changed")
	}
	if t.Owner != "" {
		return database.InvalidError("The transfer's owner cannot be changed")
	}
	if t.Rule != "" {
		return database.InvalidError("The transfer's rule cannot be changed")
	}
	if t.From != "" {
		return database.InvalidError("The transfer's source cannot be changed")
	}
	if t.To != "" {
		return database.InvalidError("The transfer's destination cannot be changed")
	}
	if t.Filename != "" {
		return database.InvalidError("The transfer's filename cannot be changed")
	}
	if t.Protocol != "" {
		return database.InvalidError("The transfer's protocol cannot be changed")
	}

	if t.Start.IsZero() {
		return database.InvalidError("The transfer's start cannot be empty")
	}
	if t.Stop.IsZero() {
		return database.InvalidError("The transfer's stop cannot be empty")
	}

	if t.Stop.Before(t.Start) {
		return database.InvalidError("The transfer's end date cannot be anterior " +
			"to the start date")
	}

	if !validateStatusForHistory(t.Status) {
		return database.InvalidError("'%s' is not a valid transfer history status", t.Status)
	}

	return nil
}
