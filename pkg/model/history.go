package model

import (
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

func init() {
	database.Tables = append(database.Tables, &TransferHistory{})
}

// TransferHistory represents one record of the 'transfers_history' table.
type TransferHistory struct {
	ID             uint64         `xorm:"pk autoincr <- 'id'"`
	Owner          string         `xorm:"notnull 'owner'"`
	IsServer       bool           `xorm:"notnull 'is_server'"`
	IsSend         bool           `xorm:"notnull 'is_send'"`
	Account        string         `xorm:"notnull 'account'"`
	Remote         string         `xorm:"notnull 'remote'"`
	Protocol       string         `xorm:"notnull 'protocol'"`
	SourceFilename string         `xorm:"notnull 'source_filename'"`
	DestFilename   string         `xorm:"notnull 'dest_filename'"`
	Rule           string         `xorm:"notnull 'rule'"`
	Start          time.Time      `xorm:"notnull 'start'"`
	Stop           time.Time      `xorm:"notnull 'stop'"`
	Status         TransferStatus `xorm:"notnull 'status'"`
	Error          TransferError  `xorm:"extends"`
	ExtInfo        []byte         `xorm:"ext_info" json:"extInfo"`
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

	if _, ok := config.ProtoConfigs[t.Protocol]; !ok {
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
