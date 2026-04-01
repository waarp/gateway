package model

import (
	"fmt"
	"maps"
	"reflect"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// EbicsTransferContext exposes EBICS execution metadata without relying on TransferInfo.
type EbicsTransferContext struct {
	OperationID     int64
	RTNEventID      int64
	TransactionID   string
	OrderType       string
	HostID          string
	PartnerID       string
	UserID          string
	RequestID       string
	CorrelationID   string
	ProtocolVersion string
	ProfileName     string
	EndpointURL     string
	RTNProviderName string
	RTNSource       string
	Service         map[string]any
}

// ToMap serializes the context as a JSON-friendly map.
func (c *EbicsTransferContext) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := map[string]any{
		"operationID":     c.OperationID,
		"rtnEventID":      c.RTNEventID,
		"transactionID":   c.TransactionID,
		"orderType":       c.OrderType,
		"hostID":          c.HostID,
		"partnerID":       c.PartnerID,
		"userID":          c.UserID,
		"requestID":       c.RequestID,
		"correlationID":   c.CorrelationID,
		"protocolVersion": c.ProtocolVersion,
		"profileName":     c.ProfileName,
		"endpointURL":     c.EndpointURL,
		"rtnProviderName": c.RTNProviderName,
		"rtnSource":       c.RTNSource,
	}

	if len(c.Service) > 0 {
		result["service"] = c.Service
	}

	for key, value := range result {
		if isZeroEbicsContextValue(value) {
			delete(result, key)
		}
	}

	return result
}

// LoadEbicsTransferContext resolves the EBICS context bound to a transfer.
func LoadEbicsTransferContext(
	db database.ReadAccess,
	transferID int64,
	_ map[string]any,
) (*EbicsTransferContext, error) {
	if isNilReadAccess(db) || transferID <= 0 {
		return nil, nil //nolint:nilnil // missing EBICS context is a valid outcome for non-EBICS transfers
	}

	operation, err := findEbicsOperationForTransfer(db, transferID)
	if err != nil {
		if database.IsNotFound(err) {
			return nil, nil //nolint:nilnil // missing EBICS context is a valid outcome for non-EBICS transfers
		}

		return nil, err
	}

	host := &EbicsHost{}
	if getErr := db.Get(host, "id=?", operation.EbicsHostID).Run(); getErr != nil {
		return nil, fmt.Errorf("load EBICS host %d for transfer %d context: %w",
			operation.EbicsHostID, transferID, getErr)
	}

	subscriber := &EbicsSubscriber{}
	if getErr := db.Get(subscriber, "id=?", operation.EbicsSubscriberID).Run(); getErr != nil {
		return nil, fmt.Errorf("load EBICS subscriber %d for transfer %d context: %w",
			operation.EbicsSubscriberID, transferID, getErr)
	}

	ctx := &EbicsTransferContext{
		OperationID:     operation.ID,
		TransactionID:   strings.TrimSpace(operation.TransactionID),
		OrderType:       strings.TrimSpace(operation.OrderType),
		HostID:          strings.TrimSpace(host.HostID),
		PartnerID:       strings.TrimSpace(subscriber.PartnerID),
		UserID:          strings.TrimSpace(subscriber.UserID),
		RequestID:       strings.TrimSpace(operation.RequestID),
		CorrelationID:   strings.TrimSpace(operation.CorrelationID),
		ProtocolVersion: strings.TrimSpace(operation.EbicsVersion),
		ProfileName:     strings.TrimSpace(readStringMapValue(operation.MetadataMap, "profileName")),
		EndpointURL:     strings.TrimSpace(readStringMapValue(operation.MetadataMap, "endpointURL")),
		RTNProviderName: strings.TrimSpace(readStringMapValue(operation.MetadataMap, "rtnProviderName")),
		RTNSource:       strings.TrimSpace(readStringMapValue(operation.MetadataMap, "rtnSource")),
		Service:         readMapMapValue(operation.MetadataMap, "service"),
	}

	if operation.RTNEventID.Valid {
		ctx.RTNEventID = operation.RTNEventID.Int64
	}

	return ctx, nil
}

func isNilReadAccess(db database.ReadAccess) bool {
	if db == nil {
		return true
	}

	value := reflect.ValueOf(db)
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		return value.IsNil()
	default:
		return false
	}
}

func findEbicsOperationForTransfer(
	db database.ReadAccess,
	transferID int64,
) (*EbicsOperation, error) {
	operation := &EbicsOperation{}
	if err := db.Get(
		operation,
		"owner=? AND transfer_id=?",
		conf.GlobalConfig.GatewayName,
		transferID,
	).Run(); err == nil {
		return operation, nil
	} else if !database.IsNotFound(err) {
		return nil, fmt.Errorf("load EBICS operation for transfer %d: %w", transferID, err)
	}

	var operations EbicsOperations
	if err := db.Select(&operations).
		Where("owner=? AND operation_type=?", conf.GlobalConfig.GatewayName, ebicsOperationTypePayload).
		Run(); err != nil {
		return nil, fmt.Errorf("load archived EBICS operations for transfer %d: %w", transferID, err)
	}

	for _, candidate := range operations {
		if readInt64MapValue(candidate.MetadataMap, "archivedTransferID") == transferID {
			return candidate, nil
		}
	}

	return nil, database.NewNotFoundError(&EbicsOperation{})
}

func readStringMapValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}

	raw, ok := values[key]
	if !ok {
		return ""
	}

	return fmt.Sprint(raw)
}

func readMapMapValue(values map[string]any, key string) map[string]any {
	if values == nil {
		return nil
	}

	raw, ok := values[key]
	if !ok {
		return nil
	}

	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	result := make(map[string]any, len(m))
	maps.Copy(result, m)

	return result
}

func readInt64MapValue(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}

	raw, ok := values[key]
	if !ok {
		return 0
	}

	switch value := raw.(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	case string:
		parsed, err := parseStringMapInt64(value)
		if err != nil {
			return 0
		}

		return parsed
	default:
		return 0
	}
}

func parseStringMapInt64(raw string) (int64, error) {
	var value int64
	_, err := fmt.Sscan(strings.TrimSpace(raw), &value)
	if err != nil {
		return 0, fmt.Errorf("parse int64 map value %q: %w", raw, err)
	}

	return value, nil
}

func isZeroEbicsContextValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case int64:
		return typed == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}
