package runtime

import (
	"database/sql"
	"fmt"
	"maps"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type OperationMappingInput struct {
	Owner             string
	LocalAgentID      int64
	ClientID          int64
	RemoteAgentID     int64
	LocalAccountID    int64
	RemoteAccountID   int64
	EbicsHostID       int64
	EbicsSubscriberID int64
	OrderType         string
	OperationType     string
	Direction         string
	TransportMode     string
	CorrelationID     string
	ContractViewID    int64
	ResolvedRequest   *ResolvedPayloadRequest
}

func NewPayloadOperation(input *OperationMappingInput) (*model.EbicsOperation, error) {
	if input == nil {
		return nil, database.NewValidationError("the EBICS operation mapping input is missing")
	}

	orderType := strings.ToUpper(strings.TrimSpace(input.OrderType))
	if input.ResolvedRequest != nil && orderType == "" {
		orderType = input.ResolvedRequest.OrderType
	}

	if orderType == "" {
		return nil, database.NewValidationError("the EBICS payload operation order type is missing")
	}

	operation := &model.EbicsOperation{
		EbicsHostID:       input.EbicsHostID,
		EbicsSubscriberID: input.EbicsSubscriberID,
		OperationType:     strings.ToUpper(strings.TrimSpace(input.OperationType)),
		OrderType:         orderType,
		Direction:         strings.ToUpper(strings.TrimSpace(input.Direction)),
		TransportMode:     strings.ToUpper(strings.TrimSpace(input.TransportMode)),
		CorrelationID:     strings.TrimSpace(input.CorrelationID),
		Status:            model.EbicsOperationStatusPlannedForRuntime(),
		Severity:          model.EbicsOperationSeverityInfoForRuntime(),
		GatewayOutcome:    model.EbicsGatewayOutcomePendingBankForRuntime(),
		RetryDecision:     model.EbicsRetryDecisionNoRetryForRuntime(),
		MetadataMap:       map[string]any{},
	}

	if input.Owner != "" {
		operation.Owner = strings.TrimSpace(input.Owner)
	}

	if input.LocalAgentID > 0 {
		operation.LocalAgentID = utils.NewNullInt64(input.LocalAgentID)
	}

	if input.ClientID > 0 {
		operation.ClientID = utils.NewNullInt64(input.ClientID)
	}

	if input.RemoteAgentID > 0 {
		operation.RemoteAgentID = utils.NewNullInt64(input.RemoteAgentID)
	}

	if input.LocalAccountID > 0 {
		operation.LocalAccountID = utils.NewNullInt64(input.LocalAccountID)
	}

	if input.RemoteAccountID > 0 {
		operation.RemoteAccountID = utils.NewNullInt64(input.RemoteAccountID)
	}

	if input.ContractViewID > 0 {
		operation.ContractViewID = utils.NewNullInt64(input.ContractViewID)
	}

	if input.ResolvedRequest != nil {
		operation.ContractViewID = nullIfPositive(
			input.ResolvedRequest.ContractViewID,
			operation.ContractViewID,
		)
		operation.MetadataMap["resolutionMode"] = input.ResolvedRequest.ResolutionMode
		operation.MetadataMap["profileName"] = input.ResolvedRequest.ProfileName
		operation.MetadataMap["ruleName"] = input.ResolvedRequest.RuleName
		operation.MetadataMap["contractItemIDs"] = input.ResolvedRequest.ContractItemIDs
		operation.MetadataMap["declaredAmount"] = input.ResolvedRequest.DeclaredAmount
		operation.MetadataMap["declaredCurrency"] = input.ResolvedRequest.DeclaredCurrency
		operation.MetadataMap["service"] = map[string]any{
			"serviceName":   input.ResolvedRequest.ResolvedService.ServiceName,
			"serviceOption": input.ResolvedRequest.ResolvedService.ServiceOption,
			"scope":         input.ResolvedRequest.ResolvedService.Scope,
			"msgName":       input.ResolvedRequest.ResolvedService.MsgName,
			"containerType": input.ResolvedRequest.ResolvedService.ContainerType,
		}

		if input.ResolvedRequest.ResolvedFile != nil {
			operation.MetadataMap["file"] = map[string]any{
				"path":       input.ResolvedRequest.ResolvedFile.Path,
				"outputName": input.ResolvedRequest.ResolvedFile.OutputName,
			}
		}

		if input.ResolvedRequest.ResolvedTarget != nil {
			operation.MetadataMap["target"] = map[string]any{
				"directory": input.ResolvedRequest.ResolvedTarget.Directory,
			}
		}

		maps.Copy(operation.MetadataMap, input.ResolvedRequest.ResolvedMetadata)
	}

	return operation, nil
}

func BindTransferToOperation(operation *model.EbicsOperation, transferID int64) error {
	if operation == nil {
		return database.NewValidationError("the EBICS operation is missing")
	}

	if transferID <= 0 {
		return database.NewValidationError("the transfer ID must be greater than zero")
	}

	if !isPayloadOrder(operation.OrderType) {
		return database.NewValidationError("only payload EBICS operations can be linked to a transfer")
	}

	operation.TransferID = utils.NewNullInt64(transferID)

	return nil
}

func UpdateOperationOutcomeFromReturnCodes(
	operation *model.EbicsOperation,
	technicalCode,
	technicalMsg,
	businessCode,
	businessMsg string,
) error {
	if operation == nil {
		return database.NewValidationError("the EBICS operation is missing")
	}

	operation.TechnicalReturnCode = strings.TrimSpace(technicalCode)
	operation.TechnicalReturnMessage = strings.TrimSpace(technicalMsg)
	operation.BusinessReturnCode = strings.TrimSpace(businessCode)
	operation.BusinessReturnMessage = strings.TrimSpace(businessMsg)

	decision, err := DecideRetryPolicy(
		operation.OrderType,
		operation.TechnicalReturnCode,
		operation.BusinessReturnCode,
	)
	if err != nil {
		return fmt.Errorf("failed to derive the EBICS retry policy: %w", err)
	}

	operation.GatewayOutcome = decision.GatewayOutcome
	operation.RetryDecision = decision.RetryDecision
	operation.ManualActionRequired = decision.ManualActionRequired

	switch decision.GatewayOutcome {
	case model.EbicsGatewayOutcomeSuccessForRuntime(),
		model.EbicsGatewayOutcomeSuccessWithWarningForRuntime(),
		model.EbicsGatewayOutcomeEmptySuccessForRuntime():
		operation.Status = model.EbicsOperationStatusCompletedForRuntime()
	default:
		operation.Status = model.EbicsOperationStatusFailedForRuntime()
	}

	if decision.GatewayOutcome == model.EbicsGatewayOutcomeSuccessWithWarningForRuntime() {
		operation.Status = model.EbicsOperationStatusCompletedWithWarningsForRuntime()
		operation.Severity = model.EbicsOperationSeverityWarningForRuntime()
	} else if decision.GatewayOutcome == model.EbicsGatewayOutcomePendingBankForRuntime() {
		operation.Status = model.EbicsOperationStatusWaitingBankForRuntime()
		operation.Severity = model.EbicsOperationSeverityInfoForRuntime()
	} else if decision.ManualActionRequired {
		operation.Severity = model.EbicsOperationSeverityWarningForRuntime()
	} else if operation.Status == model.EbicsOperationStatusFailedForRuntime() {
		operation.Severity = model.EbicsOperationSeverityErrorForRuntime()
	}

	return nil
}

func nullIfPositive(value int64, fallback sql.NullInt64) sql.NullInt64 {
	if value > 0 {
		return sql.NullInt64{Int64: value, Valid: true}
	}

	return fallback
}
