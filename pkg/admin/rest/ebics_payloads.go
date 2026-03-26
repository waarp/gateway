package rest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
)

const (
	payloadStatusPlanned         = "PLANNED"
	payloadStatusReady           = "READY"
	payloadRetryDecisionAuto     = "AUTO_RETRY_ALLOWED"
	payloadRetryDecisionManual   = "MANUAL_REPLAY_ONLY"
	payloadRetryDecisionRecovery = "RECOVERY_REQUIRED"
	payloadOperationType         = "REPORTING"
)

type restPayloadProfileResolver struct {
	db *database.DB
}

func (r *restPayloadProfileResolver) GetPayloadProfile(owner, name string) (*model.EbicsPayloadProfile, error) {
	profile := &model.EbicsPayloadProfile{}
	if err := r.db.Get(profile, "name=? AND owner=?", name, owner).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, database.NewValidationErrorf("the EBICS payload profile %q does not exist", name)
		}

		return nil, fmt.Errorf("failed to retrieve EBICS payload profile %q: %w", name, err)
	}

	return profile, nil
}

type restContractViewResolver struct {
	db *database.DB
}

func (r *restContractViewResolver) GetActiveContractView(
	owner, hostID, partnerID, userID string,
) (*model.EbicsContractView, []model.EbicsContractViewItem, error) {
	host := &model.EbicsHost{}
	if err := r.db.Get(host, "host_id=? AND owner=?", hostID, owner).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, database.NewValidationErrorf("the EBICS host %q does not exist", hostID)
		}

		return nil, nil, fmt.Errorf("failed to retrieve EBICS host %q: %w", hostID, err)
	}

	subscriber := &model.EbicsSubscriber{}
	if err := r.db.Get(
		subscriber,
		"ebics_host_id=? AND partner_id=? AND user_id=? AND owner=?",
		host.ID,
		partnerID,
		userID,
		owner,
	).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, database.NewValidationErrorf(
				"the EBICS subscriber %q/%q does not exist for host %q",
				partnerID,
				userID,
				hostID,
			)
		}

		return nil, nil, fmt.Errorf(
			"failed to retrieve EBICS subscriber %q/%q for host %q: %w",
			partnerID,
			userID,
			hostID,
			err,
		)
	}

	view := &model.EbicsContractView{}
	if err := r.db.Get(
		view,
		"ebics_host_id=? AND ebics_subscriber_id=? AND status=? AND owner=?",
		host.ID,
		subscriber.ID,
		"ACTIVE",
		owner,
	).OrderBy("fetched_at", false).Run(); err != nil {
		if database.IsNotFound(err) {
			return &model.EbicsContractView{}, []model.EbicsContractViewItem{}, nil
		}

		return nil, nil, fmt.Errorf(
			"failed to retrieve active EBICS contract view for subscriber %q/%q: %w",
			partnerID,
			userID,
			err,
		)
	}

	var items model.EbicsContractViewItems
	if err := r.db.Select(&items).Owner().Where("contract_view_id=?", view.ID).Run(); err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve EBICS contract view items: %w", err)
	}

	return view, derefContractItems(items), nil
}

func derefContractItems(items model.EbicsContractViewItems) []model.EbicsContractViewItem {
	out := make([]model.EbicsContractViewItem, len(items))
	for i, item := range items {
		out[i] = *item
	}

	return out
}

func resolveEbicsSubscriberIDs(
	db *database.DB,
	subscriberRef api.InSubscriberRef,
) (*model.EbicsHost, *model.EbicsSubscriber, error) {
	host := &model.EbicsHost{}
	if err := db.Get(host, "host_id=?", strings.TrimSpace(subscriberRef.HostID)).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, notFoundf("EBICS host %q not found", subscriberRef.HostID)
		}

		return nil, nil, fmt.Errorf("failed to retrieve EBICS host %q: %w", subscriberRef.HostID, err)
	}

	subscriber := &model.EbicsSubscriber{}
	if err := db.Get(
		subscriber,
		"ebics_host_id=? AND partner_id=? AND user_id=?",
		host.ID,
		strings.TrimSpace(subscriberRef.PartnerID),
		strings.TrimSpace(subscriberRef.UserID),
	).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, notFoundf(
				"EBICS subscriber %q/%q not found for host %q",
				subscriberRef.PartnerID,
				subscriberRef.UserID,
				subscriberRef.HostID,
			)
		}

		return nil, nil, fmt.Errorf(
			"failed to retrieve EBICS subscriber %q/%q for host %q: %w",
			subscriberRef.PartnerID,
			subscriberRef.UserID,
			subscriberRef.HostID,
			err,
		)
	}

	return host, subscriber, nil
}

func payloadRequestToRuntime(input *api.InEbicsPayloadRequest) *ebicsruntime.PayloadRequestInput {
	request := &ebicsruntime.PayloadRequestInput{
		ProfileName: input.Profile,
		RuleName:    input.Rule,
		Subscriber: ebicsruntime.PayloadSubscriberRef{
			HostID:    input.Subscriber.HostID,
			PartnerID: input.Subscriber.PartnerID,
			UserID:    input.Subscriber.UserID,
		},
		Metadata: input.Metadata,
	}

	if input.File != nil {
		request.File = &ebicsruntime.PayloadFileRef{
			Path:       input.File.Path,
			OutputName: input.File.OutputName,
		}
	}

	if input.Target != nil {
		request.Target = &ebicsruntime.PayloadTargetRef{
			Directory: input.Target.Directory,
		}
	}

	if input.Service != nil {
		request.Service = &ebicsruntime.PayloadServiceRef{
			OrderType:     input.Service.OrderType,
			ServiceName:   input.Service.ServiceName,
			ServiceOption: input.Service.ServiceOption,
			Scope:         input.Service.Scope,
			MsgName:       input.Service.MsgName,
			ContainerType: input.Service.ContainerType,
		}
	}

	return request
}

func validatePayloadContractResult(
	resolved *ebicsruntime.ResolvedPayloadRequest,
	result *ebicsruntime.ContractValidationResult,
) error {
	if result.ContractViewID == 0 && len(result.MatchedItems) == 0 {
		return badRequest("no active EBICS contract view found for the selected subscriber")
	}

	switch result.Status {
	case "MATCHED":
		return nil
	case "NO_ACTIVE_CONTRACT":
		return badRequest("no active EBICS contract view found for the selected subscriber")
	case "NO_MATCHING_ITEM":
		profileName := resolved.ProfileName
		if profileName == "" {
			profileName = "<free-input>"
		}

		return badRequestf(
			"the resolved EBICS payload request for profile %q does not match the active contract",
			profileName,
		)
	default:
		return badRequestf("unsupported EBICS contract validation status %q", result.Status)
	}
}

func derivePayloadOperationType() string {
	return payloadOperationType
}

func derivePayloadDirection(orderType string) string {
	switch model.NormalizeEbicsPayloadOrderType(orderType) {
	case "BTD":
		return "INBOUND"
	default:
		return "OUTBOUND"
	}
}

func resolvePayloadCorrelationID(metadata map[string]any) string {
	if metadata != nil {
		if value, ok := metadata["correlationId"]; ok {
			correlationID := strings.TrimSpace(fmt.Sprint(value))
			if correlationID != "" {
				return correlationID
			}
		}

		if value, ok := metadata["correlationID"]; ok {
			correlationID := strings.TrimSpace(fmt.Sprint(value))
			if correlationID != "" {
				return correlationID
			}
		}
	}

	return uuid.NewString()
}

func payloadSubmissionResponse(operation *model.EbicsOperation) *api.OutEbicsPayloadSubmission {
	response := &api.OutEbicsPayloadSubmission{
		OperationID:   operation.ID,
		OrderType:     operation.OrderType,
		Status:        operation.Status,
		CorrelationID: operation.CorrelationID,
	}

	response.TransferID = ptrInt64(operation.TransferID)
	response.ContractViewID = ptrInt64(operation.ContractViewID)
	if value, ok := operation.MetadataMap["contractItemIDs"].([]int64); ok {
		response.MatchedContractItemIDs = value
	}

	return response
}

func submitEbicsPayload(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := &api.InEbicsPayloadRequest{}
		if err := readJSON(r, request); handleError(w, logger, err) {
			return
		}

		host, subscriber, err := resolveEbicsSubscriberIDs(db, request.Subscriber)
		if handleError(w, logger, err) {
			return
		}

		resolved, err := ebicsruntime.ResolvePayloadRequest(
			payloadRequestToRuntime(request),
			"",
			map[string]any{},
			&restPayloadProfileResolver{db: db},
		)
		if handleError(w, logger, err) {
			return
		}

		validation, err := ebicsruntime.ValidateResolvedPayloadRequest(
			conf.GlobalConfig.GatewayName,
			resolved,
			&restContractViewResolver{db: db},
		)
		if handleError(w, logger, err) {
			return
		}

		if err = validatePayloadContractResult(resolved, validation); handleError(w, logger, err) {
			return
		}

		if routeOrderType, ok := mux.Vars(r)["order_type"]; ok && strings.TrimSpace(routeOrderType) != "" {
			expectedOrderType := model.NormalizeEbicsPayloadOrderType(routeOrderType)
			if resolved.OrderType != expectedOrderType {
				handleError(
					w,
					logger,
					badRequestf(
						"the resolved EBICS payload order %q does not match the route order %q",
						resolved.OrderType,
						expectedOrderType,
					),
				)

				return
			}
		}

		operation, err := ebicsruntime.NewPayloadOperation(&ebicsruntime.OperationMappingInput{
			Owner:             conf.GlobalConfig.GatewayName,
			EbicsHostID:       host.ID,
			EbicsSubscriberID: subscriber.ID,
			OrderType:         resolved.OrderType,
			OperationType:     derivePayloadOperationType(),
			Direction:         derivePayloadDirection(resolved.OrderType),
			TransportMode:     "ASYNC",
			CorrelationID:     resolvePayloadCorrelationID(resolved.ResolvedMetadata),
			ContractViewID:    resolved.ContractViewID,
			ResolvedRequest:   resolved,
		})
		if handleError(w, logger, err) {
			return
		}

		if err = db.Insert(operation).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, fmt.Sprint(operation.ID)))
		w.WriteHeader(http.StatusCreated)
		handleError(w, logger, writeJSON(w, payloadSubmissionResponse(operation)))
	}
}

func getEbicsPayload(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		operation, err := getDBEbicsOperation(r, db)
		if handleError(w, logger, err) {
			return
		}

		if !isRESTPayloadOrder(operation.OrderType) {
			handleError(w, logger, badRequestf("EBICS operation %d is not a payload operation", operation.ID))
			return
		}

		out, err := DBEbicsOperationToREST(operation)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, out))
	}
}

//nolint:dupl // list handlers stay explicit per EBICS resource
func listEbicsPayloads(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":    order{col: "id", asc: false},
		"id+":        order{col: "id", asc: true},
		"id-":        order{col: "id", asc: false},
		"status+":    order{col: "status", asc: true},
		"status-":    order{col: "status", asc: false},
		"created+":   order{col: "created_at", asc: true},
		"created-":   order{col: "created_at", asc: false},
		"orderType+": order{col: "order_type", asc: true},
		"orderType-": order{col: "order_type", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var operations model.EbicsOperations

		query, err := parseSelectQuery(r, db, validSorting, &operations)
		if handleError(w, logger, err) {
			return
		}

		query.Owner().In("order_type", "BTU", "BTD")
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsOperation, len(operations))
		for i, operation := range operations {
			out[i], err = DBEbicsOperationToREST(operation)
			if handleError(w, logger, err) {
				return
			}
		}

		handleError(w, logger, writeJSON(w, map[string]any{"payloads": out}))
	}
}

func actOnEbicsPayload(
	logger *log.Logger,
	db *database.DB,
	expectedDecision, targetStatus, actionName string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		operation, err := getDBEbicsOperation(r, db)
		if handleError(w, logger, err) {
			return
		}

		if !isRESTPayloadOrder(operation.OrderType) {
			handleError(w, logger, badRequestf("EBICS operation %d is not a payload operation", operation.ID))
			return
		}

		if operation.RetryDecision != expectedDecision && actionName == "recover" {
			handleError(
				w,
				logger,
				badRequestf("EBICS payload operation %d does not require recovery", operation.ID),
			)

			return
		}

		if actionName == "retry" &&
			operation.RetryDecision != payloadRetryDecisionAuto &&
			operation.RetryDecision != payloadRetryDecisionManual {
			handleError(
				w,
				logger,
				badRequestf("EBICS payload operation %d is not retryable", operation.ID),
			)

			return
		}

		action := &api.InEbicsPayloadAction{}
		if err = readJSON(r, action); handleError(w, logger, err) {
			return
		}

		operation.Status = targetStatus
		operation.StartedAt = time.Time{}
		operation.FinishedAt = time.Time{}
		operation.TechnicalReturnCode = ""
		operation.TechnicalReturnMessage = ""
		operation.BusinessReturnCode = ""
		operation.BusinessReturnMessage = ""
		operation.GatewayOutcome = "PENDING_BANK"
		operation.ManualActionRequired = false

		if operation.MetadataMap == nil {
			operation.MetadataMap = map[string]any{}
		}

		operation.MetadataMap["lastPayloadAction"] = actionName
		operation.MetadataMap["lastPayloadActionReason"] = strings.TrimSpace(action.Reason)
		if action.Metadata != nil {
			operation.MetadataMap["lastPayloadActionMetadata"] = action.Metadata
		}

		if actionName == "retry" {
			operation.RetryDecision = payloadRetryDecisionAuto
		}

		if err = db.Update(operation).Run(); handleError(w, logger, err) {
			return
		}

		out, err := DBEbicsOperationToREST(operation)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, out))
	}
}

func retryEbicsPayload(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsPayload(logger, db, payloadRetryDecisionAuto, payloadStatusPlanned, "retry")
}

func recoverEbicsPayload(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsPayload(logger, db, payloadRetryDecisionRecovery, payloadStatusReady, "recover")
}

func isRESTPayloadOrder(orderType string) bool {
	return model.IsEbicsPayloadOrderType(orderType)
}
