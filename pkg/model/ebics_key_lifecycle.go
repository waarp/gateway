package model

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsKeyUsageAuthentication = "AUTHENTICATION"
	ebicsKeyUsageEncryption     = "ENCRYPTION"
	ebicsKeyUsageSignature      = "SIGNATURE"

	ebicsRotationTypeInitialization = "INITIALIZATION"
	ebicsRotationTypeRotation       = "ROTATION"
	ebicsRotationTypeReplacement    = "REPLACEMENT"

	ebicsKeyLifecycleStatusDraft                   = "DRAFT"
	ebicsKeyLifecycleStatusMaterialPrepared        = "MATERIAL_PREPARED"
	ebicsKeyLifecycleStatusOrderPlanned            = "ORDER_PLANNED"
	ebicsKeyLifecycleStatusOrderSent               = "ORDER_SENT"
	ebicsKeyLifecycleStatusWaitingBankConfirmation = "WAITING_BANK_CONFIRMATION"
	ebicsKeyLifecycleStatusActivated               = "ACTIVATED"
	ebicsKeyLifecycleStatusRetired                 = "RETIRED"
	ebicsKeyLifecycleStatusCancelled               = "CANCELLED"
	ebicsKeyLifecycleStatusRejected                = "REJECTED"
)

// EbicsKeyLifecycle stores the technical lifecycle of an EBICS key rotation.
type EbicsKeyLifecycle struct {
	ID                int64  `xorm:"<- id AUTOINCR"`
	Owner             string `xorm:"owner"`
	EbicsSubscriberID int64  `xorm:"ebics_subscriber_id"`

	KeyUsage            string        `xorm:"key_usage"`
	RotationType        string        `xorm:"rotation_type"`
	Status              string        `xorm:"status"`
	CurrentCredentialID int64         `xorm:"current_credential_id"`
	NextCredentialID    sql.NullInt64 `xorm:"next_credential_id"`
	TriggerOperationID  sql.NullInt64 `xorm:"trigger_operation_id"`
	LastOperationID     sql.NullInt64 `xorm:"last_operation_id"`
	RequestedAt         time.Time     `xorm:"requested_at DATETIME(6) UTC"`
	SentAt              time.Time     `xorm:"sent_at DATETIME(6) UTC"`
	ActivatedAt         time.Time     `xorm:"activated_at DATETIME(6) UTC"`
	RetiredAt           time.Time     `xorm:"retired_at DATETIME(6) UTC"`
	Operator            string        `xorm:"operator"`
	Reason              string        `xorm:"reason"`
	Evidence            string        `xorm:"evidence"`
	CreatedAt           time.Time     `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt           time.Time     `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	EvidenceMap map[string]any `xorm:"-"`
}

// TableName returns the persistent table name for EBICS key lifecycles.
func (*EbicsKeyLifecycle) TableName() string { return TableEbicsKeyLifecycles }

// Appellation returns the display name used in validation messages.
func (*EbicsKeyLifecycle) Appellation() string { return NameEbicsKeyLifecycle }

// GetID returns the database identifier of the lifecycle.
func (l *EbicsKeyLifecycle) GetID() int64 { return l.ID }

// BeforeWrite normalizes and validates an EBICS key lifecycle before persistence.
func (l *EbicsKeyLifecycle) BeforeWrite(db database.Access) error {
	l.Owner = conf.GlobalConfig.GatewayName
	l.KeyUsage = strings.ToUpper(strings.TrimSpace(l.KeyUsage))
	l.RotationType = strings.ToUpper(strings.TrimSpace(l.RotationType))
	l.Status = strings.ToUpper(strings.TrimSpace(l.Status))
	l.Operator = strings.TrimSpace(l.Operator)
	l.Reason = strings.TrimSpace(l.Reason)
	l.Evidence = strings.TrimSpace(l.Evidence)

	if l.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS subscriber reference is missing")
	}

	if err := validateEbicsKeyUsage(l.KeyUsage); err != nil {
		return err
	}

	if err := validateEbicsRotationType(l.RotationType); err != nil {
		return err
	}

	if err := validateEbicsKeyLifecycleStatus(l.Status); err != nil {
		return err
	}

	if l.CurrentCredentialID == 0 {
		return database.NewValidationError("the current credential reference is missing")
	}

	if err := l.hydrateEvidence(); err != nil {
		return err
	}

	if err := validateEbicsKeyLifecycleBinding(l); err != nil {
		return err
	}

	if err := validateEbicsKeyLifecycleRefs(db, l); err != nil {
		return err
	}

	return validateUniqueActiveEbicsKeyLifecycle(db, l)
}

func (l *EbicsKeyLifecycle) hydrateEvidence() error {
	if l.EvidenceMap != nil {
		serialized, err := serializeStringMap(l.EvidenceMap)
		if err != nil {
			return fmt.Errorf("failed to serialize EBICS key lifecycle evidence: %w", err)
		}

		l.Evidence = serialized
	} else if l.Evidence == "" {
		l.Evidence = emptyJSONObject
	}

	evidence, err := deserializeStringMap(l.Evidence)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS key lifecycle evidence: %w", err)
	}

	l.EvidenceMap = evidence

	return nil
}

// AfterRead hydrates the transient evidence map after a database read.
func (l *EbicsKeyLifecycle) AfterRead(database.ReadAccess) error {
	evidence, err := deserializeStringMap(l.Evidence)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS key lifecycle evidence after read: %w", err)
	}

	l.EvidenceMap = evidence

	return nil
}

// AfterInsert refreshes transient state after insertion.
func (l *EbicsKeyLifecycle) AfterInsert(db database.Access) error { return l.AfterRead(db) }

// AfterUpdate refreshes transient state after update.
func (l *EbicsKeyLifecycle) AfterUpdate(db database.Access) error { return l.AfterRead(db) }

func validateEbicsKeyUsage(value string) error {
	switch value {
	case ebicsKeyUsageAuthentication, ebicsKeyUsageEncryption, ebicsKeyUsageSignature:
		return nil
	case "":
		return database.NewValidationError("the EBICS key usage is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS key usage", value)
	}
}

func validateEbicsRotationType(value string) error {
	switch value {
	case ebicsRotationTypeInitialization, ebicsRotationTypeRotation, ebicsRotationTypeReplacement:
		return nil
	case "":
		return database.NewValidationError("the EBICS rotation type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS rotation type", value)
	}
}

func validateEbicsKeyLifecycleStatus(value string) error {
	switch value {
	case ebicsKeyLifecycleStatusDraft, ebicsKeyLifecycleStatusMaterialPrepared,
		ebicsKeyLifecycleStatusOrderPlanned, ebicsKeyLifecycleStatusOrderSent,
		ebicsKeyLifecycleStatusWaitingBankConfirmation, ebicsKeyLifecycleStatusActivated,
		ebicsKeyLifecycleStatusRetired, ebicsKeyLifecycleStatusCancelled,
		ebicsKeyLifecycleStatusRejected:
		return nil
	case "":
		return database.NewValidationError("the EBICS key lifecycle status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS key lifecycle status", value)
	}
}

func validateEbicsKeyLifecycleBinding(lifecycle *EbicsKeyLifecycle) error {
	if requiresNextCredential(lifecycle.Status) && !lifecycle.NextCredentialID.Valid {
		return database.NewValidationError(
			"the next credential reference is required for the selected EBICS key lifecycle status")
	}

	if lifecycle.NextCredentialID.Valid && lifecycle.CurrentCredentialID == lifecycle.NextCredentialID.Int64 {
		return database.NewValidationError(
			"the current and next credentials cannot be identical in an EBICS key lifecycle")
	}

	if lifecycle.Status == ebicsKeyLifecycleStatusActivated && lifecycle.ActivatedAt.IsZero() {
		return database.NewValidationError(
			"an activated EBICS key lifecycle requires an activation timestamp")
	}

	if lifecycle.Status == ebicsKeyLifecycleStatusRetired {
		if lifecycle.ActivatedAt.IsZero() {
			return database.NewValidationError(
				"a retired EBICS key lifecycle requires a prior activation timestamp")
		}

		if lifecycle.RetiredAt.IsZero() {
			return database.NewValidationError(
				"a retired EBICS key lifecycle requires a retirement timestamp")
		}
	}

	if !lifecycle.SentAt.IsZero() && lifecycle.RequestedAt.IsZero() {
		return database.NewValidationError(
			"an EBICS key lifecycle cannot have a sent timestamp without a request timestamp")
	}

	if !lifecycle.ActivatedAt.IsZero() && lifecycle.SentAt.IsZero() {
		return database.NewValidationError(
			"an EBICS key lifecycle cannot be activated before the rotation order was sent")
	}

	if !lifecycle.RetiredAt.IsZero() && lifecycle.ActivatedAt.IsZero() {
		return database.NewValidationError(
			"an EBICS key lifecycle cannot be retired before being activated")
	}

	return nil
}

func validateEbicsKeyLifecycleRefs(db database.Access, lifecycle *EbicsKeyLifecycle) error {
	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", lifecycle.EbicsSubscriberID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", lifecycle.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber for key lifecycle: %w", err)
	}

	if err := validateCredentialExists(db, lifecycle.CurrentCredentialID, "current"); err != nil {
		return err
	}

	if lifecycle.NextCredentialID.Valid {
		if err := validateCredentialExists(db, lifecycle.NextCredentialID.Int64, "next"); err != nil {
			return err
		}
	}

	if lifecycle.TriggerOperationID.Valid {
		if err := validateEbicsOperationExists(db, lifecycle.TriggerOperationID.Int64, "trigger"); err != nil {
			return err
		}
	}

	if lifecycle.LastOperationID.Valid {
		if err := validateEbicsOperationExists(db, lifecycle.LastOperationID.Int64, "last"); err != nil {
			return err
		}
	}

	return nil
}

func validateUniqueActiveEbicsKeyLifecycle(db database.Access, lifecycle *EbicsKeyLifecycle) error {
	if !isActiveEbicsKeyLifecycleStatus(lifecycle.Status) {
		return nil
	}

	count, err := db.Count(lifecycle).Where(
		"id<>? AND owner=? AND ebics_subscriber_id=? AND key_usage=? AND status NOT IN (?, ?, ?)",
		lifecycle.ID,
		lifecycle.Owner,
		lifecycle.EbicsSubscriberID,
		lifecycle.KeyUsage,
		ebicsKeyLifecycleStatusRetired,
		ebicsKeyLifecycleStatusCancelled,
		ebicsKeyLifecycleStatusRejected,
	).Run()
	if err != nil {
		return fmt.Errorf("failed to check duplicate active EBICS key lifecycles: %w", err)
	}

	if count != 0 {
		return database.NewValidationErrorf(
			"an active EBICS key lifecycle already exists for subscriber %d and usage %q",
			lifecycle.EbicsSubscriberID, lifecycle.KeyUsage)
	}

	return nil
}

func requiresNextCredential(status string) bool {
	switch status {
	case ebicsKeyLifecycleStatusMaterialPrepared, ebicsKeyLifecycleStatusOrderPlanned,
		ebicsKeyLifecycleStatusOrderSent, ebicsKeyLifecycleStatusWaitingBankConfirmation,
		ebicsKeyLifecycleStatusActivated, ebicsKeyLifecycleStatusRetired, ebicsKeyLifecycleStatusRejected:
		return true
	default:
		return false
	}
}

func isActiveEbicsKeyLifecycleStatus(status string) bool {
	switch status {
	case ebicsKeyLifecycleStatusRetired, ebicsKeyLifecycleStatusCancelled, ebicsKeyLifecycleStatusRejected:
		return false
	default:
		return true
	}
}
