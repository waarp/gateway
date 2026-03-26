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
	ebicsTransactionStatusPlanned    = "PLANNED"
	ebicsTransactionStatusRunning    = "RUNNING"
	ebicsTransactionStatusRecovering = "RECOVERING"
	ebicsTransactionStatusCompleted  = "COMPLETED"
	ebicsTransactionStatusFailed     = "FAILED"
	ebicsTransactionStatusCancelled  = "CANCELLED"
)

// EbicsTransaction stores the resumable EBICS transaction state attached to an operation.
type EbicsTransaction struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsOperationID  sql.NullInt64 `xorm:"ebics_operation_id"`
	EbicsHostID       int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID int64         `xorm:"ebics_subscriber_id"`

	TransactionID   string        `xorm:"transaction_id"`
	OrderType       string        `xorm:"order_type"`
	TransferID      sql.NullInt64 `xorm:"transfer_id"`
	Status          string        `xorm:"status"`
	Direction       string        `xorm:"direction"`
	SegmentCount    int           `xorm:"segment_count"`
	CurrentSegment  int           `xorm:"current_segment"`
	TotalSize       int64         `xorm:"total_size"`
	ResumedFromTxID sql.NullInt64 `xorm:"resumed_from_tx_id"`
	Metadata        string        `xorm:"metadata"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	MetadataMap map[string]any `xorm:"-"`
}

// TableName returns the persistent table name for EBICS transactions.
func (*EbicsTransaction) TableName() string { return TableEbicsTransactions }

// Appellation returns the display name used in validation messages.
func (*EbicsTransaction) Appellation() string { return NameEbicsTransaction }

// GetID returns the database identifier of the transaction.
func (t *EbicsTransaction) GetID() int64 { return t.ID }

// BeforeWrite normalizes and validates an EBICS transaction before persistence.
func (t *EbicsTransaction) BeforeWrite(db database.Access) error {
	t.Owner = conf.GlobalConfig.GatewayName
	t.TransactionID = strings.TrimSpace(t.TransactionID)
	t.OrderType = strings.ToUpper(strings.TrimSpace(t.OrderType))
	t.Status = strings.ToUpper(strings.TrimSpace(t.Status))
	t.Direction = strings.ToUpper(strings.TrimSpace(t.Direction))
	t.Metadata = strings.TrimSpace(t.Metadata)

	if t.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS host reference is missing")
	}

	if t.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS subscriber reference is missing")
	}

	if t.TransactionID == "" {
		return database.NewValidationError("the EBICS transaction ID is missing")
	}

	if err := validateEbicsPayloadOrderType(t.OrderType); err != nil {
		return err
	}

	if err := validateEbicsTransactionStatus(t.Status); err != nil {
		return err
	}

	if err := validateEbicsTransactionDirection(t.Direction); err != nil {
		return err
	}

	if err := validateEbicsTransactionCounters(t); err != nil {
		return err
	}

	if t.MetadataMap != nil {
		serialized, err := serializeStringMap(t.MetadataMap)
		if err != nil {
			return fmt.Errorf("failed to serialize EBICS transaction metadata: %w", err)
		}

		t.Metadata = serialized
	} else if t.Metadata == "" {
		t.Metadata = emptyJSONObject
	}

	meta, err := deserializeStringMap(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS transaction metadata: %w", err)
	}

	t.MetadataMap = meta

	if n, errCount := db.Count(t).Where(
		"id<>? AND owner=? AND ebics_subscriber_id=? AND transaction_id=?",
		t.ID, t.Owner, t.EbicsSubscriberID, t.TransactionID,
	).Run(); errCount != nil {
		return fmt.Errorf("failed to check duplicate EBICS transactions: %w", errCount)
	} else if n != 0 {
		return database.NewValidationErrorf(
			"an EBICS transaction already exists for transaction ID %q", t.TransactionID)
	}

	return nil
}

// AfterRead hydrates the transient metadata map after a database read.
func (t *EbicsTransaction) AfterRead(database.ReadAccess) error {
	meta, err := deserializeStringMap(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS transaction metadata after read: %w", err)
	}

	t.MetadataMap = meta

	return nil
}

// AfterInsert refreshes transient state after insertion.
func (t *EbicsTransaction) AfterInsert(db database.Access) error {
	return t.AfterRead(db)
}

// AfterUpdate refreshes transient state after update.
func (t *EbicsTransaction) AfterUpdate(db database.Access) error {
	return t.AfterRead(db)
}

func validateEbicsTransactionStatus(value string) error {
	switch value {
	case ebicsTransactionStatusPlanned, ebicsTransactionStatusRunning,
		ebicsTransactionStatusRecovering, ebicsTransactionStatusCompleted,
		ebicsTransactionStatusFailed, ebicsTransactionStatusCancelled:
		return nil
	case "":
		return database.NewValidationError("the EBICS transaction status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS transaction status", value)
	}
}

func validateEbicsTransactionDirection(value string) error {
	switch value {
	case ebicsOperationDirectionInbound, ebicsOperationDirectionOutbound:
		return nil
	case "":
		return database.NewValidationError("the EBICS transaction direction is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS transaction direction", value)
	}
}

func validateEbicsTransactionCounters(t *EbicsTransaction) error {
	if t.SegmentCount < 0 {
		return database.NewValidationError("the EBICS transaction segment count cannot be negative")
	}

	if t.CurrentSegment < 0 {
		return database.NewValidationError("the EBICS transaction current segment cannot be negative")
	}

	if t.CurrentSegment > t.SegmentCount && t.SegmentCount != 0 {
		return database.NewValidationError(
			"the EBICS transaction current segment cannot exceed the total segment count")
	}

	if t.TotalSize < 0 {
		return database.NewValidationError("the EBICS transaction total size cannot be negative")
	}

	return nil
}

// EbicsTransactionStatusCompletedForRuntime exposes the completed transaction status.
func EbicsTransactionStatusCompletedForRuntime() string {
	return ebicsTransactionStatusCompleted
}
