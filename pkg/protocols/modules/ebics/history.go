package ebics

import (
	"database/sql"
	"fmt"
	"maps"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// RecordOperationHistory stores one append-only EBICS history snapshot for a
// terminal non-payload operation.
func RecordOperationHistory(db *database.DB, operation *model.EbicsOperation) error {
	if operation == nil {
		return database.NewValidationError("the EBICS operation history source is missing")
	}

	entry := &model.EbicsHistoryEntry{
		HistoryType:            model.EbicsHistoryTypeOperationForRuntime(),
		OperationType:          operation.OperationType,
		OrderType:              operation.OrderType,
		Direction:              operation.Direction,
		TransportMode:          operation.TransportMode,
		Status:                 operation.Status,
		Severity:               operation.Severity,
		TechnicalReturnCode:    operation.TechnicalReturnCode,
		TechnicalReturnMessage: operation.TechnicalReturnMessage,
		BusinessReturnCode:     operation.BusinessReturnCode,
		BusinessReturnMessage:  operation.BusinessReturnMessage,
		GatewayOutcome:         operation.GatewayOutcome,
		RetryDecision:          operation.RetryDecision,
		ClientID:               operation.ClientID,
		EbicsHostID:            operation.EbicsHostID,
		EbicsSubscriberID:      operation.EbicsSubscriberID,
		OperationID:            sql.NullInt64{Int64: operation.ID, Valid: true},
		TransferID:             operation.TransferID,
		RequestID:              operation.RequestID,
		CorrelationID:          operation.CorrelationID,
		TransactionID:          operation.TransactionID,
		MetadataMap:            cloneMap(operation.MetadataMap),
		StartedAt:              operation.StartedAt,
		FinishedAt:             operation.FinishedAt,
	}

	if err := db.Insert(entry).Run(); err != nil {
		return fmt.Errorf("insert EBICS operation history for operation %d: %w", operation.ID, err)
	}

	return nil
}

// RecordInitializationHistory stores one append-only EBICS history snapshot for
// one initialization workflow action.
func RecordInitializationHistory(
	db *database.DB,
	clientID int64,
	workflow *model.EbicsInitializationWorkflow,
	action string,
) error {
	if workflow == nil {
		return database.NewValidationError("the EBICS initialization history source is missing")
	}

	subscriber := &model.EbicsSubscriber{}
	if err := db.Get(subscriber, "id=?", workflow.EbicsSubscriberID).Owner().Run(); err != nil {
		return fmt.Errorf("load subscriber for initialization history %d: %w", workflow.ID, err)
	}

	entry := &model.EbicsHistoryEntry{
		HistoryType:       model.EbicsHistoryTypeActionForRuntime(),
		OperationType:     "INITIALIZATION",
		Action:            action,
		Status:            workflow.Status,
		Severity:          model.EbicsOperationSeverityInfoForRuntime(),
		ClientID:          nullableID(clientID),
		EbicsHostID:       subscriber.EbicsHostID,
		EbicsSubscriberID: workflow.EbicsSubscriberID,
		WorkflowID:        sql.NullInt64{Int64: workflow.ID, Valid: true},
		Operator:          workflow.Operator,
		Reason:            workflow.Reason,
		EvidenceMap:       cloneMap(workflow.EvidenceMap),
		MetadataMap: map[string]any{
			"currentStep":       workflow.CurrentStep,
			"bankFeedback":      workflow.BankFeedback,
			"iniOperationID":    nullInt64ToAny(workflow.IniOperationID),
			"hiaOperationID":    nullInt64ToAny(workflow.HiaOperationID),
			"h3kOperationID":    nullInt64ToAny(workflow.H3KOperationID),
			"letterGeneratedAt": timeOrNil(workflow.LetterGeneratedAt),
			"letterConfirmedAt": timeOrNil(workflow.LetterConfirmedAt),
			"bankActivationAt":  timeOrNil(workflow.BankActivationAt),
		},
		CreatedAt: time.Now().UTC(),
	}

	if operationID := operationIDFromInitializationAction(workflow, action); operationID != 0 {
		entry.OperationID = sql.NullInt64{Int64: operationID, Valid: true}
	}

	if err := db.Insert(entry).Run(); err != nil {
		return fmt.Errorf("insert EBICS initialization history for workflow %d: %w", workflow.ID, err)
	}

	return nil
}

// RecordKeyLifecycleHistory stores one append-only EBICS history snapshot for
// one local key lifecycle action.
func RecordKeyLifecycleHistory(db *database.DB, lifecycle *model.EbicsKeyLifecycle, action string) error {
	if lifecycle == nil {
		return database.NewValidationError("the EBICS key lifecycle history source is missing")
	}

	subscriber := &model.EbicsSubscriber{}
	if err := db.Get(subscriber, "id=?", lifecycle.EbicsSubscriberID).Owner().Run(); err != nil {
		return fmt.Errorf("load subscriber for key lifecycle history %d: %w", lifecycle.ID, err)
	}

	entry := &model.EbicsHistoryEntry{
		HistoryType:       model.EbicsHistoryTypeActionForRuntime(),
		OperationType:     "KEY_MANAGEMENT",
		Action:            action,
		Status:            lifecycle.Status,
		Severity:          model.EbicsOperationSeverityInfoForRuntime(),
		EbicsHostID:       subscriber.EbicsHostID,
		EbicsSubscriberID: lifecycle.EbicsSubscriberID,
		LifecycleID:       sql.NullInt64{Int64: lifecycle.ID, Valid: true},
		CoordinationID:    lifecycle.CoordinationID,
		Operator:          lifecycle.Operator,
		Reason:            lifecycle.Reason,
		EvidenceMap:       cloneMap(lifecycle.EvidenceMap),
		MetadataMap: map[string]any{
			"keyUsage":            lifecycle.KeyUsage,
			"rotationType":        lifecycle.RotationType,
			"currentCredentialID": lifecycle.CurrentCredentialID,
			"nextCredentialID":    nullInt64ToAny(lifecycle.NextCredentialID),
			"triggerOperationID":  nullInt64ToAny(lifecycle.TriggerOperationID),
			"lastOperationID":     nullInt64ToAny(lifecycle.LastOperationID),
			"requestedAt":         timeOrNil(lifecycle.RequestedAt),
			"sentAt":              timeOrNil(lifecycle.SentAt),
			"activatedAt":         timeOrNil(lifecycle.ActivatedAt),
			"retiredAt":           timeOrNil(lifecycle.RetiredAt),
		},
		CreatedAt: time.Now().UTC(),
	}

	if lifecycle.LastOperationID.Valid {
		entry.OperationID = lifecycle.LastOperationID
	}

	if err := db.Insert(entry).Run(); err != nil {
		return fmt.Errorf("insert EBICS key lifecycle history for lifecycle %d: %w", lifecycle.ID, err)
	}

	return nil
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}

	out := make(map[string]any, len(input))
	maps.Copy(out, input)

	return out
}

func nullInt64ToAny(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}

	return value.Int64
}

func timeOrNil(value time.Time) any {
	if value.IsZero() {
		return nil
	}

	return value.UTC()
}

func operationIDFromInitializationAction(workflow *model.EbicsInitializationWorkflow, action string) int64 {
	switch action {
	case "SEND_INI":
		if workflow.IniOperationID.Valid {
			return workflow.IniOperationID.Int64
		}
	case "SEND_HIA":
		if workflow.HiaOperationID.Valid {
			return workflow.HiaOperationID.Int64
		}
	case "SEND_H3K":
		if workflow.H3KOperationID.Valid {
			return workflow.H3KOperationID.Int64
		}
	case "CONFIRM_BANK_ACTIVATION":
		if workflow.H3KOperationID.Valid {
			return workflow.H3KOperationID.Int64
		}
	}

	return 0
}
