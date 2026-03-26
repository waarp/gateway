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
	ebicsOperationTypeInitialization  = "INITIALIZATION"
	ebicsOperationTypeKeyManagement   = "KEY_MANAGEMENT"
	ebicsOperationTypeContractRefresh = "CONTRACT_REFRESH"
	ebicsOperationTypeReporting       = "REPORTING"
	ebicsOperationTypeSignature       = "SIGNATURE"
	ebicsOperationTypeRTN             = "RTN"
	ebicsOperationTypeAdmin           = "ADMIN"

	ebicsOperationDirectionInbound  = "INBOUND"
	ebicsOperationDirectionOutbound = "OUTBOUND"
	ebicsOperationDirectionInternal = "INTERNAL"

	ebicsTransportModeSync          = "SYNC"
	ebicsTransportModeAsync         = "ASYNC"
	ebicsTransportModeAutoTriggered = "AUTO_TRIGGERED"

	ebicsOperationSeverityInfo    = "INFO"
	ebicsOperationSeverityWarning = "WARNING"
	ebicsOperationSeverityError   = "ERROR"

	ebicsGatewayOutcomeSuccess                    = "SUCCESS"
	ebicsGatewayOutcomeSuccessWithWarning         = "SUCCESS_WITH_WARNING"
	ebicsGatewayOutcomePendingBank                = "PENDING_BANK"
	ebicsGatewayOutcomeEmptySuccess               = "EMPTY_SUCCESS"
	ebicsGatewayOutcomeTechnicalRetryableFailure  = "TECHNICAL_RETRYABLE_FAILURE"
	ebicsGatewayOutcomeTechnicalFatalFailure      = "TECHNICAL_FATAL_FAILURE"
	ebicsGatewayOutcomeBusinessRejected           = "BUSINESS_REJECTED"
	ebicsGatewayOutcomeManualConfirmationRequired = "MANUAL_CONFIRMATION_REQUIRED"

	ebicsRetryDecisionNoRetry                = "NO_RETRY"
	ebicsRetryDecisionAutoRetryAllowed       = "AUTO_RETRY_ALLOWED"
	ebicsRetryDecisionRecoveryRequired       = "RECOVERY_REQUIRED"
	ebicsRetryDecisionManualReplayOnly       = "MANUAL_REPLAY_ONLY"
	ebicsRetryDecisionManualConfirmationOnly = "MANUAL_CONFIRMATION_ONLY"

	ebicsOperationStatusPlanned                     = "PLANNED"
	ebicsOperationStatusReady                       = "READY"
	ebicsOperationStatusRunning                     = "RUNNING"
	ebicsOperationStatusWaitingExternalConfirmation = "WAITING_EXTERNAL_CONFIRMATION"
	ebicsOperationStatusWaitingBank                 = "WAITING_BANK"
	ebicsOperationStatusWaitingPayloadTransfer      = "WAITING_PAYLOAD_TRANSFER"
	ebicsOperationStatusCompleted                   = "COMPLETED"
	ebicsOperationStatusCompletedWithWarnings       = "COMPLETED_WITH_WARNINGS"
	ebicsOperationStatusFailed                      = "FAILED"
	ebicsOperationStatusCancelled                   = "CANCELLED"
)

// EbicsOperation stores the protocol-level execution state of an EBICS order.
type EbicsOperation struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	LocalAgentID      sql.NullInt64 `xorm:"local_agent_id"`
	ClientID          sql.NullInt64 `xorm:"client_id"`
	RemoteAgentID     sql.NullInt64 `xorm:"remote_agent_id"`
	LocalAccountID    sql.NullInt64 `xorm:"local_account_id"`
	RemoteAccountID   sql.NullInt64 `xorm:"remote_account_id"`
	EbicsHostID       int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID int64         `xorm:"ebics_subscriber_id"`

	OperationType string `xorm:"operation_type"`
	OrderType     string `xorm:"order_type"`
	Direction     string `xorm:"direction"`
	TransportMode string `xorm:"transport_mode"`

	TransactionID string `xorm:"transaction_id"`
	RequestID     string `xorm:"request_id"`
	CorrelationID string `xorm:"correlation_id"`
	EbicsVersion  string `xorm:"ebics_version"`

	Status                 string `xorm:"status"`
	Severity               string `xorm:"severity"`
	TechnicalReturnCode    string `xorm:"technical_return_code"`
	TechnicalReturnMessage string `xorm:"technical_return_message"`
	BusinessReturnCode     string `xorm:"business_return_code"`
	BusinessReturnMessage  string `xorm:"business_return_message"`
	GatewayOutcome         string `xorm:"gateway_outcome"`
	RetryDecision          string `xorm:"retry_decision"`
	ManualActionRequired   bool   `xorm:"manual_action_required"`

	TransferID     sql.NullInt64 `xorm:"transfer_id"`
	ContractViewID sql.NullInt64 `xorm:"contract_view_id"`
	RTNEventID     sql.NullInt64 `xorm:"rtn_event_id"`
	Metadata       string        `xorm:"metadata"`

	StartedAt  time.Time `xorm:"started_at DATETIME(6) UTC"`
	FinishedAt time.Time `xorm:"finished_at DATETIME(6) UTC"`
	CreatedAt  time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt  time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	MetadataMap map[string]any `xorm:"-"`
}

// TableName returns the persistent table name for EBICS operations.
func (*EbicsOperation) TableName() string { return TableEbicsOperations }

// Appellation returns the display name used in validation messages.
func (*EbicsOperation) Appellation() string { return NameEbicsOperation }

// GetID returns the database identifier of the operation.
func (o *EbicsOperation) GetID() int64 { return o.ID }

// BeforeWrite normalizes and validates an EBICS operation before persistence.
func (o *EbicsOperation) BeforeWrite(db database.Access) error {
	o.normalize()

	if err := o.validate(); err != nil {
		return err
	}

	if err := o.hydrateMetadata(); err != nil {
		return err
	}

	bindingErr := validateEbicsOperationBinding(o)
	if bindingErr != nil {
		return bindingErr
	}

	return validateEbicsOperationRefs(db, o)
}

func (o *EbicsOperation) normalize() {
	o.Owner = conf.GlobalConfig.GatewayName
	o.OperationType = strings.ToUpper(strings.TrimSpace(o.OperationType))
	o.OrderType = strings.ToUpper(strings.TrimSpace(o.OrderType))
	o.Direction = strings.ToUpper(strings.TrimSpace(o.Direction))
	o.TransportMode = strings.ToUpper(strings.TrimSpace(o.TransportMode))
	o.TransactionID = strings.TrimSpace(o.TransactionID)
	o.RequestID = strings.TrimSpace(o.RequestID)
	o.CorrelationID = strings.TrimSpace(o.CorrelationID)
	o.EbicsVersion = strings.ToUpper(strings.TrimSpace(o.EbicsVersion))
	o.Status = strings.ToUpper(strings.TrimSpace(o.Status))
	o.Severity = strings.ToUpper(strings.TrimSpace(o.Severity))
	o.TechnicalReturnCode = strings.TrimSpace(o.TechnicalReturnCode)
	o.TechnicalReturnMessage = strings.TrimSpace(o.TechnicalReturnMessage)
	o.BusinessReturnCode = strings.TrimSpace(o.BusinessReturnCode)
	o.BusinessReturnMessage = strings.TrimSpace(o.BusinessReturnMessage)
	o.GatewayOutcome = strings.ToUpper(strings.TrimSpace(o.GatewayOutcome))
	o.RetryDecision = strings.ToUpper(strings.TrimSpace(o.RetryDecision))
	o.Metadata = strings.TrimSpace(o.Metadata)
}

func (o *EbicsOperation) validate() error {
	if o.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS host reference is missing")
	}

	if o.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS subscriber reference is missing")
	}

	if err := validateEbicsOperationType(o.OperationType); err != nil {
		return err
	}

	if err := validateEbicsPayloadOrderType(o.OrderType); err != nil {
		return err
	}

	if err := validateEbicsOperationDirection(o.Direction); err != nil {
		return err
	}

	if err := validateEbicsTransportMode(o.TransportMode); err != nil {
		return err
	}

	if err := validateEbicsOperationStatus(o.Status); err != nil {
		return err
	}

	if err := validateEbicsOperationSeverity(o.Severity); err != nil {
		return err
	}

	if err := validateEbicsGatewayOutcome(o.GatewayOutcome); err != nil {
		return err
	}

	return validateEbicsRetryDecision(o.RetryDecision)
}

func (o *EbicsOperation) hydrateMetadata() error {
	if o.MetadataMap != nil {
		serialized, err := serializeStringMap(o.MetadataMap)
		if err != nil {
			return fmt.Errorf("failed to serialize EBICS operation metadata: %w", err)
		}

		o.Metadata = serialized
	} else if o.Metadata == "" {
		o.Metadata = emptyJSONObject
	}

	meta, err := deserializeStringMap(o.Metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS operation metadata: %w", err)
	}

	o.MetadataMap = meta

	return nil
}

func validateEbicsOperationRefs(db database.Access, operation *EbicsOperation) error {
	var host EbicsHost
	err := db.Get(&host, "id=?", operation.EbicsHostID).Run()
	if err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %d does not exist", operation.EbicsHostID)
		}

		return fmt.Errorf("failed to retrieve EBICS host: %w", err)
	}

	var subscriber EbicsSubscriber
	err = db.Get(&subscriber, "id=?", operation.EbicsSubscriberID).Run()
	if err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS subscriber %d does not exist", operation.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber: %w", err)
	}

	if subscriber.EbicsHostID != operation.EbicsHostID {
		return database.NewValidationError("the EBICS operation subscriber does not belong to the selected host")
	}

	return nil
}

// AfterRead hydrates the transient metadata map after a database read.
func (o *EbicsOperation) AfterRead(database.ReadAccess) error {
	meta, err := deserializeStringMap(o.Metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS operation metadata after read: %w", err)
	}

	o.MetadataMap = meta

	return nil
}

// AfterInsert refreshes transient state after insertion.
func (o *EbicsOperation) AfterInsert(db database.Access) error {
	return o.AfterRead(db)
}

// AfterUpdate refreshes transient state after update.
func (o *EbicsOperation) AfterUpdate(db database.Access) error {
	return o.AfterRead(db)
}

func validateEbicsOperationType(value string) error {
	switch value {
	case ebicsOperationTypeInitialization, ebicsOperationTypeKeyManagement,
		ebicsOperationTypeContractRefresh, ebicsOperationTypeReporting,
		ebicsOperationTypeSignature, ebicsOperationTypeRTN, ebicsOperationTypeAdmin:
		return nil
	case "":
		return database.NewValidationError("the EBICS operation type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS operation type", value)
	}
}

func validateEbicsOperationStatus(value string) error {
	switch value {
	case ebicsOperationStatusPlanned, ebicsOperationStatusReady, ebicsOperationStatusRunning,
		ebicsOperationStatusWaitingExternalConfirmation, ebicsOperationStatusWaitingBank,
		ebicsOperationStatusWaitingPayloadTransfer, ebicsOperationStatusCompleted,
		ebicsOperationStatusCompletedWithWarnings, ebicsOperationStatusFailed,
		ebicsOperationStatusCancelled:
		return nil
	case "":
		return database.NewValidationError("the EBICS operation status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS operation status", value)
	}
}

func validateEbicsOperationSeverity(value string) error {
	switch value {
	case ebicsOperationSeverityInfo, ebicsOperationSeverityWarning, ebicsOperationSeverityError:
		return nil
	case "":
		return database.NewValidationError("the EBICS operation severity is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS operation severity", value)
	}
}

func validateEbicsOperationDirection(value string) error {
	switch value {
	case ebicsOperationDirectionInbound, ebicsOperationDirectionOutbound, ebicsOperationDirectionInternal:
		return nil
	case "":
		return database.NewValidationError("the EBICS operation direction is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS operation direction", value)
	}
}

func validateEbicsTransportMode(value string) error {
	switch value {
	case ebicsTransportModeSync, ebicsTransportModeAsync, ebicsTransportModeAutoTriggered:
		return nil
	case "":
		return database.NewValidationError("the EBICS transport mode is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS transport mode", value)
	}
}

func validateEbicsGatewayOutcome(value string) error {
	switch value {
	case ebicsGatewayOutcomeSuccess, ebicsGatewayOutcomeSuccessWithWarning,
		ebicsGatewayOutcomePendingBank, ebicsGatewayOutcomeEmptySuccess,
		ebicsGatewayOutcomeTechnicalRetryableFailure, ebicsGatewayOutcomeTechnicalFatalFailure,
		ebicsGatewayOutcomeBusinessRejected, ebicsGatewayOutcomeManualConfirmationRequired:
		return nil
	case "":
		return database.NewValidationError("the EBICS gateway outcome is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS gateway outcome", value)
	}
}

func validateEbicsRetryDecision(value string) error {
	switch value {
	case ebicsRetryDecisionNoRetry, ebicsRetryDecisionAutoRetryAllowed,
		ebicsRetryDecisionRecoveryRequired, ebicsRetryDecisionManualReplayOnly,
		ebicsRetryDecisionManualConfirmationOnly:
		return nil
	case "":
		return database.NewValidationError("the EBICS retry decision is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS retry decision", value)
	}
}

func validateEbicsOperationBinding(o *EbicsOperation) error {
	if o.TransferID.Valid && !isPayloadOrderType(o.OrderType) {
		return database.NewValidationError("a non-payload EBICS operation cannot reference a transfer")
	}

	if o.FinishedAt.IsZero() {
		return nil
	}

	if !o.StartedAt.IsZero() && o.FinishedAt.Before(o.StartedAt) {
		return database.NewValidationError("the EBICS operation finishedAt cannot be before startedAt")
	}

	return nil
}

func isPayloadOrderType(orderType string) bool {
	switch orderType {
	case ebicsPayloadOrderBTU, ebicsPayloadOrderBTD, ebicsPayloadOrderFUL, ebicsPayloadOrderFDL:
		return true
	default:
		return false
	}
}

// EbicsOperationStatusPlannedForRuntime exposes the planned status to runtime packages.
func EbicsOperationStatusPlannedForRuntime() string {
	return ebicsOperationStatusPlanned
}

// EbicsOperationStatusWaitingBankForRuntime exposes the waiting-bank status to runtime packages.
func EbicsOperationStatusWaitingBankForRuntime() string {
	return ebicsOperationStatusWaitingBank
}

// EbicsOperationStatusCompletedForRuntime exposes the completed status to runtime packages.
func EbicsOperationStatusCompletedForRuntime() string {
	return ebicsOperationStatusCompleted
}

// EbicsOperationStatusCompletedWithWarningsForRuntime exposes the completed-with-warnings status.
func EbicsOperationStatusCompletedWithWarningsForRuntime() string {
	return ebicsOperationStatusCompletedWithWarnings
}

// EbicsOperationStatusFailedForRuntime exposes the failed status to runtime packages.
func EbicsOperationStatusFailedForRuntime() string {
	return ebicsOperationStatusFailed
}

// EbicsOperationSeverityInfoForRuntime exposes the info severity to runtime packages.
func EbicsOperationSeverityInfoForRuntime() string {
	return ebicsOperationSeverityInfo
}

// EbicsOperationSeverityWarningForRuntime exposes the warning severity to runtime packages.
func EbicsOperationSeverityWarningForRuntime() string {
	return ebicsOperationSeverityWarning
}

// EbicsOperationSeverityErrorForRuntime exposes the error severity to runtime packages.
func EbicsOperationSeverityErrorForRuntime() string {
	return ebicsOperationSeverityError
}

// EbicsGatewayOutcomeSuccessForRuntime exposes the success outcome to runtime packages.
func EbicsGatewayOutcomeSuccessForRuntime() string {
	return ebicsGatewayOutcomeSuccess
}

// EbicsGatewayOutcomeSuccessWithWarningForRuntime exposes the warning outcome to runtime packages.
func EbicsGatewayOutcomeSuccessWithWarningForRuntime() string {
	return ebicsGatewayOutcomeSuccessWithWarning
}

// EbicsGatewayOutcomePendingBankForRuntime exposes the pending-bank outcome to runtime packages.
func EbicsGatewayOutcomePendingBankForRuntime() string {
	return ebicsGatewayOutcomePendingBank
}

// EbicsGatewayOutcomeEmptySuccessForRuntime exposes the empty-success outcome to runtime packages.
func EbicsGatewayOutcomeEmptySuccessForRuntime() string {
	return ebicsGatewayOutcomeEmptySuccess
}

// EbicsGatewayOutcomeTechnicalRetryableFailureForRuntime exposes the retryable technical failure outcome.
func EbicsGatewayOutcomeTechnicalRetryableFailureForRuntime() string {
	return ebicsGatewayOutcomeTechnicalRetryableFailure
}

// EbicsGatewayOutcomeTechnicalFatalFailureForRuntime exposes the fatal technical failure outcome.
func EbicsGatewayOutcomeTechnicalFatalFailureForRuntime() string {
	return ebicsGatewayOutcomeTechnicalFatalFailure
}

// EbicsGatewayOutcomeBusinessRejectedForRuntime exposes the business rejection outcome to runtime packages.
func EbicsGatewayOutcomeBusinessRejectedForRuntime() string {
	return ebicsGatewayOutcomeBusinessRejected
}

// EbicsGatewayOutcomeManualConfirmationRequiredForRuntime exposes the manual-confirmation outcome.
func EbicsGatewayOutcomeManualConfirmationRequiredForRuntime() string {
	return ebicsGatewayOutcomeManualConfirmationRequired
}

// EbicsRetryDecisionNoRetryForRuntime exposes the no-retry decision to runtime packages.
func EbicsRetryDecisionNoRetryForRuntime() string {
	return ebicsRetryDecisionNoRetry
}

// EbicsRetryDecisionAutoRetryAllowedForRuntime exposes the automatic retry decision.
func EbicsRetryDecisionAutoRetryAllowedForRuntime() string {
	return ebicsRetryDecisionAutoRetryAllowed
}

// EbicsRetryDecisionRecoveryRequiredForRuntime exposes the recovery-required decision.
func EbicsRetryDecisionRecoveryRequiredForRuntime() string {
	return ebicsRetryDecisionRecoveryRequired
}

// EbicsRetryDecisionManualReplayOnlyForRuntime exposes the manual-replay-only decision.
func EbicsRetryDecisionManualReplayOnlyForRuntime() string {
	return ebicsRetryDecisionManualReplayOnly
}

// EbicsRetryDecisionManualConfirmationOnlyForRuntime exposes the manual-confirmation-only decision.
func EbicsRetryDecisionManualConfirmationOnlyForRuntime() string {
	return ebicsRetryDecisionManualConfirmationOnly
}
