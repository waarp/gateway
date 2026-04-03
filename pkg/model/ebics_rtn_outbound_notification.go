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
	ebicsRTNOutboundNotificationStatusPending     = "PENDING"
	ebicsRTNOutboundNotificationStatusProcessing  = "PROCESSING"
	ebicsRTNOutboundNotificationStatusSent        = "SENT"
	ebicsRTNOutboundNotificationStatusRetryable   = "RETRYABLE"
	ebicsRTNOutboundNotificationStatusQuarantined = "QUARANTINED"
	ebicsRTNOutboundNotificationStatusFailed      = "FAILED"

	DefaultEbicsRTNOutboundMaxAttempts       = 3
	DefaultEbicsRTNOutboundRetryDelaySeconds = 60
)

type EbicsRTNOutboundNotification struct {
	ID              int64  `xorm:"<- id AUTOINCR"`
	Owner           string `xorm:"owner"`
	ProviderID      int64  `xorm:"provider_id"`
	EventType       string `xorm:"event_type"`
	SourceOrderType string `xorm:"source_order_type"`
	CorrelationID   string `xorm:"correlation_id"`

	EbicsHostID            int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID      int64         `xorm:"ebics_subscriber_id"`
	ServerReportingSetID   sql.NullInt64 `xorm:"server_reporting_set_id"`
	ServerReportingItemKey string        `xorm:"server_reporting_item_key"`

	Payload     string    `xorm:"payload"`
	Status      string    `xorm:"status"`
	Attempts    int       `xorm:"attempts"`
	NextRetryAt time.Time `xorm:"next_retry_at DATETIME(6) UTC"`
	SentAt      time.Time `xorm:"sent_at DATETIME(6) UTC"`
	LastError   string    `xorm:"last_error"`
	CreatedAt   time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt   time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	PayloadMap map[string]any `xorm:"-"`
}

func (*EbicsRTNOutboundNotification) TableName() string   { return TableEbicsRTNOutboundNotifications }
func (*EbicsRTNOutboundNotification) Appellation() string { return NameEbicsRTNOutboundNotification }
func (n *EbicsRTNOutboundNotification) GetID() int64      { return n.ID }

func (n *EbicsRTNOutboundNotification) BeforeWrite(db database.Access) error {
	n.Owner = conf.GlobalConfig.GatewayName
	n.EventType = strings.ToUpper(strings.TrimSpace(n.EventType))
	n.SourceOrderType = strings.ToUpper(strings.TrimSpace(n.SourceOrderType))
	n.CorrelationID = strings.TrimSpace(n.CorrelationID)
	n.ServerReportingItemKey = strings.TrimSpace(n.ServerReportingItemKey)
	n.Payload = strings.TrimSpace(n.Payload)
	n.Status = strings.ToUpper(strings.TrimSpace(n.Status))
	n.LastError = strings.TrimSpace(n.LastError)

	if n.ProviderID == 0 {
		return database.NewValidationError("the outbound RTN notification provider is missing")
	}
	if n.EventType == "" {
		return database.NewValidationError("the outbound RTN notification event type is missing")
	}
	if n.SourceOrderType == "" {
		return database.NewValidationError("the outbound RTN notification source order type is missing")
	}
	if err := validateEbicsRTNOutboundNotificationStatus(n.Status); err != nil {
		return err
	}
	if n.Attempts < 0 {
		return database.NewValidationError("the outbound RTN notification attempts cannot be negative")
	}
	if err := n.hydratePayload(); err != nil {
		return err
	}
	if err := validateEbicsRTNOutboundNotificationRefs(db, n); err != nil {
		return err
	}

	return validateEbicsRTNOutboundNotificationCoherence(n)
}

func (n *EbicsRTNOutboundNotification) hydratePayload() error {
	if n.PayloadMap != nil {
		serialized, err := serializeStringMap(n.PayloadMap)
		if err != nil {
			return fmt.Errorf("failed to serialize outbound RTN notification payload: %w", err)
		}

		n.Payload = serialized
	} else if n.Payload == "" {
		n.Payload = emptyJSONObject
	}

	payload, err := deserializeStringMap(n.Payload)
	if err != nil {
		return fmt.Errorf("failed to deserialize outbound RTN notification payload: %w", err)
	}

	n.PayloadMap = payload

	return nil
}

func (n *EbicsRTNOutboundNotification) AfterRead(database.ReadAccess) error {
	payload, err := deserializeStringMap(n.Payload)
	if err != nil {
		return fmt.Errorf("failed to deserialize outbound RTN notification payload after read: %w", err)
	}

	n.PayloadMap = payload

	return nil
}

func (n *EbicsRTNOutboundNotification) AfterInsert(db database.Access) error { return n.AfterRead(db) }
func (n *EbicsRTNOutboundNotification) AfterUpdate(db database.Access) error { return n.AfterRead(db) }

func validateEbicsRTNOutboundNotificationStatus(value string) error {
	switch value {
	case ebicsRTNOutboundNotificationStatusPending,
		ebicsRTNOutboundNotificationStatusProcessing,
		ebicsRTNOutboundNotificationStatusSent,
		ebicsRTNOutboundNotificationStatusRetryable,
		ebicsRTNOutboundNotificationStatusQuarantined,
		ebicsRTNOutboundNotificationStatusFailed:
		return nil
	case "":
		return database.NewValidationError("the outbound RTN notification status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported outbound RTN notification status", value)
	}
}

func validateEbicsRTNOutboundNotificationRefs(db database.Access, notification *EbicsRTNOutboundNotification) error {
	var provider EbicsRTNOutboundProvider
	if err := db.Get(&provider, "id=?", notification.ProviderID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the outbound RTN provider %d does not exist", notification.ProviderID)
		}

		return fmt.Errorf("failed to retrieve outbound RTN provider: %w", err)
	}

	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", provider.EbicsSubscriberID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve subscriber for outbound RTN notification: %w", err)
	}

	notification.EbicsSubscriberID = subscriber.ID
	notification.EbicsHostID = subscriber.EbicsHostID

	if !notification.ServerReportingSetID.Valid {
		return nil
	}

	var set EbicsServerReportingSet
	if err := db.Get(&set, "id=?", notification.ServerReportingSetID.Int64).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS server reporting set %d does not exist", notification.ServerReportingSetID.Int64)
		}

		return fmt.Errorf("failed to retrieve EBICS server reporting set: %w", err)
	}

	if set.EbicsHostID != notification.EbicsHostID || set.EbicsSubscriberID.Int64 != notification.EbicsSubscriberID {
		return database.NewValidationError(
			"the outbound RTN notification reporting set does not belong to the provider subscriber")
	}

	if NormalizeEbicsOrderType(set.SourceOrderType) != NormalizeEbicsOrderType(notification.SourceOrderType) {
		return database.NewValidationError(
			"the outbound RTN notification source order type does not match the reporting set")
	}

	if notification.ServerReportingItemKey == "" {
		return nil
	}

	var item EbicsServerReportingItem
	if err := db.Get(
		&item,
		"server_reporting_set_id=? AND item_key=?",
		set.ID,
		notification.ServerReportingItemKey,
	).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS server reporting item %q does not exist in set %d",
				notification.ServerReportingItemKey,
				set.ID,
			)
		}

		return fmt.Errorf("failed to retrieve EBICS server reporting item: %w", err)
	}

	return nil
}

func validateEbicsRTNOutboundNotificationCoherence(notification *EbicsRTNOutboundNotification) error {
	if notification.Status == ebicsRTNOutboundNotificationStatusRetryable && notification.NextRetryAt.IsZero() {
		return database.NewValidationError("a retryable outbound RTN notification requires a nextRetryAt timestamp")
	}
	if notification.Status == ebicsRTNOutboundNotificationStatusFailed && notification.LastError == "" {
		return database.NewValidationError("a failed outbound RTN notification requires a lastError message")
	}
	if notification.Status == ebicsRTNOutboundNotificationStatusSent && notification.SentAt.IsZero() {
		return database.NewValidationError("a sent outbound RTN notification requires a sentAt timestamp")
	}

	return nil
}

func EbicsRTNOutboundNotificationStatusPendingForRuntime() string {
	return ebicsRTNOutboundNotificationStatusPending
}

func EbicsRTNOutboundNotificationStatusProcessingForRuntime() string {
	return ebicsRTNOutboundNotificationStatusProcessing
}

func EbicsRTNOutboundNotificationStatusSentForRuntime() string {
	return ebicsRTNOutboundNotificationStatusSent
}

func EbicsRTNOutboundNotificationStatusRetryableForRuntime() string {
	return ebicsRTNOutboundNotificationStatusRetryable
}

func EbicsRTNOutboundNotificationStatusQuarantinedForRuntime() string {
	return ebicsRTNOutboundNotificationStatusQuarantined
}

func EbicsRTNOutboundNotificationStatusFailedForRuntime() string {
	return ebicsRTNOutboundNotificationStatusFailed
}
