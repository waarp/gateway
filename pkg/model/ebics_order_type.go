package model

import (
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsOrderHEV = "HEV"
	ebicsOrderINI = "INI"
	ebicsOrderHIA = "HIA"
	ebicsOrderH3K = "H3K"
	ebicsOrderHPB = "HPB"
	ebicsOrderPUB = "PUB"
	ebicsOrderHCA = "HCA"
	ebicsOrderHCS = "HCS"
	ebicsOrderHSA = "HSA"
	ebicsOrderSPR = "SPR"
	ebicsOrderHPD = "HPD"
	ebicsOrderHKD = "HKD"
	ebicsOrderHTD = "HTD"
	ebicsOrderHAA = "HAA"
	ebicsOrderHAC = "HAC"
	ebicsOrderHVD = "HVD"
	ebicsOrderHVU = "HVU"
	ebicsOrderHVZ = "HVZ"
	ebicsOrderHVT = "HVT"
	ebicsOrderHVE = "HVE"
	ebicsOrderHVS = "HVS"
)

// NormalizeEbicsOrderType returns the canonical Gateway EBICS order type.
func NormalizeEbicsOrderType(orderType string) string {
	normalized := strings.ToUpper(strings.TrimSpace(orderType))

	switch normalized {
	case ebicsPayloadOrderFUL:
		return ebicsPayloadOrderBTU
	case ebicsPayloadOrderFDL:
		return ebicsPayloadOrderBTD
	default:
		return normalized
	}
}

// IsEbicsOrderType reports whether the provided order type belongs to the supported EBICS family.
func IsEbicsOrderType(orderType string) bool {
	switch NormalizeEbicsOrderType(orderType) {
	case ebicsPayloadOrderBTU, ebicsPayloadOrderBTD,
		ebicsOrderHEV, ebicsOrderINI, ebicsOrderHIA, ebicsOrderH3K, ebicsOrderHPB,
		ebicsOrderPUB, ebicsOrderHCA, ebicsOrderHCS, ebicsOrderHSA, ebicsOrderSPR,
		ebicsOrderHPD, ebicsOrderHKD, ebicsOrderHTD, ebicsOrderHAA,
		ebicsOrderHAC, ebicsOrderHVD, ebicsOrderHVU, ebicsOrderHVZ, ebicsOrderHVT,
		ebicsOrderHVE, ebicsOrderHVS:
		return true
	default:
		return false
	}
}

func validateEbicsOrderType(orderType string) error {
	switch NormalizeEbicsOrderType(orderType) {
	case "":
		return database.NewValidationError("the EBICS order type is missing")
	default:
		if !IsEbicsOrderType(orderType) {
			return database.NewValidationErrorf("%q is not a supported EBICS order type", orderType)
		}

		return nil
	}
}

// ValidateEbicsOrderTypeForRuntime validates EBICS order types for non-model packages.
func ValidateEbicsOrderTypeForRuntime(orderType string) error {
	return validateEbicsOrderType(orderType)
}
