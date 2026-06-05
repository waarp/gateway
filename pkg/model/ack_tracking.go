package model

import (
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// AckState represents the state of an ACK tracking entry.
type AckState string

const (
	AckStateRequested AckState = "REQUESTED"
	AckStateSent      AckState = "SENT"
	AckStateReceived  AckState = "RECEIVED"
)

// AckTracking represents an entry in the ack_tracking table. Each transfer
// has at most one ack_tracking entry (UNIQUE on transfer_id). The table is
// protocol-agnostic and can be used for PeSIT F.MESSAGE, AS2 MDN, OFTP2 EERP, etc.
type AckTracking struct {
	ID         int64    `xorm:"<- id AUTOINCR"`
	TransferID int64    `xorm:"transfer_id"`
	RemoteID   string   `xorm:"remote_id"`
	IsSend     bool     `xorm:"is_send"`
	State      AckState `xorm:"state"`
	Partner    string   `xorm:"partner"`
	Account    string   `xorm:"account"`
	Origin     string   `xorm:"origin"`
	Message    string   `xorm:"message"`
	CustomerID string   `xorm:"customer_id"`
	BankID     string   `xorm:"bank_id"`
	CreatedAt  time.Time `xorm:"created_at DATETIME(6) UTC"`
	UpdatedAt  time.Time `xorm:"updated_at DATETIME(6) UTC"`
}

func (*AckTracking) TableName() string   { return TableAckTracking }
func (*AckTracking) Appellation() string { return NameAckTracking }
func (a *AckTracking) GetID() int64      { return a.ID }

// BeforeWrite validates the AckTracking entry before inserting or updating.
func (a *AckTracking) BeforeWrite(database.Access) error {
	if a.TransferID == 0 {
		return database.NewValidationError("the ack tracking entry is missing a transfer ID")
	}

	switch a.State {
	case AckStateRequested, AckStateSent, AckStateReceived:
		// valid
	default:
		return database.NewValidationErrorf("invalid ack tracking state: %q", a.State)
	}

	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}

	a.UpdatedAt = now

	return nil
}

// InsertAckTracking atomically inserts an ack_tracking entry.
func InsertAckTracking(db database.Access, ack *AckTracking) error {
	if err := db.Insert(ack).Run(); err != nil {
		return fmt.Errorf("failed to insert ack tracking entry: %w", err)
	}

	return nil
}

// UpdateAckReceived atomically updates an ack_tracking entry to RECEIVED state.
// Uses a direct SQL UPDATE for atomicity (no read-modify-write cycle).
func UpdateAckReceived(db *database.DB, transferID int64,
	message, customerID, bankID, origin string,
) error {
	now := time.Now().UTC()
	if err := db.Exec(
		`UPDATE ack_tracking SET state=?, message=?, customer_id=?, bank_id=?, origin=?, updated_at=? WHERE transfer_id=?`,
		string(AckStateReceived), message, customerID, bankID, origin, now, transferID,
	); err != nil {
		return fmt.Errorf("failed to update ack tracking to RECEIVED: %w", err)
	}

	return nil
}

// GetAckTracking retrieves the ack_tracking entry for a given transfer.
// Returns nil if no entry exists.
func GetAckTracking(db database.ReadAccess, transferID int64) *AckTracking {
	var ack AckTracking
	if err := db.Get(&ack, "transfer_id=?", transferID).Run(); err != nil {
		return nil
	}

	return &ack
}
