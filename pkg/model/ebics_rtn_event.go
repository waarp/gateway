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
	ebicsRTNEventStatusReceived    = "RECEIVED"
	ebicsRTNEventStatusDuplicate   = "DUPLICATE"
	ebicsRTNEventStatusProcessing  = "PROCESSING"
	ebicsRTNEventStatusProcessed   = "PROCESSED"
	ebicsRTNEventStatusRetryable   = "RETRYABLE"
	ebicsRTNEventStatusQuarantined = "QUARANTINED"
	ebicsRTNEventStatusFailed      = "FAILED"
)

// EbicsRTNEvent stores a normalized RTN event with durable idempotence information.
type EbicsRTNEvent struct {
	ID                int64         `xorm:"<- id AUTOINCR"`
	Owner             string        `xorm:"owner"`
	Source            string        `xorm:"source"`
	EventID           string        `xorm:"event_id"`
	CorrelationID     string        `xorm:"correlation_id"`
	IdempotenceKey    string        `xorm:"idempotence_key"`
	EbicsHostID       sql.NullInt64 `xorm:"ebics_host_id"`
	EbicsSubscriberID sql.NullInt64 `xorm:"ebics_subscriber_id"`
	OrderTypeHint     string        `xorm:"order_type_hint"`
	ProfileID         string        `xorm:"profile_id"`
	Payload           string        `xorm:"payload"`
	Status            string        `xorm:"status"`
	Attempts          int           `xorm:"attempts"`
	NextRetryAt       time.Time     `xorm:"next_retry_at DATETIME(6) UTC"`
	ReceivedAt        time.Time     `xorm:"received_at DATETIME(6) UTC"`
	ProcessedAt       time.Time     `xorm:"processed_at DATETIME(6) UTC"`
	LastError         string        `xorm:"last_error"`
	CreatedAt         time.Time     `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt         time.Time     `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	PayloadMap map[string]any `xorm:"-"`
}

// TableName returns the persistent table name for RTN events.
func (*EbicsRTNEvent) TableName() string { return TableEbicsRTNEvents }

// Appellation returns the display name used in validation messages.
func (*EbicsRTNEvent) Appellation() string { return NameEbicsRTNEvent }

// GetID returns the database identifier of the event.
func (e *EbicsRTNEvent) GetID() int64 { return e.ID }

// BeforeWrite normalizes and validates an RTN event before persistence.
func (e *EbicsRTNEvent) BeforeWrite(db database.Access) error {
	e.Owner = conf.GlobalConfig.GatewayName
	e.Source = strings.TrimSpace(e.Source)
	e.EventID = strings.TrimSpace(e.EventID)
	e.CorrelationID = strings.TrimSpace(e.CorrelationID)
	e.IdempotenceKey = strings.TrimSpace(e.IdempotenceKey)
	e.OrderTypeHint = strings.ToUpper(strings.TrimSpace(e.OrderTypeHint))
	e.ProfileID = strings.TrimSpace(e.ProfileID)
	e.Payload = strings.TrimSpace(e.Payload)
	e.Status = strings.ToUpper(strings.TrimSpace(e.Status))
	e.LastError = strings.TrimSpace(e.LastError)

	if err := validateEbicsRTNEventSource(e.Source); err != nil {
		return err
	}

	if e.IdempotenceKey == "" {
		return database.NewValidationError("the RTN event idempotence key is missing")
	}

	if err := validateEbicsRTNEventStatus(e.Status); err != nil {
		return err
	}

	if e.Attempts < 0 {
		return database.NewValidationError("the RTN event attempt counter cannot be negative")
	}

	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now().UTC()
	}

	if err := e.hydratePayload(); err != nil {
		return err
	}

	if err := validateEbicsRTNEventCoherence(e); err != nil {
		return err
	}

	if err := validateEbicsRTNEventRefs(db, e); err != nil {
		return err
	}

	return validateEbicsRTNEventUniqueness(db, e)
}

func (e *EbicsRTNEvent) hydratePayload() error {
	if e.PayloadMap != nil {
		serialized, err := serializeStringMap(e.PayloadMap)
		if err != nil {
			return fmt.Errorf("failed to serialize EBICS RTN payload: %w", err)
		}

		e.Payload = serialized
	} else if e.Payload == "" {
		e.Payload = emptyJSONObject
	}

	payload, err := deserializeStringMap(e.Payload)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS RTN payload: %w", err)
	}

	e.PayloadMap = payload

	return nil
}

// AfterRead hydrates the transient payload map after a database read.
func (e *EbicsRTNEvent) AfterRead(database.ReadAccess) error {
	payload, err := deserializeStringMap(e.Payload)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS RTN payload after read: %w", err)
	}

	e.PayloadMap = payload

	return nil
}

// AfterInsert refreshes transient state after insertion.
func (e *EbicsRTNEvent) AfterInsert(db database.Access) error { return e.AfterRead(db) }

// AfterUpdate refreshes transient state after update.
func (e *EbicsRTNEvent) AfterUpdate(db database.Access) error { return e.AfterRead(db) }

func validateEbicsRTNEventSource(value string) error {
	if value == "" {
		return database.NewValidationError("the RTN event source is missing")
	}

	return nil
}

func validateEbicsRTNEventStatus(value string) error {
	switch value {
	case ebicsRTNEventStatusReceived, ebicsRTNEventStatusDuplicate,
		ebicsRTNEventStatusProcessing, ebicsRTNEventStatusProcessed,
		ebicsRTNEventStatusRetryable, ebicsRTNEventStatusQuarantined,
		ebicsRTNEventStatusFailed:
		return nil
	case "":
		return database.NewValidationError("the RTN event status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported RTN event status", value)
	}
}

func validateEbicsRTNEventCoherence(event *EbicsRTNEvent) error {
	if !event.ProcessedAt.IsZero() && event.ProcessedAt.Before(event.ReceivedAt) {
		return database.NewValidationError("the RTN event processedAt cannot be before receivedAt")
	}

	if event.Status == ebicsRTNEventStatusProcessed && event.ProcessedAt.IsZero() {
		return database.NewValidationError("a processed RTN event requires a processedAt timestamp")
	}

	if event.Status == ebicsRTNEventStatusRetryable && event.NextRetryAt.IsZero() {
		return database.NewValidationError("a retryable RTN event requires a nextRetryAt timestamp")
	}

	if event.Status == ebicsRTNEventStatusFailed && event.LastError == "" {
		return database.NewValidationError("a failed RTN event requires a lastError message")
	}

	return nil
}

func validateEbicsRTNEventRefs(db database.Access, event *EbicsRTNEvent) error {
	if event.EbicsHostID.Valid {
		var host EbicsHost
		if err := db.Get(&host, "id=?", event.EbicsHostID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf("the EBICS host %d does not exist", event.EbicsHostID.Int64)
			}

			return fmt.Errorf("failed to retrieve EBICS host for RTN event: %w", err)
		}
	}

	if event.EbicsSubscriberID.Valid {
		var subscriber EbicsSubscriber
		if err := db.Get(&subscriber, "id=?", event.EbicsSubscriberID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf(
					"the EBICS subscriber %d does not exist", event.EbicsSubscriberID.Int64)
			}

			return fmt.Errorf("failed to retrieve EBICS subscriber for RTN event: %w", err)
		}

		if event.EbicsHostID.Valid && subscriber.EbicsHostID != event.EbicsHostID.Int64 {
			return database.NewValidationError(
				"the RTN event subscriber does not belong to the selected EBICS host")
		}
	}

	return nil
}

func validateEbicsRTNEventUniqueness(db database.Access, event *EbicsRTNEvent) error {
	count, err := db.Count(event).Where(
		"id<>? AND owner=? AND idempotence_key=?",
		event.ID, event.Owner, event.IdempotenceKey,
	).Run()
	if err != nil {
		return fmt.Errorf("failed to check duplicate RTN idempotence keys: %w", err)
	}

	if count != 0 {
		return database.NewValidationErrorf(
			"an RTN event already exists for idempotence key %q", event.IdempotenceKey)
	}

	return nil
}
