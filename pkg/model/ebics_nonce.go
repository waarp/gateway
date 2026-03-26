package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// EbicsNonce stores one durable anti-replay nonce for an EBICS subscriber.
type EbicsNonce struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsSubscriberID int64     `xorm:"ebics_subscriber_id"`
	Nonce             string    `xorm:"nonce"`
	Timestamp         time.Time `xorm:"timestamp DATETIME(6) UTC"`
	ExpiresAt         time.Time `xorm:"expires_at DATETIME(6) UTC"`
	CreatedAt         time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt         time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

// TableName returns the persistent table name for EBICS nonces.
func (*EbicsNonce) TableName() string { return TableEbicsNonces }

// Appellation returns the display name used in validation messages.
func (*EbicsNonce) Appellation() string { return NameEbicsNonce }

// GetID returns the database identifier of the nonce row.
func (n *EbicsNonce) GetID() int64 { return n.ID }

// BeforeWrite normalizes and validates an EBICS nonce before persistence.
func (n *EbicsNonce) BeforeWrite(db database.Access) error {
	n.Owner = conf.GlobalConfig.GatewayName
	n.Nonce = strings.TrimSpace(n.Nonce)

	if n.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS nonce subscriber reference is missing")
	}

	if n.Nonce == "" {
		return database.NewValidationError("the EBICS nonce value is missing")
	}

	if n.Timestamp.IsZero() {
		return database.NewValidationError("the EBICS nonce timestamp is missing")
	}

	if n.ExpiresAt.IsZero() {
		return database.NewValidationError("the EBICS nonce expiration timestamp is missing")
	}

	if !n.ExpiresAt.After(n.Timestamp) {
		return database.NewValidationError("the EBICS nonce expiration timestamp must be after the nonce timestamp")
	}

	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", n.EbicsSubscriberID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", n.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber for nonce: %w", err)
	}

	if count, err := db.Count(n).Where(
		"id<>? AND owner=? AND ebics_subscriber_id=? AND nonce=?",
		n.ID, n.Owner, n.EbicsSubscriberID, n.Nonce,
	).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS nonces: %w", err)
	} else if count != 0 {
		return database.NewValidationError("this EBICS nonce already exists for the selected subscriber")
	}

	return nil
}
