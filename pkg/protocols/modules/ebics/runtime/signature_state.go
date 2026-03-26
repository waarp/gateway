package runtime

import (
	"errors"
	"strings"
)

const (
	signatureOrderHVE = "HVE"
	signatureOrderHVS = "HVS"

	SignatureStateNotApplicable      = "NOT_APPLICABLE"
	SignatureStateUnknown            = "SIGNATURES_UNKNOWN"
	SignatureStateWaiting            = "WAITING_SIGNATURES"
	SignatureStatePartiallyAvailable = "SIGNATURE_PARTIALLY_AVAILABLE"
	SignatureStateReady              = "SIGNATURE_READY"
	SignatureStateAdded              = "SIGNATURE_ADDED"
	SignatureStateCancelled          = "SIGNATURE_CANCELLED"
	SignatureStateRejected           = "SIGNATURE_REJECTED"
	SignatureStateInvalid            = "SIGNATURE_INVALID"
)

var errMissingEbicsOrderType = errors.New("the EBICS order type is missing")

// DeriveSignatureState derives a stable technical signature state from EBICS order metadata.
func DeriveSignatureState(orderType, technicalCode, businessCode string, metadata map[string]any) (string, error) {
	orderType = strings.ToUpper(strings.TrimSpace(orderType))
	technicalCode = strings.TrimSpace(technicalCode)
	businessCode = strings.TrimSpace(businessCode)

	if orderType == "" {
		return "", errMissingEbicsOrderType
	}

	if !IsSignatureOrder(orderType) {
		return SignatureStateNotApplicable, nil
	}

	if businessCode != "" && businessCode != "000000" {
		return SignatureStateRejected, nil
	}

	if derivedState := deriveMetadataSignatureState(metadata); derivedState != "" {
		return derivedState, nil
	}

	switch orderType {
	case signatureOrderHVE:
		if technicalCode == "" || technicalCode == "000000" {
			return SignatureStateAdded, nil
		}
	case signatureOrderHVS:
		if technicalCode == "" || technicalCode == "000000" {
			return SignatureStateCancelled, nil
		}
	}

	if readBoolMetadata(metadata, "signatureReady") {
		return SignatureStateReady, nil
	}

	expected, hasExpected := readIntMetadata(metadata, "signaturesExpected")
	available, hasAvailable := readIntMetadata(metadata, "signaturesAvailable")
	if hasExpected && hasAvailable {
		switch {
		case available <= 0:
			return SignatureStateWaiting, nil
		case available < expected:
			return SignatureStatePartiallyAvailable, nil
		default:
			return SignatureStateReady, nil
		}
	}

	if strings.HasPrefix(technicalCode, "09") {
		return SignatureStateInvalid, nil
	}

	return SignatureStateUnknown, nil
}

// IsSignatureOrder returns true when the EBICS order is related to signatures.
func IsSignatureOrder(orderType string) bool {
	switch strings.ToUpper(strings.TrimSpace(orderType)) {
	case "HVD", signatureOrderHVE, "HVT", "HVU", "HVZ", signatureOrderHVS:
		return true
	default:
		return false
	}
}

func deriveMetadataSignatureState(metadata map[string]any) string {
	if readBoolMetadata(metadata, "signatureInvalid") {
		return SignatureStateInvalid
	}

	state := strings.ToUpper(readStringMetadata(metadata, "signatureState"))
	switch state {
	case SignatureStateUnknown, SignatureStateWaiting, SignatureStatePartiallyAvailable,
		SignatureStateReady, SignatureStateAdded, SignatureStateCancelled,
		SignatureStateRejected, SignatureStateInvalid:
		return state
	default:
		return ""
	}
}

func readStringMetadata(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	value, ok := metadata[key]
	if !ok {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func readBoolMetadata(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}

	value, ok := metadata[key]
	if !ok {
		return false
	}

	typed, ok := value.(bool)

	return ok && typed
}

func readIntMetadata(metadata map[string]any, key string) (int, bool) {
	if metadata == nil {
		return 0, false
	}

	value, ok := metadata[key]
	if !ok {
		return 0, false
	}

	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}
