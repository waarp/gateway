package model

import (
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsPayloadOrderBTU = "BTU"
	ebicsPayloadOrderBTD = "BTD"
	ebicsPayloadOrderFUL = "FUL"
	ebicsPayloadOrderFDL = "FDL"
)

// NormalizeEbicsPayloadOrderType returns the canonical Gateway payload order type.
func NormalizeEbicsPayloadOrderType(orderType string) string {
	switch strings.ToUpper(strings.TrimSpace(orderType)) {
	case ebicsPayloadOrderFUL:
		return ebicsPayloadOrderBTU
	case ebicsPayloadOrderFDL:
		return ebicsPayloadOrderBTD
	default:
		return strings.ToUpper(strings.TrimSpace(orderType))
	}
}

// IsEbicsPayloadOrderType reports whether the provided order type belongs to the payload family.
func IsEbicsPayloadOrderType(orderType string) bool {
	switch NormalizeEbicsPayloadOrderType(orderType) {
	case ebicsPayloadOrderBTU, ebicsPayloadOrderBTD:
		return true
	default:
		return false
	}
}

// IsEbicsPayloadDownloadOrder reports whether the canonical order is download-oriented.
func IsEbicsPayloadDownloadOrder(orderType string) bool {
	return NormalizeEbicsPayloadOrderType(orderType) == ebicsPayloadOrderBTD
}

func validateEbicsPayloadOrderType(orderType string) error {
	switch NormalizeEbicsPayloadOrderType(orderType) {
	case ebicsPayloadOrderBTU, ebicsPayloadOrderBTD:
		return nil
	case "":
		return database.NewValidationError("the EBICS payload order type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS payload order type", orderType)
	}
}

// ValidateEbicsPayloadOrderTypeForRuntime validates payload order types for non-model packages.
func ValidateEbicsPayloadOrderTypeForRuntime(orderType string) error {
	return validateEbicsPayloadOrderType(orderType)
}
