package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/rtn"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	RTNProcessingPlanManualOnly   = "MANUAL_ONLY"
	RTNProcessingPlanAutoPull     = "AUTO_PULL"
	RTNProcessingPlanAutoFiltered = "AUTO_PULL_FILTERED"
)

var (
	errMissingRTNEventStore           = errors.New("the RTN event store is missing")
	errMissingRTNEventOwner           = errors.New("the RTN event owner is missing")
	errInsufficientRTNIdempotenceData = errors.New(
		"the RTN event does not contain enough information to compute an idempotence key",
	)
)

// RTNIngestionResult captures the durable result of RTN ingestion.
type RTNIngestionResult struct {
	Event          *model.EbicsRTNEvent
	IsDuplicate    bool
	ProcessingPlan string
}

// RTNEventStore defines the persistence contract required by RTN ingestion.
type RTNEventStore interface {
	InsertRTNEvent(event *model.EbicsRTNEvent) error
	UpdateRTNEvent(event *model.EbicsRTNEvent) error
	GetRTNEventByIdempotenceKey(owner, key string) (*model.EbicsRTNEvent, error)
}

// IngestRTNEvent normalizes and persists an RTN event with durable idempotence.
func IngestRTNEvent(
	owner string,
	raw *rtn.RawEvent,
	idempotenceKey string,
	store RTNEventStore,
) (*RTNIngestionResult, error) {
	if store == nil {
		return nil, errMissingRTNEventStore
	}

	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil, errMissingRTNEventOwner
	}

	if raw == nil {
		return nil, errInsufficientRTNIdempotenceData
	}

	if idempotenceKey == "" {
		derivedKey, err := ComputeRTNIdempotenceKey(raw)
		if err != nil {
			return nil, err
		}

		idempotenceKey = derivedKey
	}

	existing, err := store.GetRTNEventByIdempotenceKey(owner, idempotenceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the RTN event by idempotence key: %w", err)
	}

	if existing != nil {
		return &RTNIngestionResult{
			Event:          existing,
			IsDuplicate:    true,
			ProcessingPlan: resolveRTNProcessingPlan(existing),
		}, nil
	}

	event := &model.EbicsRTNEvent{
		Owner:          owner,
		Source:         raw.Source,
		EventID:        raw.EventID,
		CorrelationID:  raw.CorrelationID,
		IdempotenceKey: idempotenceKey,
		OrderTypeHint:  readRawEventString(raw.Metadata, "orderTypeHint"),
		ProfileID:      readRawEventString(raw.Metadata, "profileID"),
		Status:         "RECEIVED",
		ReceivedAt:     raw.ReceivedAt,
		PayloadMap:     map[string]any{},
	}

	if raw.Metadata != nil {
		event.PayloadMap = raw.Metadata
	}

	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = raw.ReceivedAt
	}
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now().UTC()
	}

	if hostID, ok := readRawEventInt64(raw.Metadata, "ebicsHostID"); ok {
		event.EbicsHostID = utils.NewNullInt64(hostID)
	}

	if subscriberID, ok := readRawEventInt64(raw.Metadata, "ebicsSubscriberID"); ok {
		event.EbicsSubscriberID = utils.NewNullInt64(subscriberID)
	}

	insertErr := store.InsertRTNEvent(event)
	if insertErr != nil {
		return nil, fmt.Errorf("failed to insert the RTN event: %w", insertErr)
	}

	return &RTNIngestionResult{
		Event:          event,
		IsDuplicate:    false,
		ProcessingPlan: resolveRTNProcessingPlan(event),
	}, nil
}

// ComputeRTNIdempotenceKey computes a stable idempotence key for an RTN message.
func ComputeRTNIdempotenceKey(raw *rtn.RawEvent) (string, error) {
	if raw == nil {
		return "", errInsufficientRTNIdempotenceData
	}

	if len(raw.Payload) == 0 && strings.TrimSpace(raw.EventID) == "" && strings.TrimSpace(raw.CorrelationID) == "" {
		return "", errInsufficientRTNIdempotenceData
	}

	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(raw.Source),
		strings.TrimSpace(raw.EventID),
		strings.TrimSpace(raw.CorrelationID),
		string(raw.Payload),
	}, "|")))

	return hex.EncodeToString(sum[:]), nil
}

func resolveRTNProcessingPlan(event *model.EbicsRTNEvent) string {
	switch readPayloadString(event.PayloadMap, "autoPullPolicy") {
	case "AUTO":
		return RTNProcessingPlanAutoPull
	case "AUTO_FILTERED":
		return RTNProcessingPlanAutoFiltered
	default:
		return RTNProcessingPlanManualOnly
	}
}

func readRawEventString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	value, ok := metadata[key]
	if !ok {
		return ""
	}

	raw, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(raw)
}

func readRawEventInt64(metadata map[string]any, key string) (int64, bool) {
	if metadata == nil {
		return 0, false
	}

	value, ok := metadata[key]
	if !ok {
		return 0, false
	}

	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}

		return parsed, true
	default:
		return 0, false
	}
}

func readPayloadString(payload map[string]any, key string) string {
	return strings.ToUpper(readRawEventString(payload, key))
}
