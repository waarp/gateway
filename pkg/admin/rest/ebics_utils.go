package rest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
)

func parseRESTInt64Param(r *http.Request, name, label string) (int64, error) {
	raw, ok := mux.Vars(r)[name]
	if !ok || strings.TrimSpace(raw) == "" {
		return 0, notFoundf("missing %s identifier", label)
	}

	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, badRequestf("invalid %s identifier %q", label, raw)
	}

	return id, nil
}

func ptrTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	utc := value.UTC()

	return &utc
}

func ptrInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}

	v := value.Int64

	return &v
}

func metadataInt64(metadata map[string]any, key string) *int64 {
	if metadata == nil {
		return nil
	}

	raw, ok := metadata[key]
	if !ok {
		return nil
	}

	switch value := raw.(type) {
	case int:
		v := int64(value)
		return &v
	case int32:
		v := int64(value)
		return &v
	case int64:
		return &value
	case float32:
		v := int64(value)
		if float32(v) == value {
			return &v
		}
	case float64:
		v := int64(value)
		if float64(v) == value {
			return &v
		}
	case json.Number:
		if v, err := value.Int64(); err == nil {
			return &v
		}
	case string:
		if v, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
			return &v
		}
	}

	return nil
}

func operationTransferID(operation *model.EbicsOperation) *int64 {
	if transferID := ptrInt64(operation.TransferID); transferID != nil {
		return transferID
	}

	return metadataInt64(operation.MetadataMap, "archivedTransferID")
}

func getDBEbicsPayloadProfile(r *http.Request, db *database.DB) (*model.EbicsPayloadProfile, error) {
	name, ok := mux.Vars(r)["payload_profile"]
	if !ok || strings.TrimSpace(name) == "" {
		return nil, notFound("missing EBICS payload profile name")
	}

	profile := &model.EbicsPayloadProfile{}
	if err := db.Get(profile, "name=?", strings.TrimSpace(name)).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("EBICS payload profile %q not found", name)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS payload profile %q: %w", name, err)
	}

	return profile, nil
}

func getDBEbicsContractView(r *http.Request, db *database.DB) (*model.EbicsContractView, error) {
	id, err := parseRESTInt64Param(r, "contract_view", "EBICS contract view")
	if err != nil {
		return nil, err
	}

	view := &model.EbicsContractView{}
	if err = db.Get(view, "id=?", id).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("EBICS contract view %d not found", id)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS contract view %d: %w", id, err)
	}

	return view, nil
}

func getDBEbicsOperation(r *http.Request, db *database.DB) (*model.EbicsOperation, error) {
	id, err := parseRESTInt64Param(r, "ebics_operation", "EBICS operation")
	if err != nil {
		return nil, err
	}

	operation := &model.EbicsOperation{}
	if err = db.Get(operation, "id=?", id).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("EBICS operation %d not found", id)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS operation %d: %w", id, err)
	}

	return operation, nil
}

func getDBEbicsTransaction(r *http.Request, db *database.DB) (*model.EbicsTransaction, error) {
	id, err := parseRESTInt64Param(r, "ebics_transaction", "EBICS transaction")
	if err != nil {
		return nil, err
	}

	transaction := &model.EbicsTransaction{}
	if err = db.Get(transaction, "id=?", id).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("EBICS transaction %d not found", id)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS transaction %d: %w", id, err)
	}

	return transaction, nil
}

func getDBEbicsTransactionSegment(r *http.Request, db *database.DB) (*model.EbicsTransactionSegment, error) {
	transaction, err := getDBEbicsTransaction(r, db)
	if err != nil {
		return nil, err
	}

	segmentNumber, err := parseRESTInt64Param(r, "segment_number", "EBICS transaction segment")
	if err != nil {
		return nil, err
	}

	segment := &model.EbicsTransactionSegment{}
	if err = db.Get(
		segment,
		"ebics_transaction_id=? AND segment_number=?",
		transaction.ID,
		segmentNumber,
	).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf(
				"EBICS transaction segment %d not found for transaction %d",
				segmentNumber,
				transaction.ID,
			)
		}

		return nil, fmt.Errorf(
			"failed to retrieve EBICS transaction segment %d for transaction %d: %w",
			segmentNumber,
			transaction.ID,
			err,
		)
	}

	return segment, nil
}

func getDBEbicsKeyLifecycle(r *http.Request, db *database.DB) (*model.EbicsKeyLifecycle, error) {
	id, err := parseRESTInt64Param(r, "ebics_key_lifecycle", "EBICS key lifecycle")
	if err != nil {
		return nil, err
	}

	lifecycle := &model.EbicsKeyLifecycle{}
	if err = db.Get(lifecycle, "id=?", id).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("EBICS key lifecycle %d not found", id)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS key lifecycle %d: %w", id, err)
	}

	return lifecycle, nil
}

func getDBEbicsInitialization(r *http.Request, db *database.DB) (*model.EbicsInitializationWorkflow, error) {
	id, err := parseRESTInt64Param(r, "ebics_initialization", "EBICS initialization workflow")
	if err != nil {
		return nil, err
	}

	workflow := &model.EbicsInitializationWorkflow{}
	if err = db.Get(workflow, "id=?", id).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("EBICS initialization workflow %d not found", id)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS initialization workflow %d: %w", id, err)
	}

	return workflow, nil
}

func getDBEbicsRTNEvent(r *http.Request, db *database.DB) (*model.EbicsRTNEvent, error) {
	id, err := parseRESTInt64Param(r, "ebics_rtn_event", "EBICS RTN event")
	if err != nil {
		return nil, err
	}

	event := &model.EbicsRTNEvent{}
	if err = db.Get(event, "id=?", id).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("EBICS RTN event %d not found", id)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS RTN event %d: %w", id, err)
	}

	return event, nil
}

func getDBEbicsRTNProvider(r *http.Request, db *database.DB) (*model.EbicsRTNProvider, error) {
	name, ok := mux.Vars(r)["ebics_rtn_provider"]
	if !ok || strings.TrimSpace(name) == "" {
		return nil, notFound("missing EBICS RTN provider name")
	}

	provider := &model.EbicsRTNProvider{}
	if err := db.Get(provider, "name=?", strings.TrimSpace(name)).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("EBICS RTN provider %q not found", name)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS RTN provider %q: %w", name, err)
	}

	return provider, nil
}

// DBEbicsPayloadProfileToREST transforms an EBICS payload profile into its REST representation.
func DBEbicsPayloadProfileToREST(
	db database.ReadAccess,
	profile *model.EbicsPayloadProfile,
) (*api.OutEbicsPayloadProfile, error) {
	restProfile := &api.OutEbicsPayloadProfile{
		ID:                     profile.ID,
		Name:                   profile.Name,
		Label:                  profile.Label,
		Description:            profile.Description,
		OrderType:              profile.OrderType,
		Direction:              profile.Direction,
		ServiceName:            profile.ServiceName,
		ServiceOption:          profile.ServiceOption,
		Scope:                  profile.Scope,
		MsgName:                profile.MsgName,
		ContainerType:          profile.ContainerType,
		DefaultTargetDirectory: profile.DefaultTargetDirectory,
		RequiresDeclaredAmount: profile.RequiresDeclaredAmount,
		DefaultCurrency:        profile.DefaultCurrency,
		AllowedExtensions:      profile.AllowedExtensionsList,
		FilenamePattern:        profile.FilenamePattern,
		StrictContractCheck:    profile.StrictContractCheck,
		IsEnabled:              profile.IsEnabled,
		Metadata:               profile.MetadataMap,
	}

	if profile.DefaultRuleID.Valid {
		rule := &model.Rule{}
		if err := db.Get(rule, "id=?", profile.DefaultRuleID.Int64).Owner().Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve default Gateway rule for payload profile %q: %w", profile.Name, err)
		}

		restProfile.DefaultRule = rule.Name
	}

	return restProfile, nil
}

// DBEbicsContractViewToREST transforms an EBICS contract view into its REST representation.
func DBEbicsContractViewToREST(
	db database.ReadAccess,
	view *model.EbicsContractView,
) (*api.OutEbicsContractView, error) {
	host := &model.EbicsHost{}
	if err := db.Get(host, "id=?", view.EbicsHostID).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS host for contract view %d: %w", view.ID, err)
	}

	restView := &api.OutEbicsContractView{
		ID:              view.ID,
		HostID:          host.HostID,
		SourceOrderType: view.SourceOrderType,
		VersionTag:      view.VersionTag,
		Status:          view.Status,
		FetchedAt:       view.FetchedAt.UTC(),
	}

	if view.EbicsSubscriberID.Valid {
		subscriber := &model.EbicsSubscriber{}
		if err := db.Get(subscriber, "id=?", view.EbicsSubscriberID.Int64).Owner().Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve EBICS subscriber for contract view %d: %w", view.ID, err)
		}

		restView.PartnerID = subscriber.PartnerID
		restView.UserID = subscriber.UserID
	}

	return restView, nil
}

// DBEbicsContractViewItemToREST transforms an EBICS contract view item into its REST representation.
func DBEbicsContractViewItemToREST(item *model.EbicsContractViewItem) *api.OutEbicsContractViewItem {
	return &api.OutEbicsContractViewItem{
		ID:                 item.ID,
		ItemType:           item.ItemType,
		ItemKey:            item.ItemKey,
		OrderType:          item.OrderType,
		ServiceName:        item.ServiceName,
		ServiceOption:      item.ServiceOption,
		Scope:              item.Scope,
		MsgName:            item.MsgName,
		ContainerType:      item.ContainerType,
		AdminOrderType:     item.AdminOrderType,
		AuthorisationLevel: item.AuthorisationLevel,
		AccountID:          item.AccountID,
		MaxAmountValue:     item.MaxAmountValue,
		MaxAmountCurrency:  item.MaxAmountCurrency,
		IsEnabled:          item.IsEnabled,
	}
}

// DBEbicsOperationToREST transforms an EBICS operation into its REST representation.
func DBEbicsOperationToREST(operation *model.EbicsOperation) (*api.OutEbicsOperation, error) {
	signatureState, err := ebicsruntime.DeriveSignatureState(
		operation.OrderType,
		operation.TechnicalReturnCode,
		operation.BusinessReturnCode,
		operation.MetadataMap,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive EBICS signature state for operation %d: %w", operation.ID, err)
	}

	return &api.OutEbicsOperation{
		ID:                     operation.ID,
		OperationType:          operation.OperationType,
		OrderType:              operation.OrderType,
		SignatureState:         signatureState,
		Direction:              operation.Direction,
		TransportMode:          operation.TransportMode,
		Status:                 operation.Status,
		Severity:               operation.Severity,
		TransactionID:          operation.TransactionID,
		RequestID:              operation.RequestID,
		CorrelationID:          operation.CorrelationID,
		TechnicalReturnCode:    operation.TechnicalReturnCode,
		TechnicalReturnMessage: operation.TechnicalReturnMessage,
		BusinessReturnCode:     operation.BusinessReturnCode,
		BusinessReturnMessage:  operation.BusinessReturnMessage,
		GatewayOutcome:         operation.GatewayOutcome,
		RetryDecision:          operation.RetryDecision,
		ManualActionRequired:   operation.ManualActionRequired,
		TransferID:             operationTransferID(operation),
		Metadata:               operation.MetadataMap,
	}, nil
}

// DBEbicsOperationDetailToREST transforms an EBICS operation and its related
// technical objects into a detailed REST representation.
func DBEbicsOperationDetailToREST(
	db database.ReadAccess,
	operation *model.EbicsOperation,
) (*api.OutEbicsOperationDetail, error) {
	restOperation, err := DBEbicsOperationToREST(operation)
	if err != nil {
		return nil, err
	}

	host := &model.EbicsHost{}
	if err = db.Get(host, "id=?", operation.EbicsHostID).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS host for operation %d: %w", operation.ID, err)
	}

	detail := &api.OutEbicsOperationDetail{
		Operation:  restOperation,
		HostID:     host.HostID,
		StartedAt:  ptrTime(operation.StartedAt),
		FinishedAt: ptrTime(operation.FinishedAt),
	}

	subscriber := &model.EbicsSubscriber{}
	if err = db.Get(subscriber, "id=?", operation.EbicsSubscriberID).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS subscriber for operation %d: %w", operation.ID, err)
	}
	detail.PartnerID = subscriber.PartnerID
	detail.UserID = subscriber.UserID

	if transferID := operationTransferID(operation); transferID != nil ||
		operation.ContractViewID.Valid || operation.RTNEventID.Valid {
		detail.Links = &api.OutEbicsOperationLinks{
			TransferID:     operationTransferID(operation),
			ContractViewID: ptrInt64(operation.ContractViewID),
			RTNEventID:     ptrInt64(operation.RTNEventID),
		}
	}

	transaction := &model.EbicsTransaction{}
	if err = db.Get(transaction, "ebics_operation_id=?", operation.ID).Owner().Run(); err == nil {
		detail.Transaction = DBEbicsTransactionToREST(transaction)

		var segments model.EbicsTransactionSegments
		if err = db.Select(&segments).Owner().Where("ebics_transaction_id=?", transaction.ID).
			OrderBy("segment_number", true).Run(); err != nil {
			return nil, fmt.Errorf(
				"failed to retrieve EBICS transaction segments for operation %d: %w",
				operation.ID,
				err,
			)
		}

		detail.Segments = make([]*api.OutEbicsTransactionSegment, len(segments))
		for i, segment := range segments {
			detail.Segments[i] = DBEbicsTransactionSegmentToREST(segment)
		}
	} else if !database.IsNotFound(err) {
		return nil, fmt.Errorf("failed to retrieve EBICS transaction for operation %d: %w", operation.ID, err)
	}

	return detail, nil
}

// DBEbicsTransactionToREST transforms an EBICS transaction into its REST representation.
func DBEbicsTransactionToREST(transaction *model.EbicsTransaction) *api.OutEbicsTransaction {
	return &api.OutEbicsTransaction{
		ID:             transaction.ID,
		TransactionID:  transaction.TransactionID,
		OrderType:      transaction.OrderType,
		Status:         transaction.Status,
		Direction:      transaction.Direction,
		SegmentCount:   transaction.SegmentCount,
		CurrentSegment: transaction.CurrentSegment,
		TotalSize:      transaction.TotalSize,
		TransferID:     ptrInt64(transaction.TransferID),
	}
}

// DBEbicsTransactionDetailToREST transforms an EBICS transaction and its
// related subscriber/operation references into a detailed REST representation.
func DBEbicsTransactionDetailToREST(
	db database.ReadAccess,
	transaction *model.EbicsTransaction,
	segments []*api.OutEbicsTransactionSegment,
) (*api.OutEbicsTransactionDetail, error) {
	host := &model.EbicsHost{}
	if err := db.Get(host, "id=?", transaction.EbicsHostID).Owner().Run(); err != nil {
		return nil, fmt.Errorf(
			"failed to retrieve EBICS host for transaction %d: %w",
			transaction.ID,
			err,
		)
	}

	subscriber := &model.EbicsSubscriber{}
	if err := db.Get(subscriber, "id=?", transaction.EbicsSubscriberID).Owner().Run(); err != nil {
		return nil, fmt.Errorf(
			"failed to retrieve EBICS subscriber for transaction %d: %w",
			transaction.ID,
			err,
		)
	}

	detail := &api.OutEbicsTransactionDetail{
		Transaction: DBEbicsTransactionToREST(transaction),
		HostID:      host.HostID,
		PartnerID:   subscriber.PartnerID,
		UserID:      subscriber.UserID,
		Segments:    segments,
	}

	if transaction.EbicsOperationID.Valid {
		operation := &model.EbicsOperation{}
		if err := db.Get(operation, "id=?", transaction.EbicsOperationID.Int64).Owner().Run(); err != nil {
			return nil, fmt.Errorf(
				"failed to retrieve EBICS operation %d for transaction %d: %w",
				transaction.EbicsOperationID.Int64,
				transaction.ID,
				err,
			)
		}

		detail.RequestID = operation.RequestID
		detail.CorrelationID = operation.CorrelationID
	}

	return detail, nil
}

// DBEbicsTransactionSegmentToREST transforms an EBICS transaction segment into its REST representation.
func DBEbicsTransactionSegmentToREST(segment *model.EbicsTransactionSegment) *api.OutEbicsTransactionSegment {
	return &api.OutEbicsTransactionSegment{
		ID:               segment.ID,
		SegmentNumber:    segment.SegmentNumber,
		SegmentStatus:    segment.SegmentStatus,
		PayloadSize:      segment.PayloadSize,
		Checksum:         segment.Checksum,
		StoredPayloadRef: segment.StoredPayloadRef,
	}
}

// DBEbicsKeyLifecycleToREST transforms an EBICS key lifecycle into its REST representation.
func DBEbicsKeyLifecycleToREST(lifecycle *model.EbicsKeyLifecycle) *api.OutEbicsKeyLifecycle {
	return &api.OutEbicsKeyLifecycle{
		ID:                  lifecycle.ID,
		KeyUsage:            lifecycle.KeyUsage,
		RotationType:        lifecycle.RotationType,
		CoordinationID:      lifecycle.CoordinationID,
		Status:              lifecycle.Status,
		CurrentCredentialID: lifecycle.CurrentCredentialID,
		NextCredentialID:    ptrInt64(lifecycle.NextCredentialID),
		TriggerOperationID:  ptrInt64(lifecycle.TriggerOperationID),
		LastOperationID:     ptrInt64(lifecycle.LastOperationID),
		RequestedAt:         ptrTime(lifecycle.RequestedAt),
		SentAt:              ptrTime(lifecycle.SentAt),
		ActivatedAt:         ptrTime(lifecycle.ActivatedAt),
		RetiredAt:           ptrTime(lifecycle.RetiredAt),
		Operator:            lifecycle.Operator,
		Reason:              lifecycle.Reason,
		Evidence:            lifecycle.EvidenceMap,
	}
}

// DBEbicsInitializationToREST transforms an EBICS initialization workflow into its REST representation.
func DBEbicsInitializationToREST(workflow *model.EbicsInitializationWorkflow) *api.OutEbicsInitializationWorkflow {
	return &api.OutEbicsInitializationWorkflow{
		ID:                workflow.ID,
		Status:            workflow.Status,
		CurrentStep:       workflow.CurrentStep,
		IniOperationID:    ptrInt64(workflow.IniOperationID),
		HiaOperationID:    ptrInt64(workflow.HiaOperationID),
		H3KOperationID:    ptrInt64(workflow.H3KOperationID),
		LetterGeneratedAt: ptrTime(workflow.LetterGeneratedAt),
		LetterConfirmedAt: ptrTime(workflow.LetterConfirmedAt),
		BankActivationAt:  ptrTime(workflow.BankActivationAt),
		Operator:          workflow.Operator,
		Reason:            workflow.Reason,
		BankFeedback:      workflow.BankFeedback,
		Evidence:          workflow.EvidenceMap,
	}
}

// DBEbicsRTNEventToREST transforms an RTN event into its REST representation.
func DBEbicsRTNEventToREST(event *model.EbicsRTNEvent) *api.OutEbicsRTNEvent {
	operatorAction := ""
	operatorReason := ""
	var operatorMetadata map[string]any
	if event.PayloadMap != nil {
		if value, ok := event.PayloadMap["lastOperatorAction"].(string); ok {
			operatorAction = value
		}
		if value, ok := event.PayloadMap["lastOperatorReason"].(string); ok {
			operatorReason = value
		}
		if value, ok := event.PayloadMap["lastOperatorMetadata"].(map[string]any); ok {
			operatorMetadata = value
		}
	}

	return &api.OutEbicsRTNEvent{
		ID:               event.ID,
		Source:           event.Source,
		EventID:          event.EventID,
		CorrelationID:    event.CorrelationID,
		IdempotenceKey:   event.IdempotenceKey,
		OrderTypeHint:    event.OrderTypeHint,
		ProfileID:        event.ProfileID,
		Status:           event.Status,
		Attempts:         event.Attempts,
		NextRetryAt:      ptrTime(event.NextRetryAt),
		ReceivedAt:       event.ReceivedAt.UTC(),
		ProcessedAt:      ptrTime(event.ProcessedAt),
		LastError:        event.LastError,
		OperatorAction:   operatorAction,
		OperatorReason:   operatorReason,
		OperatorMetadata: operatorMetadata,
	}
}

// DBEbicsRTNProviderToREST transforms an RTN provider into its REST representation.
func DBEbicsRTNProviderToREST(provider *model.EbicsRTNProvider) *api.OutEbicsRTNProvider {
	return &api.OutEbicsRTNProvider{
		ID:               provider.ID,
		Name:             provider.Name,
		Transport:        provider.Transport,
		Enabled:          provider.Enabled,
		SubscriberID:     provider.EbicsSubscriberID,
		AutoPullPolicy:   provider.AutoPullPolicy,
		LastConnectionAt: ptrTime(provider.LastConnectionAt),
		LastError:        provider.LastError,
	}
}
