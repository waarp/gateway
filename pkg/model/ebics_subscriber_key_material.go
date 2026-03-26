package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsSubscriberKeyStateActive  = "ACTIVE"
	ebicsSubscriberKeyStateRetired = "RETIRED"
)

// EbicsSubscriberKeyMaterial stores the public EBICS key material known for one subscriber usage.
type EbicsSubscriberKeyMaterial struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsSubscriberID int64 `xorm:"ebics_subscriber_id"`

	KeyUsage           string `xorm:"key_usage"`
	PublicKey          string `xorm:"public_key"`
	PublicKeyVersion   string `xorm:"public_key_version"`
	Certificate        string `xorm:"certificate"`
	CertificateVersion string `xorm:"certificate_version"`
	State              string `xorm:"state"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

// TableName returns the persistent table name for EBICS subscriber key materials.
func (*EbicsSubscriberKeyMaterial) TableName() string { return TableEbicsSubscriberKeyMaterials }

// Appellation returns the display name used in validation messages.
func (*EbicsSubscriberKeyMaterial) Appellation() string { return NameEbicsSubscriberKeyMaterial }

// GetID returns the database identifier of the subscriber key material.
func (m *EbicsSubscriberKeyMaterial) GetID() int64 { return m.ID }

// BeforeWrite normalizes and validates EBICS subscriber key material before persistence.
func (m *EbicsSubscriberKeyMaterial) BeforeWrite(db database.Access) error {
	m.Owner = conf.GlobalConfig.GatewayName
	m.KeyUsage = strings.ToUpper(strings.TrimSpace(m.KeyUsage))
	m.PublicKey = strings.TrimSpace(m.PublicKey)
	m.PublicKeyVersion = strings.ToUpper(strings.TrimSpace(m.PublicKeyVersion))
	m.Certificate = strings.TrimSpace(m.Certificate)
	m.CertificateVersion = strings.ToUpper(strings.TrimSpace(m.CertificateVersion))
	m.State = strings.ToUpper(strings.TrimSpace(m.State))

	if m.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS subscriber key material reference is missing")
	}

	if err := validateEbicsKeyUsage(m.KeyUsage); err != nil {
		return err
	}

	if err := validateEbicsSubscriberKeyState(m.State); err != nil {
		return err
	}

	if m.PublicKey == "" && m.Certificate == "" {
		return database.NewValidationError("the EBICS subscriber key material requires a public key or a certificate")
	}

	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", m.EbicsSubscriberID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", m.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber for key material: %w", err)
	}

	if n, err := db.Count(m).Where(
		"id<>? AND owner=? AND ebics_subscriber_id=? AND key_usage=? AND state=?",
		m.ID, m.Owner, m.EbicsSubscriberID, m.KeyUsage, ebicsSubscriberKeyStateActive,
	).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate active EBICS subscriber key materials: %w", err)
	} else if n != 0 && m.State == ebicsSubscriberKeyStateActive {
		return database.NewValidationErrorf(
			"an active EBICS subscriber key material already exists for subscriber %d and usage %q",
			m.EbicsSubscriberID, m.KeyUsage)
	}

	return nil
}

func validateEbicsSubscriberKeyState(value string) error {
	switch value {
	case ebicsSubscriberKeyStateActive, ebicsSubscriberKeyStateRetired:
		return nil
	case "":
		return database.NewValidationError("the EBICS subscriber key material state is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS subscriber key material state", value)
	}
}
