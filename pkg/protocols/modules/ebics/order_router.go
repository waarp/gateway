package ebics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	libebics "code.waarp.fr/lib/ebics/ebics"
	liborders "code.waarp.fr/lib/ebics/ebics/orders"
	libreturncode "code.waarp.fr/lib/ebics/ebics/returncode"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	transferInfoKeyEbicsOperationID   = "ebicsOperationID"
	transferInfoKeyEbicsOrderType     = "ebicsOrderType"
	transferInfoKeyEbicsHostID        = "ebicsHostID"
	transferInfoKeyEbicsPartnerID     = "ebicsPartnerID"
	transferInfoKeyEbicsUserID        = "ebicsUserID"
	transferInfoKeyEbicsRequestID     = "ebicsRequestID"
	transferInfoKeyEbicsCorrelationID = "ebicsCorrelationID"
	transferInfoKeyEbicsProtocol      = "ebicsProtocol"
	transferInfoKeyEbicsService       = "ebicsService"
)

type payloadOrderRouter struct {
	db     *database.DB
	logger *log.Logger
}

func newPayloadOrderRouter(db *database.DB, logger *log.Logger) *payloadOrderRouter {
	return &payloadOrderRouter{
		db:     db,
		logger: logger,
	}
}

//nolint:gocritic // lib-ebics provider interface imposes this value signature.
func (r *payloadOrderRouter) Upload(
	ctx context.Context,
	req libebics.OrderContext,
	params *ebicsxml.BTUOrderParams,
) error {
	operation, route, err := r.prepareRoute(ctx, req, buildUploadResolvedRequest(&req, params))
	if err != nil {
		return err
	}

	operation.Direction = model.EbicsOperationDirectionInboundForRuntime()
	operation.TransportMode = model.EbicsTransportModeSyncForRuntime()
	operation.StartedAt = time.Now().UTC()
	operation.Status = model.EbicsOperationStatusRunningForRuntime()

	if err = r.db.Insert(operation).Run(); err != nil {
		return fmt.Errorf("insert EBICS upload operation: %w", err)
	}

	transfer := r.newIncomingTransfer(operation, route, req, params)

	if err = r.db.Insert(transfer).Run(); err != nil {
		r.markOperationFailed(operation, err)
		return fmt.Errorf("insert EBICS upload transfer: %w", err)
	}

	if err = ebicsruntime.BindTransferToOperation(operation, transfer.ID); err != nil {
		r.markOperationFailed(operation, err)
		return fmt.Errorf("bind EBICS upload operation to transfer: %w", err)
	}

	if err = r.db.Update(operation).Run(); err != nil {
		r.markOperationFailed(operation, err)
		return fmt.Errorf("update EBICS upload operation transfer link: %w", err)
	}

	if err = r.ingestIncomingPayload(transfer, req.PayloadRaw); err != nil {
		r.markOperationFailed(operation, err)
		return err
	}

	return r.completeOperation(operation)
}

//nolint:gocritic // lib-ebics provider interface imposes this value signature.
func (r *payloadOrderRouter) Download(
	ctx context.Context,
	req libebics.OrderContext,
	params *ebicsxml.BTDOrderParams,
) (libebics.OrderResult, error) {
	operation, route, err := r.prepareRoute(ctx, req, buildDownloadResolvedRequest(&req, params))
	if err != nil {
		return libebics.OrderResult{}, err
	}

	operation.Direction = model.EbicsOperationDirectionOutboundForRuntime()
	operation.TransportMode = model.EbicsTransportModeSyncForRuntime()
	operation.StartedAt = time.Now().UTC()
	operation.Status = model.EbicsOperationStatusRunningForRuntime()

	if err = r.db.Insert(operation).Run(); err != nil {
		return libebics.OrderResult{}, fmt.Errorf("insert EBICS download operation: %w", err)
	}

	transfer, err := r.findOutgoingTransfer(route)
	if err != nil {
		r.markOperationFailed(operation, err)
		return libebics.OrderResult{}, err
	}

	enrichTransferInfo(transfer, operation, route.resolved)
	if transfer.ID != 0 {
		if updateErr := r.db.Update(transfer).Run(); updateErr != nil {
			err = fmt.Errorf("update EBICS download transfer metadata: %w", updateErr)
			r.markOperationFailed(operation, err)

			return libebics.OrderResult{}, err
		}
	}

	if err = ebicsruntime.BindTransferToOperation(operation, transfer.ID); err != nil {
		r.markOperationFailed(operation, err)
		return libebics.OrderResult{}, fmt.Errorf("bind EBICS download operation to transfer: %w", err)
	}

	if err = r.db.Update(operation).Run(); err != nil {
		r.markOperationFailed(operation, err)
		return libebics.OrderResult{}, fmt.Errorf("update EBICS download operation transfer link: %w", err)
	}

	payload, err := r.extractOutgoingPayload(transfer)
	if err != nil {
		r.markOperationFailed(operation, err)
		return libebics.OrderResult{}, err
	}

	completeErr := r.completeOperation(operation)
	if completeErr != nil {
		return libebics.OrderResult{}, completeErr
	}

	return libebics.SuccessResult(payload), nil
}

type payloadRoute struct {
	host       *model.EbicsHost
	subscriber *model.EbicsSubscriber
	localAgent *model.LocalAgent
	localAcc   *model.LocalAccount
	profile    *model.EbicsPayloadProfile
	rule       *model.Rule
	resolved   *ebicsruntime.ResolvedPayloadRequest
	contract   *ebicsruntime.ContractValidationResult
}

//nolint:gocritic // keeping req by value avoids repeated dereference noise at call sites.
func (r *payloadOrderRouter) prepareRoute(
	_ context.Context,
	req libebics.OrderContext,
	resolved *ebicsruntime.ResolvedPayloadRequest,
) (*model.EbicsOperation, *payloadRoute, error) {
	route, err := r.resolveRoute(req, resolved)
	if err != nil {
		return nil, nil, err
	}

	operation, err := ebicsruntime.NewPayloadOperation(&ebicsruntime.OperationMappingInput{
		Owner:             conf.GlobalConfig.GatewayName,
		LocalAgentID:      route.localAgent.ID,
		LocalAccountID:    route.localAcc.ID,
		EbicsHostID:       route.host.ID,
		EbicsSubscriberID: route.subscriber.ID,
		OrderType:         resolved.OrderType,
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		Direction:         model.EbicsOperationDirectionInternalForRuntime(),
		TransportMode:     model.EbicsTransportModeSyncForRuntime(),
		CorrelationID:     resolveRuntimeCorrelationID(&req),
		ContractViewID:    resolved.ContractViewID,
		ResolvedRequest:   resolved,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("build EBICS payload operation: %w", err)
	}

	operation.RequestID = strings.TrimSpace(req.OrderID)
	operation.TransactionID = strings.TrimSpace(string(req.TransID))
	operation.EbicsVersion = strings.ToUpper(strings.TrimSpace(req.ProtocolVersion))

	return operation, route, nil
}

//nolint:gocritic // keeping req by value avoids repeated dereference noise at call sites.
func (r *payloadOrderRouter) resolveRoute(
	req libebics.OrderContext,
	resolved *ebicsruntime.ResolvedPayloadRequest,
) (*payloadRoute, error) {
	host, subscriber, err := r.resolveHostAndSubscriber(req)
	if err != nil {
		return nil, err
	}

	localAcc, localAgent, err := r.resolveServerAccount(subscriber)
	if err != nil {
		return nil, err
	}

	profile, rule, err := r.matchPayloadProfile(resolved.OrderType, resolved.ResolvedService)
	if err != nil {
		return nil, err
	}

	resolved.Profile = profile
	resolved.ProfileName = profile.Name
	resolved.RuleName = rule.Name
	resolved.ContractViewID = 0
	resolved.ContractItemIDs = nil

	validation, err := ebicsruntime.ValidateResolvedPayloadRequest(
		conf.GlobalConfig.GatewayName,
		resolved,
		&routerContractViewResolver{db: r.db},
	)
	if err != nil {
		return nil, fmt.Errorf("validate EBICS payload request against contract: %w", err)
	}

	contractErr := validateRouterContract(resolved, validation)
	if contractErr != nil {
		return nil, contractErr
	}

	if rule.IsSend != isDownloadOrder(resolved.OrderType) {
		expectedDirection := "receive"
		if isDownloadOrder(resolved.OrderType) {
			expectedDirection = "send"
		}

		return nil, mappedOrderError(fmt.Errorf(
			"%w: the EBICS payload profile %q is bound to an incompatible Gateway rule %q, expected a %s rule",
			liborders.ErrProcessing,
			profile.Name,
			rule.Name,
			expectedDirection,
		))
	}

	return &payloadRoute{
		host:       host,
		subscriber: subscriber,
		localAgent: localAgent,
		localAcc:   localAcc,
		profile:    profile,
		rule:       rule,
		resolved:   resolved,
		contract:   validation,
	}, nil
}

//nolint:gocritic // keeping req by value avoids repeated dereference noise at call sites.
func (r *payloadOrderRouter) resolveHostAndSubscriber(
	req libebics.OrderContext,
) (*model.EbicsHost, *model.EbicsSubscriber, error) {
	host := &model.EbicsHost{}
	if err := r.db.Get(host, "owner=? AND host_id=?",
		conf.GlobalConfig.GatewayName, strings.TrimSpace(string(req.HostID))).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, mappedOrderError(fmt.Errorf("%w: unknown EBICS host %q",
				liborders.ErrInvalidOrderParams, req.HostID))
		}

		return nil, nil, fmt.Errorf("load EBICS host %q: %w", req.HostID, err)
	}

	subscriber := &model.EbicsSubscriber{}
	if err := r.db.Get(
		subscriber,
		"owner=? AND ebics_host_id=? AND partner_id=? AND user_id=?",
		conf.GlobalConfig.GatewayName,
		host.ID,
		strings.TrimSpace(string(req.PartnerID)),
		strings.TrimSpace(string(req.UserID)),
	).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, mappedOrderError(fmt.Errorf(
				"%w: unknown EBICS subscriber %q/%q on host %q",
				liborders.ErrInvalidOrderParams,
				req.PartnerID,
				req.UserID,
				req.HostID,
			))
		}

		return nil, nil, fmt.Errorf("load EBICS subscriber %q/%q on host %q: %w",
			req.PartnerID, req.UserID, req.HostID, err)
	}

	if !subscriber.Enabled {
		return nil, nil, mappedOrderError(fmt.Errorf(
			"%w: the EBICS subscriber %q/%q is disabled",
			liborders.ErrInvalidOrderParams,
			req.PartnerID,
			req.UserID,
		))
	}

	return host, subscriber, nil
}

func (r *payloadOrderRouter) resolveServerAccount(
	subscriber *model.EbicsSubscriber,
) (*model.LocalAccount, *model.LocalAgent, error) {
	if !subscriber.LocalAccountID.Valid {
		return nil, nil, mappedOrderError(fmt.Errorf(
			"%w: the EBICS subscriber %q is not linked to a Gateway local account",
			liborders.ErrProcessing,
			subscriber.Name,
		))
	}

	localAcc := &model.LocalAccount{}
	if err := r.db.Get(localAcc, "id=?", subscriber.LocalAccountID.Int64).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, mappedOrderError(fmt.Errorf(
				"%w: the linked local account %d does not exist",
				liborders.ErrProcessing,
				subscriber.LocalAccountID.Int64,
			))
		}

		return nil, nil, fmt.Errorf("load linked local account %d: %w",
			subscriber.LocalAccountID.Int64, err)
	}

	localAgent := &model.LocalAgent{}
	if err := r.db.Get(localAgent, "id=?", localAcc.LocalAgentID).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, mappedOrderError(fmt.Errorf(
				"%w: the linked local agent %d does not exist",
				liborders.ErrProcessing,
				localAcc.LocalAgentID,
			))
		}

		return nil, nil, fmt.Errorf("load local agent %d for EBICS subscriber %q: %w",
			localAcc.LocalAgentID, subscriber.Name, err)
	}

	if localAgent.Protocol != EBICS {
		return nil, nil, mappedOrderError(fmt.Errorf(
			"%w: the linked local agent %q does not use the EBICS protocol",
			liborders.ErrProcessing,
			localAgent.Name,
		))
	}

	return localAcc, localAgent, nil
}

//nolint:gocritic // service is intentionally passed by value as a small immutable descriptor here.
func (r *payloadOrderRouter) matchPayloadProfile(
	orderType string,
	service ebicsruntime.PayloadServiceRef,
) (*model.EbicsPayloadProfile, *model.Rule, error) {
	var profiles model.EbicsPayloadProfiles
	if err := r.db.Select(&profiles).Owner().
		Where("order_type=? AND is_enabled=?", strings.ToUpper(strings.TrimSpace(orderType)), true).
		Run(); err != nil {
		return nil, nil, fmt.Errorf("load EBICS payload profiles for order type %q: %w", orderType, err)
	}

	var (
		bestProfile *model.EbicsPayloadProfile
		bestRule    *model.Rule
		bestScore   = -1
	)

	for _, profile := range profiles {
		score, ok := scorePayloadProfile(profile, service)
		if !ok {
			continue
		}

		rule, err := r.resolveProfileRule(profile)
		if err != nil {
			return nil, nil, err
		}

		switch {
		case score > bestScore:
			bestProfile = profile
			bestRule = rule
			bestScore = score
		case score == bestScore:
			return nil, nil, mappedOrderError(fmt.Errorf(
				"%w: ambiguous EBICS payload profile match between %q and %q",
				liborders.ErrInvalidOrderParams,
				bestProfile.Name,
				profile.Name,
			))
		}
	}

	if bestProfile == nil || bestRule == nil {
		return nil, nil, mappedOrderError(fmt.Errorf(
			"%w: no enabled EBICS payload profile matches %s/%s/%s/%s for order %q",
			liborders.ErrInvalidOrderParams,
			service.ServiceName,
			service.ServiceOption,
			service.Scope,
			service.MsgName,
			orderType,
		))
	}

	bestProfile.DefaultRuleName = bestRule.Name

	return bestProfile, bestRule, nil
}

func (r *payloadOrderRouter) resolveProfileRule(
	profile *model.EbicsPayloadProfile,
) (*model.Rule, error) {
	if profile == nil || !profile.DefaultRuleID.Valid {
		return nil, mappedOrderError(fmt.Errorf(
			"%w: the EBICS payload profile %q is missing its default Gateway rule",
			liborders.ErrProcessing,
			profile.Name,
		))
	}

	rule := &model.Rule{}
	if err := r.db.Get(rule, "id=?", profile.DefaultRuleID.Int64).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, mappedOrderError(fmt.Errorf(
				"%w: the Gateway rule %d referenced by profile %q does not exist",
				liborders.ErrProcessing,
				profile.DefaultRuleID.Int64,
				profile.Name,
			))
		}

		return nil, fmt.Errorf("load Gateway rule %d for profile %q: %w",
			profile.DefaultRuleID.Int64, profile.Name, err)
	}

	return rule, nil
}

//nolint:gocritic // keeping req by value avoids repeated dereference noise at call sites.
func (r *payloadOrderRouter) newIncomingTransfer(
	operation *model.EbicsOperation,
	route *payloadRoute,
	req libebics.OrderContext,
	params *ebicsxml.BTUOrderParams,
) *model.Transfer {
	filename := deriveIncomingFilename(params, &req)
	transfer := &model.Transfer{
		RuleID:         route.rule.ID,
		LocalAccountID: utils.NewNullInt64(route.localAcc.ID),
		DestFilename:   filename,
		Filesize:       int64(len(req.PayloadRaw)),
		Start:          time.Now().UTC(),
		Status:         types.StatusAvailable,
		TransferInfo:   map[string]any{},
	}

	enrichTransferInfo(transfer, operation, route.resolved)

	return transfer
}

func (r *payloadOrderRouter) ingestIncomingPayload(transfer *model.Transfer, payload []byte) error {
	pip, err := pipeline.NewServerPipeline(r.db, r.logger, transfer, snmp.GlobalService)
	if err != nil {
		return fmt.Errorf("initialize Gateway pipeline for incoming EBICS payload: %w", err)
	}

	if taskErr := pip.PreTasks(); taskErr != nil {
		return fmt.Errorf("run pre-tasks for incoming EBICS payload: %w", taskErr)
	}

	stream, taskErr := pip.StartData()
	if taskErr != nil {
		return fmt.Errorf("open Gateway stream for incoming EBICS payload: %w", taskErr)
	}

	if _, copyErr := io.Copy(stream, bytes.NewReader(payload)); copyErr != nil {
		pip.SetError(types.TeConnectionReset, copyErr.Error())
		return fmt.Errorf("write incoming EBICS payload to Gateway stream: %w", copyErr)
	}

	if taskErr = pip.EndData(); taskErr != nil {
		return fmt.Errorf("finalize incoming EBICS payload data phase: %w", taskErr)
	}

	if taskErr = pip.PostTasks(); taskErr != nil {
		return fmt.Errorf("run post-tasks for incoming EBICS payload: %w", taskErr)
	}

	if taskErr = pip.EndTransfer(); taskErr != nil {
		return fmt.Errorf("finalize incoming EBICS payload transfer: %w", taskErr)
	}

	return nil
}

func (r *payloadOrderRouter) findOutgoingTransfer(route *payloadRoute) (*model.Transfer, error) {
	transfer, err := pipeline.GetAvailableTransferByRule(r.db, "", route.localAcc, route.rule)
	if err == nil {
		return transfer, nil
	}

	if !database.IsNotFound(err) {
		return nil, fmt.Errorf("retrieve available EBICS outgoing transfer: %w", err)
	}

	return nil, mappedOrderError(liborders.ErrNoDownloadData)
}

func (r *payloadOrderRouter) extractOutgoingPayload(transfer *model.Transfer) ([]byte, error) {
	pip, err := pipeline.NewServerPipeline(r.db, r.logger, transfer, snmp.GlobalService)
	if err != nil {
		return nil, fmt.Errorf("initialize Gateway pipeline for outgoing EBICS payload: %w", err)
	}

	if taskErr := pip.PreTasks(); taskErr != nil {
		return nil, fmt.Errorf("run pre-tasks for outgoing EBICS payload: %w", taskErr)
	}

	stream, taskErr := pip.StartData()
	if taskErr != nil {
		return nil, fmt.Errorf("open Gateway stream for outgoing EBICS payload: %w", taskErr)
	}

	payload, readErr := io.ReadAll(stream)
	if readErr != nil {
		pip.SetError(types.TeConnectionReset, readErr.Error())
		return nil, fmt.Errorf("read outgoing EBICS payload from Gateway stream: %w", readErr)
	}

	if taskErr = pip.EndData(); taskErr != nil {
		return nil, fmt.Errorf("finalize outgoing EBICS payload data phase: %w", taskErr)
	}

	if taskErr = pip.PostTasks(); taskErr != nil {
		return nil, fmt.Errorf("run post-tasks for outgoing EBICS payload: %w", taskErr)
	}

	if taskErr = pip.EndTransfer(); taskErr != nil {
		return nil, fmt.Errorf("finalize outgoing EBICS payload transfer: %w", taskErr)
	}

	return payload, nil
}

func (r *payloadOrderRouter) completeOperation(operation *model.EbicsOperation) error {
	if err := ebicsruntime.UpdateOperationOutcomeFromReturnCodes(operation, "", "", "", ""); err != nil {
		return fmt.Errorf("derive successful EBICS operation outcome: %w", err)
	}

	operation.FinishedAt = time.Now().UTC()

	if err := r.db.Update(operation).Run(); err != nil {
		return fmt.Errorf("update successful EBICS operation %d: %w", operation.ID, err)
	}

	return nil
}

func (r *payloadOrderRouter) markOperationFailed(operation *model.EbicsOperation, err error) {
	if operation == nil || operation.ID == 0 {
		return
	}

	code, ok := libreturncode.FromError(err)
	if ok {
		technicalCode, businessCode := "", ""
		technicalMsg, businessMsg := "", ""

		switch code.Scope {
		case libreturncode.ScopeTechnical:
			technicalCode = string(code.Value)
			technicalMsg = err.Error()
		case libreturncode.ScopeBusiness:
			businessCode = string(code.Value)
			businessMsg = err.Error()
		}

		if mapErr := ebicsruntime.UpdateOperationOutcomeFromReturnCodes(
			operation,
			technicalCode,
			technicalMsg,
			businessCode,
			businessMsg,
		); mapErr != nil {
			r.logger.Warningf("failed to map EBICS return code on operation %d: %v", operation.ID, mapErr)
		}
	} else {
		operation.Status = model.EbicsOperationStatusFailedForRuntime()
		operation.Severity = model.EbicsOperationSeverityErrorForRuntime()
		operation.GatewayOutcome = model.EbicsGatewayOutcomeTechnicalFatalFailureForRuntime()
		operation.RetryDecision = model.EbicsRetryDecisionNoRetryForRuntime()
		operation.TechnicalReturnMessage = err.Error()
	}

	operation.FinishedAt = time.Now().UTC()
	if updateErr := r.db.Update(operation).Run(); updateErr != nil {
		r.logger.Warningf("failed to persist EBICS operation %d failure state: %v", operation.ID, updateErr)
	}
}

type routerContractViewResolver struct {
	db *database.DB
}

//nolint:nilnil // nil view is the expected "no active contract" signal for the contract validator.
func (r *routerContractViewResolver) GetActiveContractView(
	owner, hostID, partnerID, userID string,
) (*model.EbicsContractView, []model.EbicsContractViewItem, error) {
	host := &model.EbicsHost{}
	if err := r.db.Get(host, "host_id=? AND owner=?", hostID, owner).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, nil
		}

		return nil, nil, fmt.Errorf("load EBICS host %q for contract resolution: %w", hostID, err)
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
			return nil, nil, nil
		}

		return nil, nil, fmt.Errorf("load EBICS subscriber %q/%q for contract resolution: %w",
			partnerID, userID, err)
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
			return nil, nil, nil
		}

		return nil, nil, fmt.Errorf("load active EBICS contract view for %q/%q: %w",
			partnerID, userID, err)
	}

	var rows model.EbicsContractViewItems
	if err := r.db.Select(&rows).Owner().Where("contract_view_id=?", view.ID).Run(); err != nil {
		return nil, nil, fmt.Errorf("load EBICS contract view items: %w", err)
	}

	items := make([]model.EbicsContractViewItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, *row)
	}

	return view, items, nil
}

func buildUploadResolvedRequest(
	req *libebics.OrderContext,
	params *ebicsxml.BTUOrderParams,
) *ebicsruntime.ResolvedPayloadRequest {
	return &ebicsruntime.ResolvedPayloadRequest{
		OrderType:      model.NormalizeEbicsPayloadOrderType(string(req.OrderType)),
		ResolutionMode: "profile",
		Subscriber: ebicsruntime.PayloadSubscriberRef{
			HostID:    strings.TrimSpace(string(req.HostID)),
			PartnerID: strings.TrimSpace(string(req.PartnerID)),
			UserID:    strings.TrimSpace(string(req.UserID)),
		},
		ResolvedFile: &ebicsruntime.PayloadFileRef{
			OutputName: strings.TrimSpace(orderUploadFilename(params)),
		},
		ResolvedService: normalizeBTUService(params, req.OrderType),
		ResolvedMetadata: map[string]any{
			"requestID":     strings.TrimSpace(req.OrderID),
			"correlationID": resolveRuntimeCorrelationID(req),
			"protocol":      strings.ToUpper(strings.TrimSpace(req.ProtocolVersion)),
		},
	}
}

func buildDownloadResolvedRequest(
	req *libebics.OrderContext,
	params *ebicsxml.BTDOrderParams,
) *ebicsruntime.ResolvedPayloadRequest {
	return &ebicsruntime.ResolvedPayloadRequest{
		OrderType:      model.NormalizeEbicsPayloadOrderType(string(req.OrderType)),
		ResolutionMode: "profile",
		Subscriber: ebicsruntime.PayloadSubscriberRef{
			HostID:    strings.TrimSpace(string(req.HostID)),
			PartnerID: strings.TrimSpace(string(req.PartnerID)),
			UserID:    strings.TrimSpace(string(req.UserID)),
		},
		ResolvedService: normalizeBTDService(params, req.OrderType),
		ResolvedMetadata: map[string]any{
			"requestID":     strings.TrimSpace(req.OrderID),
			"correlationID": resolveRuntimeCorrelationID(req),
			"protocol":      strings.ToUpper(strings.TrimSpace(req.ProtocolVersion)),
		},
	}
}

func validateRouterContract(
	resolved *ebicsruntime.ResolvedPayloadRequest,
	validation *ebicsruntime.ContractValidationResult,
) error {
	if validation == nil {
		return mappedOrderError(fmt.Errorf("%w: missing EBICS contract validation result",
			liborders.ErrProcessing))
	}

	switch validation.Status {
	case "MATCHED":
		return nil
	case "NO_ACTIVE_CONTRACT":
		return mappedOrderError(fmt.Errorf("%w: no active EBICS contract view found",
			liborders.ErrInvalidOrderParams))
	case "NO_MATCHING_ITEM":
		return mappedOrderError(fmt.Errorf(
			"%w: the EBICS payload profile %q is not allowed by the active contract",
			liborders.ErrInvalidOrderParams,
			resolved.ProfileName,
		))
	default:
		return mappedOrderError(fmt.Errorf(
			"%w: unsupported EBICS contract validation status %q",
			liborders.ErrProcessing,
			validation.Status,
		))
	}
}

//nolint:gocritic // service is intentionally passed by value as a small immutable descriptor here.
func scorePayloadProfile(profile *model.EbicsPayloadProfile, service ebicsruntime.PayloadServiceRef) (int, bool) {
	if profile == nil {
		return 0, false
	}

	score := 0
	if !matchOptionalProfileField(profile.ServiceName, service.ServiceName, &score) {
		return 0, false
	}
	if !matchOptionalProfileField(profile.ServiceOption, service.ServiceOption, &score) {
		return 0, false
	}
	if !matchOptionalProfileField(profile.Scope, service.Scope, &score) {
		return 0, false
	}
	if !matchOptionalProfileField(profile.MsgName, service.MsgName, &score) {
		return 0, false
	}
	if !matchOptionalProfileField(profile.ContainerType, service.ContainerType, &score) {
		return 0, false
	}

	return score, true
}

func matchOptionalProfileField(expected, actual string, score *int) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" {
		return true
	}
	if expected != actual {
		return false
	}
	(*score)++

	return true
}

func normalizeBTUService(params *ebicsxml.BTUOrderParams, orderType libebics.OrderType) ebicsruntime.PayloadServiceRef {
	service := ebicsruntime.PayloadServiceRef{OrderType: model.NormalizeEbicsPayloadOrderType(string(orderType))}
	if params == nil {
		return service
	}

	service.ServiceName = strings.TrimSpace(params.Service.ServiceName)
	service.ServiceOption = strings.TrimSpace(params.Service.ServiceOption)
	service.Scope = strings.TrimSpace(params.Service.Scope)
	service.MsgName = strings.TrimSpace(params.Service.MsgName.Value)
	if params.Service.Container != nil {
		service.ContainerType = strings.TrimSpace(params.Service.Container.ContainerType)
	}

	return service
}

func normalizeBTDService(params *ebicsxml.BTDOrderParams, orderType libebics.OrderType) ebicsruntime.PayloadServiceRef {
	service := ebicsruntime.PayloadServiceRef{OrderType: model.NormalizeEbicsPayloadOrderType(string(orderType))}
	if params == nil {
		return service
	}

	service.ServiceName = strings.TrimSpace(params.Service.ServiceName)
	service.ServiceOption = strings.TrimSpace(params.Service.ServiceOption)
	service.Scope = strings.TrimSpace(params.Service.Scope)
	service.MsgName = strings.TrimSpace(params.Service.MsgName.Value)
	if params.Service.Container != nil {
		service.ContainerType = strings.TrimSpace(params.Service.Container.ContainerType)
	}

	return service
}

func orderUploadFilename(params *ebicsxml.BTUOrderParams) string {
	if params == nil {
		return ""
	}

	return strings.TrimSpace(params.FileName)
}

//nolint:gocritic // keeping req by value avoids repeated dereference noise at call sites.
func deriveIncomingFilename(params *ebicsxml.BTUOrderParams, req *libebics.OrderContext) string {
	if fileName := strings.TrimSpace(orderUploadFilename(params)); fileName != "" {
		return filepath.ToSlash(fileName)
	}

	if requestID := strings.TrimSpace(req.OrderID); requestID != "" {
		return filepath.ToSlash(requestID + ".xml")
	}

	return filepath.ToSlash("ebics-" + strings.ToLower(string(req.OrderType)) + ".xml")
}

//nolint:gocritic // keeping req by value avoids repeated dereference noise at call sites.
func resolveRuntimeCorrelationID(req *libebics.OrderContext) string {
	if value := strings.TrimSpace(req.CorrelationID); value != "" {
		return value
	}

	if value := strings.TrimSpace(req.OrderID); value != "" {
		return value
	}

	return strings.TrimSpace(string(req.TransID))
}

func mappedOrderError(err error) error {
	return fmt.Errorf("map EBICS order error: %w", liborders.MapOrderError(err))
}

func enrichTransferInfo(
	transfer *model.Transfer,
	operation *model.EbicsOperation,
	resolved *ebicsruntime.ResolvedPayloadRequest,
) {
	if transfer.TransferInfo == nil {
		transfer.TransferInfo = map[string]any{}
	}

	transfer.TransferInfo[transferInfoKeyEbicsOperationID] = operation.ID
	transfer.TransferInfo[transferInfoKeyEbicsOrderType] = operation.OrderType
	transfer.TransferInfo[transferInfoKeyEbicsHostID] = resolved.Subscriber.HostID
	transfer.TransferInfo[transferInfoKeyEbicsPartnerID] = resolved.Subscriber.PartnerID
	transfer.TransferInfo[transferInfoKeyEbicsUserID] = resolved.Subscriber.UserID
	transfer.TransferInfo[transferInfoKeyEbicsRequestID] = operation.RequestID
	transfer.TransferInfo[transferInfoKeyEbicsCorrelationID] = operation.CorrelationID
	transfer.TransferInfo[transferInfoKeyEbicsProtocol] = operation.EbicsVersion
	transfer.TransferInfo[transferInfoKeyEbicsService] = map[string]any{
		"serviceName":   resolved.ResolvedService.ServiceName,
		"serviceOption": resolved.ResolvedService.ServiceOption,
		"scope":         resolved.ResolvedService.Scope,
		"msgName":       resolved.ResolvedService.MsgName,
		"containerType": resolved.ResolvedService.ContainerType,
	}
}

func isDownloadOrder(orderType string) bool {
	return model.IsEbicsPayloadDownloadOrder(orderType)
}
