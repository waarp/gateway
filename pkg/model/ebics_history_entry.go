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
	ebicsHistoryTypeOperation = "OPERATION"
	ebicsHistoryTypeAction    = "ACTION"
)

// EbicsHistoryEntry stores one durable, append-only history snapshot for a
// non-payload EBICS action or operation.
type EbicsHistoryEntry struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	HistoryType string `xorm:"history_type"`

	OperationType string `xorm:"operation_type"`
	Action        string `xorm:"action"`
	OrderType     string `xorm:"order_type"`
	Direction     string `xorm:"direction"`
	TransportMode string `xorm:"transport_mode"`

	Status                 string `xorm:"status"`
	Severity               string `xorm:"severity"`
	TechnicalReturnCode    string `xorm:"technical_return_code"`
	TechnicalReturnMessage string `xorm:"technical_return_message"`
	BusinessReturnCode     string `xorm:"business_return_code"`
	BusinessReturnMessage  string `xorm:"business_return_message"`
	GatewayOutcome         string `xorm:"gateway_outcome"`
	RetryDecision          string `xorm:"retry_decision"`

	ClientID          sql.NullInt64 `xorm:"client_id"`
	EbicsHostID       int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID int64         `xorm:"ebics_subscriber_id"`
	OperationID       sql.NullInt64 `xorm:"operation_id"`
	TransferID        sql.NullInt64 `xorm:"transfer_id"`
	WorkflowID        sql.NullInt64 `xorm:"workflow_id"`
	LifecycleID       sql.NullInt64 `xorm:"lifecycle_id"`
	CoordinationID    string        `xorm:"coordination_id"`

	RequestID     string `xorm:"request_id"`
	CorrelationID string `xorm:"correlation_id"`
	TransactionID string `xorm:"transaction_id"`
	Operator      string `xorm:"operator"`
	Reason        string `xorm:"reason"`
	Evidence      string `xorm:"evidence"`
	Metadata      string `xorm:"metadata"`

	StartedAt  time.Time `xorm:"started_at DATETIME(6) UTC"`
	FinishedAt time.Time `xorm:"finished_at DATETIME(6) UTC"`
	CreatedAt  time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`

	EvidenceMap map[string]any `xorm:"-"`
	MetadataMap map[string]any `xorm:"-"`
}

func (*EbicsHistoryEntry) TableName() string   { return TableEbicsHistoryEntries }
func (*EbicsHistoryEntry) Appellation() string { return NameEbicsHistoryEntry }
func (e *EbicsHistoryEntry) GetID() int64      { return e.ID }

func (e *EbicsHistoryEntry) BeforeWrite(db database.Access) error {
	e.Owner = conf.GlobalConfig.GatewayName
	e.HistoryType = strings.ToUpper(strings.TrimSpace(e.HistoryType))
	e.OperationType = strings.ToUpper(strings.TrimSpace(e.OperationType))
	e.Action = strings.TrimSpace(e.Action)
	e.OrderType = NormalizeEbicsOrderType(e.OrderType)
	e.Direction = strings.ToUpper(strings.TrimSpace(e.Direction))
	e.TransportMode = strings.ToUpper(strings.TrimSpace(e.TransportMode))
	e.Status = strings.ToUpper(strings.TrimSpace(e.Status))
	e.Severity = strings.ToUpper(strings.TrimSpace(e.Severity))
	e.TechnicalReturnCode = strings.TrimSpace(e.TechnicalReturnCode)
	e.TechnicalReturnMessage = strings.TrimSpace(e.TechnicalReturnMessage)
	e.BusinessReturnCode = strings.TrimSpace(e.BusinessReturnCode)
	e.BusinessReturnMessage = strings.TrimSpace(e.BusinessReturnMessage)
	e.GatewayOutcome = strings.ToUpper(strings.TrimSpace(e.GatewayOutcome))
	e.RetryDecision = strings.ToUpper(strings.TrimSpace(e.RetryDecision))
	e.CoordinationID = strings.TrimSpace(e.CoordinationID)
	e.RequestID = strings.TrimSpace(e.RequestID)
	e.CorrelationID = strings.TrimSpace(e.CorrelationID)
	e.TransactionID = strings.TrimSpace(e.TransactionID)
	e.Operator = strings.TrimSpace(e.Operator)
	e.Reason = strings.TrimSpace(e.Reason)
	e.Evidence = strings.TrimSpace(e.Evidence)
	e.Metadata = strings.TrimSpace(e.Metadata)

	if e.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS history host reference is missing")
	}

	if e.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS history subscriber reference is missing")
	}

	switch e.HistoryType {
	case ebicsHistoryTypeOperation, ebicsHistoryTypeAction:
	default:
		if e.HistoryType == "" {
			return database.NewValidationError("the EBICS history type is missing")
		}

		return database.NewValidationErrorf("%q is not a supported EBICS history type", e.HistoryType)
	}

	if e.OperationType == "" {
		return database.NewValidationError("the EBICS history operation type is missing")
	}

	if e.Status == "" {
		return database.NewValidationError("the EBICS history status is missing")
	}

	if err := e.hydratePayloads(); err != nil {
		return err
	}

	return validateEbicsHistoryRefs(db, e)
}

func (e *EbicsHistoryEntry) hydratePayloads() error {
	if e.EvidenceMap != nil {
		payload, err := serializeStringMap(e.EvidenceMap)
		if err != nil {
			return fmt.Errorf("failed to serialize EBICS history evidence: %w", err)
		}
		e.Evidence = payload
	} else if e.Evidence == "" {
		e.Evidence = emptyJSONObject
	}

	if e.MetadataMap != nil {
		payload, err := serializeStringMap(e.MetadataMap)
		if err != nil {
			return fmt.Errorf("failed to serialize EBICS history metadata: %w", err)
		}
		e.Metadata = payload
	} else if e.Metadata == "" {
		e.Metadata = emptyJSONObject
	}

	evidence, err := deserializeStringMap(e.Evidence)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS history evidence: %w", err)
	}
	metadata, err := deserializeStringMap(e.Metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS history metadata: %w", err)
	}

	e.EvidenceMap = evidence
	e.MetadataMap = metadata

	return nil
}

func (e *EbicsHistoryEntry) AfterRead(database.ReadAccess) error {
	evidence, err := deserializeStringMap(e.Evidence)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS history evidence after read: %w", err)
	}
	metadata, err := deserializeStringMap(e.Metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS history metadata after read: %w", err)
	}

	e.EvidenceMap = evidence
	e.MetadataMap = metadata

	return nil
}

func (e *EbicsHistoryEntry) AfterInsert(db database.Access) error { return e.AfterRead(db) }
func (e *EbicsHistoryEntry) AfterUpdate(db database.Access) error { return e.AfterRead(db) }

func validateEbicsHistoryRefs(db database.Access, entry *EbicsHistoryEntry) error {
	var host EbicsHost
	if err := db.Get(&host, "id=?", entry.EbicsHostID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %d does not exist", entry.EbicsHostID)
		}

		return fmt.Errorf("failed to retrieve EBICS history host: %w", err)
	}

	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", entry.EbicsSubscriberID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", entry.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS history subscriber: %w", err)
	}

	if subscriber.EbicsHostID != entry.EbicsHostID {
		return database.NewValidationError("the EBICS history subscriber does not belong to the selected host")
	}

	if entry.OperationID.Valid {
		if err := validateEbicsOperationExists(db, entry.OperationID.Int64, "history"); err != nil {
			return err
		}
	}
	if entry.WorkflowID.Valid {
		var workflow EbicsInitializationWorkflow
		if err := db.Get(&workflow, "id=?", entry.WorkflowID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf(
					"the history initialization workflow %d does not exist", entry.WorkflowID.Int64)
			}

			return fmt.Errorf("failed to retrieve EBICS history initialization workflow: %w", err)
		}
	}
	if entry.LifecycleID.Valid {
		var lifecycle EbicsKeyLifecycle
		if err := db.Get(&lifecycle, "id=?", entry.LifecycleID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf(
					"the history key lifecycle %d does not exist", entry.LifecycleID.Int64)
			}

			return fmt.Errorf("failed to retrieve EBICS history key lifecycle: %w", err)
		}
	}

	return nil
}

func EbicsHistoryTypeOperationForRuntime() string { return ebicsHistoryTypeOperation }
func EbicsHistoryTypeActionForRuntime() string    { return ebicsHistoryTypeAction }
