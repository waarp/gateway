package runtime

import (
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// RetryPolicyDecision captures the Gateway decision derived from EBICS return codes.
type RetryPolicyDecision struct {
	GatewayOutcome       string
	RetryDecision        string
	ManualActionRequired bool
	Message              string
}

// DecideRetryPolicy derives a Gateway retry policy from EBICS technical and business return codes.
func DecideRetryPolicy(orderType, technicalCode, businessCode string) (*RetryPolicyDecision, error) {
	orderType = model.NormalizeEbicsPayloadOrderType(orderType)
	technicalCode = strings.TrimSpace(technicalCode)
	businessCode = strings.TrimSpace(businessCode)

	if orderType == "" {
		return nil, database.NewValidationError("the EBICS order type is missing")
	}

	if businessCode != "" && businessCode != "000000" {
		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeBusinessRejectedForRuntime(),
			RetryDecision:        model.EbicsRetryDecisionNoRetryForRuntime(),
			ManualActionRequired: true,
			Message:              "business EBICS rejection, no automatic retry allowed",
		}, nil
	}

	switch technicalCode {
	case "", "000000":
		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeSuccessForRuntime(),
			RetryDecision:        model.EbicsRetryDecisionNoRetryForRuntime(),
			ManualActionRequired: false,
			Message:              "successful EBICS outcome",
		}, nil
	case "090005":
		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeEmptySuccessForRuntime(),
			RetryDecision:        model.EbicsRetryDecisionNoRetryForRuntime(),
			ManualActionRequired: false,
			Message:              "successful EBICS request with no download data available",
		}, nil
	case "061101":
		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeTechnicalRetryableFailureForRuntime(),
			RetryDecision:        model.EbicsRetryDecisionRecoveryRequiredForRuntime(),
			ManualActionRequired: false,
			Message:              "EBICS recovery synchronization is required",
		}, nil
	case "091103", "091105":
		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeTechnicalFatalFailureForRuntime(),
			RetryDecision:        model.EbicsRetryDecisionManualReplayOnlyForRuntime(),
			ManualActionRequired: true,
			Message:              "EBICS replay/recovery issue requires a controlled manual action",
		}, nil
	case "091008":
		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeManualConfirmationRequiredForRuntime(),
			RetryDecision:        model.EbicsRetryDecisionManualConfirmationOnlyForRuntime(),
			ManualActionRequired: true,
			Message:              "EBICS bank public key update required",
		}, nil
	}

	if strings.HasPrefix(technicalCode, "06") {
		retryDecision := model.EbicsRetryDecisionAutoRetryAllowedForRuntime()
		if isManualReplayOrder(orderType) {
			retryDecision = model.EbicsRetryDecisionManualReplayOnlyForRuntime()
		}

		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeTechnicalRetryableFailureForRuntime(),
			RetryDecision:        retryDecision,
			ManualActionRequired: retryDecision != model.EbicsRetryDecisionAutoRetryAllowedForRuntime(),
			Message:              "retryable technical EBICS failure",
		}, nil
	}

	if strings.HasPrefix(technicalCode, "09") {
		retryDecision := model.EbicsRetryDecisionNoRetryForRuntime()
		manualAction := false
		if isManualReplayOrder(orderType) {
			retryDecision = model.EbicsRetryDecisionManualReplayOnlyForRuntime()
			manualAction = true
		}

		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeTechnicalFatalFailureForRuntime(),
			RetryDecision:        retryDecision,
			ManualActionRequired: manualAction,
			Message:              "fatal technical EBICS failure",
		}, nil
	}

	if strings.HasPrefix(technicalCode, "01") || strings.HasPrefix(technicalCode, "03") {
		return &RetryPolicyDecision{
			GatewayOutcome:       model.EbicsGatewayOutcomeSuccessWithWarningForRuntime(),
			RetryDecision:        model.EbicsRetryDecisionNoRetryForRuntime(),
			ManualActionRequired: false,
			Message:              "successful EBICS execution with warning",
		}, nil
	}

	return &RetryPolicyDecision{
		GatewayOutcome:       model.EbicsGatewayOutcomePendingBankForRuntime(),
		RetryDecision:        model.EbicsRetryDecisionNoRetryForRuntime(),
		ManualActionRequired: false,
		Message:              "EBICS outcome not yet classifiable, waiting for bank-side completion",
	}, nil
}

func isManualReplayOrder(orderType string) bool {
	switch strings.ToUpper(strings.TrimSpace(orderType)) {
	case "INI", "HIA", "H3K", "PUB", "HCA", "HCS", "HSA", "SPR", "HVE", "HVS":
		return true
	default:
		return false
	}
}

func isPayloadOrder(orderType string) bool {
	return model.IsEbicsPayloadOrderType(orderType)
}
