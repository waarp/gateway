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
	ID             uint64         `xorm:"pk 'id'" json:"id"`
	Owner          string         `xorm:"notnull 'owner'" json:"-"`
	IsServer       bool           `xorm:"notnull 'is_server'" json:"isServer"`
	IsSend         bool           `xorm:"notnull 'is_send'" json:"isSend"`
	Account        string         `xorm:"notnull 'account'" json:"account"`
	Remote         string         `xorm:"notnull 'remote'" json:"remote"`
	Protocol       string         `xorm:"notnull 'protocol'" json:"protocol"`
	SourceFilename string         `xorm:"notnull 'source_filename'" json:"sourceFilename"`
	DestFilename   string         `xorm:"notnull 'dest_filename'" json:"destFilename"`
	Rule           string         `xorm:"notnull 'rule'" json:"rule"`
	Start          time.Time      `xorm:"notnull 'start'" json:"start"`
	Stop           time.Time      `xorm:"notnull 'stop'" json:"stop"`
	Status         TransferStatus `xorm:"notnull 'status'" json:"status"`
	ErrorCode      ReadOnlyByte   `xorm:"notnull 'error_code'" json:"errorCode"`
	ErrorMsg       ReadOnlyString `xorm:"notnull 'error_msg'" json:"errorMsg"`
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
	if t.Account == "" {
		return database.InvalidError("The transfer's account cannot be empty")
	}
	if t.Remote == "" {
		return database.InvalidError("The transfer's remote cannot be empty")
	}
	if t.ErrorCode != 0 {
		return database.InvalidError("The transfer's error code must be empty")
	}
	if t.ErrorMsg != "" {
		return database.InvalidError("The transfer's error message must be empty")
	}
	if t.IsServer {
		if t.IsSend && t.DestFilename == "" {
			return database.InvalidError("The transfer's destination filename cannot be empty")
		} else if !t.IsSend && t.SourceFilename == "" {
			return database.InvalidError("The transfer's destination filename cannot be empty")
		}
	} else {
		if t.SourceFilename == "" {
			return database.InvalidError("The transfer's source filename cannot be empty")
		}
		if t.DestFilename == "" {
			return database.InvalidError("The transfer's destination filename cannot be empty")
		}
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
	if t.Account != "" {
		return database.InvalidError("The transfer's account cannot be changed")
	}
	if t.Remote != "" {
		return database.InvalidError("The transfer's remote cannot be changed")
	}
	if t.SourceFilename != "" {
		return database.InvalidError("The transfer's source filename cannot be changed")
	}
	if t.DestFilename != "" {
		return database.InvalidError("The transfer's destination filename cannot be changed")
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
