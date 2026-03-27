package ebics

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
	"github.com/google/uuid"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	keyRotationOrderPUB = "PUB"
	keyRotationOrderHCA = "HCA"
	keyRotationOrderHCS = "HCS"
	keyRotationOrderHSA = "HSA"
	keyRotationOrderSPR = "SPR"

	defaultSignatureRotationOrder = keyRotationOrderHSA
)

var (
	errMissingRotationUsageSelection  = errors.New("no coordinated EBICS key rotation usage was selected")
	errIncompleteKeyPairRotation      = errors.New("authentication and encryption rotations must be prepared together")
	errUnsupportedSignatureOrderType  = errors.New("unsupported EBICS signature rotation order type")
	errMissingRotationCoordinationID  = errors.New("missing coordinated EBICS key rotation identifier")
	errMissingRotationSignatureData   = errors.New("missing coordinated EBICS key rotation signature data")
	errUnsupportedRotationAction      = errors.New("unsupported coordinated EBICS key rotation action")
	errRotationRequiresPendingGroup   = errors.New("no pending coordinated EBICS key rotation group exists")
	errRotationRequiresSentGroup      = errors.New("the coordinated EBICS key rotation group was not sent to the bank")
	errRotationMissingActivatedSource = errors.New("missing activated EBICS key lifecycle for coordinated rotation")
)

// KeyRotationPrepareInput defines one coordinated EBICS key rotation preparation.
type KeyRotationPrepareInput struct {
	EbicsSubscriberID              int64
	RotationType                   string
	CoordinationID                 string
	NextAuthenticationCredentialID *int64
	NextEncryptionCredentialID     *int64
	NextSignatureCredentialID      *int64
	SignatureOrderType             string
	Operator                       string
	Reason                         string
	Evidence                       map[string]any
}

// KeyRotationActionInput defines one coordinated action on a prepared key rotation.
type KeyRotationActionInput struct {
	EbicsSubscriberID  int64
	CoordinationID     string
	SignatureOrderType string
	SignatureData      []byte
	Operator           string
	Reason             string
	Evidence           map[string]any
}

// KeyRotationGroupResult exposes one coordinated rotation group state.
type KeyRotationGroupResult struct {
	CoordinationID string
	Lifecycles     []*model.EbicsKeyLifecycle
	Operations     []*model.EbicsOperation
}

type rotationPlan struct {
	orderType string
	usages    []string
}

// PrepareCoordinatedKeyRotation prepares one multi-key rotation group for a subscriber.
func PrepareCoordinatedKeyRotation(
	ctx context.Context,
	db *database.DB,
	input *KeyRotationPrepareInput,
) (*KeyRotationGroupResult, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.prepareCoordinatedKeyRotation(input)
}

// SendCoordinatedKeyRotation sends the prepared rotation group to the bank.
func SendCoordinatedKeyRotation(
	ctx context.Context,
	db *database.DB,
	input *KeyRotationActionInput,
) (*KeyRotationGroupResult, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.sendCoordinatedKeyRotation(input)
}

// ConfirmCoordinatedKeyRotation activates one rotation group after bank confirmation.
func ConfirmCoordinatedKeyRotation(
	ctx context.Context,
	db *database.DB,
	input *KeyRotationActionInput,
) (*KeyRotationGroupResult, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.confirmCoordinatedKeyRotation(input)
}

// CancelCoordinatedKeyRotation cancels one prepared rotation group before activation.
func CancelCoordinatedKeyRotation(
	ctx context.Context,
	db *database.DB,
	input *KeyRotationActionInput,
) (*KeyRotationGroupResult, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.finalizeCoordinatedKeyRotation(input, model.EbicsKeyLifecycleStatusCancelledForRuntime())
}

// RejectCoordinatedKeyRotation rejects one prepared rotation group after bank rejection.
func RejectCoordinatedKeyRotation(
	ctx context.Context,
	db *database.DB,
	input *KeyRotationActionInput,
) (*KeyRotationGroupResult, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.finalizeCoordinatedKeyRotation(input, model.EbicsKeyLifecycleStatusRejectedForRuntime())
}

// RevokeCoordinatedKeyRotation sends one coordinated SPR revocation and cancels the group.
func RevokeCoordinatedKeyRotation(
	ctx context.Context,
	db *database.DB,
	input *KeyRotationActionInput,
) (*KeyRotationGroupResult, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.revokeCoordinatedKeyRotation(input)
}

func (c *Client) prepareCoordinatedKeyRotation(input *KeyRotationPrepareInput) (*KeyRotationGroupResult, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}

	if err := validateKeyRotationPrepareInput(input); err != nil {
		return nil, err
	}

	coordinationID := strings.TrimSpace(input.CoordinationID)
	if coordinationID == "" {
		coordinationID = uuid.NewString()
	}

	selected := selectedRotationCredentials(input)
	state, err := c.loadKeyLifecycleState(input.EbicsSubscriberID)
	if err != nil {
		return nil, err
	}

	err = c.db.Transaction(func(tx *database.Session) error {
		for usage, nextCredentialID := range selected {
			current := state.activated[usage]
			if current == nil {
				return fmt.Errorf("%w: subscriber=%d usage=%s", errRotationMissingActivatedSource, input.EbicsSubscriberID, usage)
			}

			pending := state.pending[usage]
			if pending != nil && pending.CoordinationID != "" && pending.CoordinationID != coordinationID {
				return database.NewValidationErrorf(
					"an EBICS key rotation is already pending for subscriber %d and usage %q",
					input.EbicsSubscriberID,
					usage,
				)
			}

			target := pending
			if target == nil {
				target = &model.EbicsKeyLifecycle{
					EbicsSubscriberID:   input.EbicsSubscriberID,
					KeyUsage:            usage,
					CurrentCredentialID: current.CurrentCredentialID,
				}
			}

			target.RotationType = defaultRotationType(input.RotationType)
			target.CoordinationID = coordinationID
			target.NextCredentialID = sql.NullInt64{Int64: *nextCredentialID, Valid: true}
			target.Status = model.EbicsKeyLifecycleStatusMaterialPreparedForRuntime()
			target.Operator = strings.TrimSpace(input.Operator)
			target.Reason = strings.TrimSpace(input.Reason)
			target.EvidenceMap = mergeRotationEvidence(input.Evidence, coordinationID, input.SignatureOrderType)
			target.RequestedAt = time.Time{}
			target.SentAt = time.Time{}
			target.ActivatedAt = time.Time{}
			target.RetiredAt = time.Time{}
			target.TriggerOperationID = sql.NullInt64{}
			target.LastOperationID = sql.NullInt64{}

			if pending == nil {
				if insertErr := tx.Insert(target).Run(); insertErr != nil {
					return fmt.Errorf("insert pending EBICS key lifecycle for usage %s: %w", usage, insertErr)
				}
			} else {
				if updateErr := tx.Update(target).Run(); updateErr != nil {
					return fmt.Errorf("update pending EBICS key lifecycle %d: %w", target.ID, updateErr)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	lifecycles, loadErr := c.loadRotationGroup(input.EbicsSubscriberID, coordinationID)
	if loadErr != nil {
		return nil, loadErr
	}

	return &KeyRotationGroupResult{
		CoordinationID: coordinationID,
		Lifecycles:     lifecycles,
	}, nil
}

func (c *Client) sendCoordinatedKeyRotation(input *KeyRotationActionInput) (*KeyRotationGroupResult, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}

	lifecycles, err := c.loadPendingRotationGroup(input.EbicsSubscriberID, input.CoordinationID)
	if err != nil {
		return nil, err
	}

	plan, err := resolveRotationPlan(lifecycles, input.SignatureOrderType)
	if err != nil {
		return nil, err
	}

	execCtx, err := c.newAdminExecutionContext(input.EbicsSubscriberID)
	if err != nil {
		return nil, err
	}

	operations := make([]*model.EbicsOperation, 0, 1)
	requestCtx, cancel := context.WithTimeout(context.Background(), c.adminRequestTimeout())
	defer cancel()

	operation, err := c.insertNonPayloadOperation(execCtx, "KEY_MANAGEMENT", plan.orderType, "OUTBOUND")
	if err != nil {
		return nil, err
	}

	orderData, err := c.buildRotationOrderData(execCtx.subscriber, lifecycles, plan)
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "build coordinated key rotation order data", err)
	}

	if execErr := c.executeRotationPlanStep(requestCtx, execCtx, plan.orderType, orderData); execErr != nil {
		return nil, c.failNonPayloadOperation(operation, "send coordinated key rotation", execErr)
	}

	operation.MetadataMap["keyRotation"] = map[string]any{
		"coordinationID": input.CoordinationID,
		"usages":         plan.usages,
	}
	if completeErr := c.completeNonPayloadOperation(operation, "", ""); completeErr != nil {
		return nil, completeErr
	}
	operations = append(operations, operation)

	now := time.Now().UTC()
	err = c.db.Transaction(func(tx *database.Session) error {
		for _, lifecycle := range lifecycles {
			lifecycle.Status = model.EbicsKeyLifecycleStatusOrderSentForRuntime()
			lifecycle.Operator = strings.TrimSpace(input.Operator)
			lifecycle.Reason = strings.TrimSpace(input.Reason)
			lifecycle.EvidenceMap = mergeRotationEvidence(input.Evidence, input.CoordinationID, input.SignatureOrderType)
			if lifecycle.RequestedAt.IsZero() {
				lifecycle.RequestedAt = now
			}
			lifecycle.SentAt = now
			if !lifecycle.TriggerOperationID.Valid {
				lifecycle.TriggerOperationID = sql.NullInt64{Int64: operation.ID, Valid: true}
			}
			lifecycle.LastOperationID = sql.NullInt64{Int64: operation.ID, Valid: true}

			if updateErr := tx.Update(lifecycle).Run(); updateErr != nil {
				return fmt.Errorf("update EBICS key lifecycle %d after send: %w", lifecycle.ID, updateErr)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	lifecycles, err = c.loadRotationGroup(input.EbicsSubscriberID, input.CoordinationID)
	if err != nil {
		return nil, err
	}

	return &KeyRotationGroupResult{
		CoordinationID: input.CoordinationID,
		Lifecycles:     lifecycles,
		Operations:     operations,
	}, nil
}

func (c *Client) confirmCoordinatedKeyRotation(input *KeyRotationActionInput) (*KeyRotationGroupResult, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}

	lifecycles, err := c.loadRotationGroup(input.EbicsSubscriberID, input.CoordinationID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	err = c.db.Transaction(func(tx *database.Session) error {
		for _, lifecycle := range lifecycles {
			if lifecycle.Status != model.EbicsKeyLifecycleStatusOrderSentForRuntime() &&
				lifecycle.Status != model.EbicsKeyLifecycleStatusWaitingBankConfirmationForRuntime() {
				return fmt.Errorf("%w: coordination=%s lifecycle=%d status=%s",
					errRotationRequiresSentGroup, input.CoordinationID, lifecycle.ID, lifecycle.Status)
			}

			current := &model.EbicsKeyLifecycle{}
			if getErr := tx.Get(
				current,
				"id<>? AND owner=? AND ebics_subscriber_id=? AND key_usage=? AND status=?",
				lifecycle.ID,
				lifecycle.Owner,
				lifecycle.EbicsSubscriberID,
				lifecycle.KeyUsage,
				model.EbicsKeyLifecycleStatusActivatedForRuntime(),
			).Run(); getErr != nil {
				return fmt.Errorf("%w: %s", errRotationMissingActivatedSource, getErr)
			}

			current.Status = model.EbicsKeyLifecycleStatusRetiredForRuntime()
			current.RetiredAt = now
			current.CoordinationID = input.CoordinationID
			if updateErr := tx.Update(current).Run(); updateErr != nil {
				return fmt.Errorf("retire previous EBICS key lifecycle %d: %w", current.ID, updateErr)
			}

			lifecycle.CurrentCredentialID = lifecycle.NextCredentialID.Int64
			lifecycle.NextCredentialID = sql.NullInt64{}
			lifecycle.Status = model.EbicsKeyLifecycleStatusActivatedForRuntime()
			lifecycle.ActivatedAt = now
			lifecycle.Operator = strings.TrimSpace(input.Operator)
			lifecycle.Reason = strings.TrimSpace(input.Reason)
			lifecycle.EvidenceMap = mergeRotationEvidence(input.Evidence, input.CoordinationID, input.SignatureOrderType)
			if updateErr := tx.Update(lifecycle).Run(); updateErr != nil {
				return fmt.Errorf("activate coordinated EBICS key lifecycle %d: %w", lifecycle.ID, updateErr)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	lifecycles, err = c.loadRotationGroup(input.EbicsSubscriberID, input.CoordinationID)
	if err != nil {
		return nil, err
	}

	return &KeyRotationGroupResult{
		CoordinationID: input.CoordinationID,
		Lifecycles:     lifecycles,
	}, nil
}

func (c *Client) finalizeCoordinatedKeyRotation(
	input *KeyRotationActionInput,
	targetStatus string,
) (*KeyRotationGroupResult, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}

	lifecycles, err := c.loadPendingRotationGroup(input.EbicsSubscriberID, input.CoordinationID)
	if err != nil {
		return nil, err
	}

	err = c.db.Transaction(func(tx *database.Session) error {
		for _, lifecycle := range lifecycles {
			lifecycle.Status = targetStatus
			lifecycle.Operator = strings.TrimSpace(input.Operator)
			lifecycle.Reason = strings.TrimSpace(input.Reason)
			lifecycle.EvidenceMap = mergeRotationEvidence(input.Evidence, input.CoordinationID, input.SignatureOrderType)
			if updateErr := tx.Update(lifecycle).Run(); updateErr != nil {
				return fmt.Errorf("finalize coordinated EBICS key lifecycle %d: %w", lifecycle.ID, updateErr)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	lifecycles, err = c.loadRotationGroup(input.EbicsSubscriberID, input.CoordinationID)
	if err != nil {
		return nil, err
	}

	return &KeyRotationGroupResult{
		CoordinationID: input.CoordinationID,
		Lifecycles:     lifecycles,
	}, nil
}

func (c *Client) revokeCoordinatedKeyRotation(input *KeyRotationActionInput) (*KeyRotationGroupResult, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}
	if strings.TrimSpace(input.CoordinationID) == "" {
		return nil, errMissingRotationCoordinationID
	}
	if len(input.SignatureData) == 0 {
		return nil, errMissingRotationSignatureData
	}

	lifecycles, err := c.loadPendingRotationGroup(input.EbicsSubscriberID, input.CoordinationID)
	if err != nil {
		return nil, err
	}

	execCtx, err := c.newAdminExecutionContext(input.EbicsSubscriberID)
	if err != nil {
		return nil, err
	}

	operation, err := c.insertNonPayloadOperation(execCtx, "KEY_MANAGEMENT", keyRotationOrderSPR, "OUTBOUND")
	if err != nil {
		return nil, err
	}

	requestCtx, cancel := context.WithTimeout(context.Background(), c.adminRequestTimeout())
	defer cancel()

	if execErr := execCtx.libClient.UploadSPR(requestCtx, libebicsclient.FlowSPRRequired{
		URL:           execCtx.endpointURL,
		HostID:        execCtx.host.HostID,
		PartnerID:     execCtx.subscriber.PartnerID,
		UserID:        execCtx.subscriber.UserID,
		SignatureData: input.SignatureData,
	}, libebicsclient.FlowSPROptional{
		ResponseSigner: execCtx.responseSigner,
	}); execErr != nil {
		return nil, c.failNonPayloadOperation(operation, "send coordinated SPR revocation", execErr)
	}

	operation.MetadataMap["keyRotation"] = map[string]any{
		"coordinationID": input.CoordinationID,
		"action":         keyRotationOrderSPR,
	}
	if completeErr := c.completeNonPayloadOperation(operation, "", ""); completeErr != nil {
		return nil, completeErr
	}

	now := time.Now().UTC()
	err = c.db.Transaction(func(tx *database.Session) error {
		for _, lifecycle := range lifecycles {
			lifecycle.Status = model.EbicsKeyLifecycleStatusCancelledForRuntime()
			lifecycle.Operator = strings.TrimSpace(input.Operator)
			lifecycle.Reason = strings.TrimSpace(input.Reason)
			lifecycle.EvidenceMap = mergeRotationEvidence(input.Evidence, input.CoordinationID, input.SignatureOrderType)
			lifecycle.LastOperationID = sql.NullInt64{Int64: operation.ID, Valid: true}
			if lifecycle.SentAt.IsZero() {
				lifecycle.SentAt = now
			}
			if updateErr := tx.Update(lifecycle).Run(); updateErr != nil {
				return fmt.Errorf("cancel coordinated EBICS key lifecycle %d after SPR: %w", lifecycle.ID, updateErr)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	lifecycles, err = c.loadRotationGroup(input.EbicsSubscriberID, input.CoordinationID)
	if err != nil {
		return nil, err
	}

	return &KeyRotationGroupResult{
		CoordinationID: input.CoordinationID,
		Lifecycles:     lifecycles,
		Operations:     []*model.EbicsOperation{operation},
	}, nil
}

func validateKeyRotationPrepareInput(input *KeyRotationPrepareInput) error {
	if input == nil {
		return database.NewValidationError("the coordinated EBICS key rotation request is missing")
	}
	if input.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS subscriber reference is missing")
	}

	selected := selectedRotationCredentials(input)
	if len(selected) == 0 {
		return errMissingRotationUsageSelection
	}

	_, hasAuth := selected[model.EbicsKeyUsageAuthenticationForRuntime()]
	_, hasEnc := selected[model.EbicsKeyUsageEncryptionForRuntime()]
	if hasAuth != hasEnc {
		return errIncompleteKeyPairRotation
	}

	if signatureOrderType := strings.TrimSpace(input.SignatureOrderType); signatureOrderType != "" &&
		signatureOrderType != keyRotationOrderPUB && signatureOrderType != keyRotationOrderHSA {
		return fmt.Errorf("%w: %s", errUnsupportedSignatureOrderType, signatureOrderType)
	}

	return nil
}

func selectedRotationCredentials(input *KeyRotationPrepareInput) map[string]*int64 {
	selected := map[string]*int64{}
	if input.NextAuthenticationCredentialID != nil {
		selected[model.EbicsKeyUsageAuthenticationForRuntime()] = input.NextAuthenticationCredentialID
	}
	if input.NextEncryptionCredentialID != nil {
		selected[model.EbicsKeyUsageEncryptionForRuntime()] = input.NextEncryptionCredentialID
	}
	if input.NextSignatureCredentialID != nil {
		selected[model.EbicsKeyUsageSignatureForRuntime()] = input.NextSignatureCredentialID
	}

	return selected
}

func defaultRotationType(value string) string {
	if strings.TrimSpace(value) == "" {
		return model.EbicsRotationTypeRotationForRuntime()
	}

	return strings.ToUpper(strings.TrimSpace(value))
}

func mergeRotationEvidence(base map[string]any, coordinationID, signatureOrderType string) map[string]any {
	merged := map[string]any{
		"coordinationID": coordinationID,
	}

	maps.Copy(merged, base)

	if orderType := strings.TrimSpace(signatureOrderType); orderType != "" {
		merged["signatureOrderType"] = strings.ToUpper(orderType)
	}

	return merged
}

func (c *Client) loadKeyLifecycleState(subscriberID int64) (*keyLifecycleState, error) {
	var lifecycles model.EbicsKeyLifecycles
	if err := c.db.Select(&lifecycles).Owner().Where(
		"ebics_subscriber_id=? AND status NOT IN (?, ?, ?)",
		subscriberID,
		model.EbicsKeyLifecycleStatusRetiredForRuntime(),
		model.EbicsKeyLifecycleStatusCancelledForRuntime(),
		model.EbicsKeyLifecycleStatusRejectedForRuntime(),
	).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS key lifecycle state for subscriber %d: %w", subscriberID, err)
	}

	state := &keyLifecycleState{
		activated: map[string]*model.EbicsKeyLifecycle{},
		pending:   map[string]*model.EbicsKeyLifecycle{},
	}
	for _, lifecycle := range lifecycles {
		switch lifecycle.Status {
		case model.EbicsKeyLifecycleStatusActivatedForRuntime():
			state.activated[lifecycle.KeyUsage] = lifecycle
		default:
			state.pending[lifecycle.KeyUsage] = lifecycle
		}
	}

	return state, nil
}

type keyLifecycleState struct {
	activated map[string]*model.EbicsKeyLifecycle
	pending   map[string]*model.EbicsKeyLifecycle
}

func (c *Client) loadRotationGroup(subscriberID int64, coordinationID string) ([]*model.EbicsKeyLifecycle, error) {
	if strings.TrimSpace(coordinationID) == "" {
		return nil, errMissingRotationCoordinationID
	}

	var lifecycles model.EbicsKeyLifecycles
	if err := c.db.Select(&lifecycles).Owner().Where(
		"ebics_subscriber_id=? AND coordination_id=?",
		subscriberID,
		strings.TrimSpace(coordinationID),
	).OrderBy("id", true).Run(); err != nil {
		return nil, fmt.Errorf("load coordinated EBICS key rotation group %q: %w", coordinationID, err)
	}

	if len(lifecycles) == 0 {
		return nil, errRotationRequiresPendingGroup
	}

	return lifecycles, nil
}

func (c *Client) loadPendingRotationGroup(
	subscriberID int64,
	coordinationID string,
) ([]*model.EbicsKeyLifecycle, error) {
	lifecycles, err := c.loadRotationGroup(subscriberID, coordinationID)
	if err != nil {
		return nil, err
	}

	pending := make([]*model.EbicsKeyLifecycle, 0, len(lifecycles))
	for _, lifecycle := range lifecycles {
		if lifecycle.Status == model.EbicsKeyLifecycleStatusActivatedForRuntime() ||
			lifecycle.Status == model.EbicsKeyLifecycleStatusRetiredForRuntime() ||
			lifecycle.Status == model.EbicsKeyLifecycleStatusCancelledForRuntime() ||
			lifecycle.Status == model.EbicsKeyLifecycleStatusRejectedForRuntime() {
			continue
		}

		pending = append(pending, lifecycle)
	}

	if len(pending) == 0 {
		return nil, errRotationRequiresPendingGroup
	}

	return pending, nil
}

func resolveRotationPlan(lifecycles []*model.EbicsKeyLifecycle, signatureOrderType string) (*rotationPlan, error) {
	usageSet := map[string]bool{}
	for _, lifecycle := range lifecycles {
		usageSet[lifecycle.KeyUsage] = true
	}

	hasAuth := usageSet[model.EbicsKeyUsageAuthenticationForRuntime()]
	hasEnc := usageSet[model.EbicsKeyUsageEncryptionForRuntime()]
	hasSig := usageSet[model.EbicsKeyUsageSignatureForRuntime()]

	switch {
	case hasAuth && hasEnc && hasSig:
		return &rotationPlan{
			orderType: keyRotationOrderHCS,
			usages: []string{
				model.EbicsKeyUsageAuthenticationForRuntime(),
				model.EbicsKeyUsageEncryptionForRuntime(),
				model.EbicsKeyUsageSignatureForRuntime(),
			},
		}, nil
	case hasAuth && hasEnc:
		return &rotationPlan{
			orderType: keyRotationOrderHCA,
			usages: []string{
				model.EbicsKeyUsageAuthenticationForRuntime(),
				model.EbicsKeyUsageEncryptionForRuntime(),
			},
		}, nil
	case hasSig:
		orderType := strings.ToUpper(strings.TrimSpace(signatureOrderType))
		if orderType == "" {
			orderType = defaultSignatureRotationOrder
		}
		if orderType != keyRotationOrderPUB && orderType != keyRotationOrderHSA {
			return nil, fmt.Errorf("%w: %s", errUnsupportedSignatureOrderType, orderType)
		}

		return &rotationPlan{
			orderType: orderType,
			usages:    []string{model.EbicsKeyUsageSignatureForRuntime()},
		}, nil
	default:
		return nil, errIncompleteKeyPairRotation
	}
}

func (c *Client) buildRotationOrderData(
	subscriber *model.EbicsSubscriber,
	lifecycles []*model.EbicsKeyLifecycle,
	plan *rotationPlan,
) ([]byte, error) {
	lifecycleByUsage := map[string]*model.EbicsKeyLifecycle{}
	for _, lifecycle := range lifecycles {
		lifecycleByUsage[lifecycle.KeyUsage] = lifecycle
	}

	switch plan.orderType {
	case keyRotationOrderPUB, keyRotationOrderHSA:
		signature, ok := lifecycleByUsage[model.EbicsKeyUsageSignatureForRuntime()]
		if !ok {
			return nil, errMissingRotationUsageSelection
		}

		return c.buildSignatureRotationOrderData(subscriber, signature.NextCredentialID.Int64, plan.orderType)
	case keyRotationOrderHCA:
		auth, ok := lifecycleByUsage[model.EbicsKeyUsageAuthenticationForRuntime()]
		if !ok {
			return nil, errMissingRotationUsageSelection
		}
		enc, ok := lifecycleByUsage[model.EbicsKeyUsageEncryptionForRuntime()]
		if !ok {
			return nil, errMissingRotationUsageSelection
		}

		return c.buildPairRotationOrderData(subscriber, auth.NextCredentialID.Int64, enc.NextCredentialID.Int64)
	case keyRotationOrderHCS:
		auth, ok := lifecycleByUsage[model.EbicsKeyUsageAuthenticationForRuntime()]
		if !ok {
			return nil, errMissingRotationUsageSelection
		}
		enc, ok := lifecycleByUsage[model.EbicsKeyUsageEncryptionForRuntime()]
		if !ok {
			return nil, errMissingRotationUsageSelection
		}
		sig, ok := lifecycleByUsage[model.EbicsKeyUsageSignatureForRuntime()]
		if !ok {
			return nil, errMissingRotationUsageSelection
		}

		return c.buildTripleRotationOrderData(
			subscriber,
			auth.NextCredentialID.Int64,
			enc.NextCredentialID.Int64,
			sig.NextCredentialID.Int64,
		)
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedRotationAction, plan.orderType)
	}
}

func (c *Client) executeRotationPlanStep(
	ctx context.Context,
	execCtx *adminExecutionContext,
	orderType string,
	orderData []byte,
) error {
	switch orderType {
	case keyRotationOrderPUB:
		if err := execCtx.libClient.UploadPUB(ctx, libebicsclient.FlowPUBRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		}, libebicsclient.FlowKeyMgmtOptional{
			ResponseSigner: execCtx.responseSigner,
		}, orderData); err != nil {
			return fmt.Errorf("upload PUB order: %w", err)
		}

		return nil
	case keyRotationOrderHCA:
		if err := execCtx.libClient.UploadHCA(ctx, libebicsclient.FlowHCARequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		}, libebicsclient.FlowKeyMgmtOptional{
			ResponseSigner: execCtx.responseSigner,
		}, orderData); err != nil {
			return fmt.Errorf("upload HCA order: %w", err)
		}

		return nil
	case keyRotationOrderHCS:
		if err := execCtx.libClient.UploadHCS(ctx, libebicsclient.FlowHCSRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		}, libebicsclient.FlowKeyMgmtOptional{
			ResponseSigner: execCtx.responseSigner,
		}, orderData); err != nil {
			return fmt.Errorf("upload HCS order: %w", err)
		}

		return nil
	case keyRotationOrderHSA:
		if err := execCtx.libClient.UploadHSA(ctx, libebicsclient.FlowHSARequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		}, libebicsclient.FlowKeyMgmtOptional{
			ResponseSigner: execCtx.responseSigner,
		}, orderData); err != nil {
			return fmt.Errorf("upload HSA order: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("%w: %s", errUnsupportedRotationAction, orderType)
	}
}

func (c *Client) buildSignatureRotationOrderData(
	subscriber *model.EbicsSubscriber,
	credentialID int64,
	orderType string,
) ([]byte, error) {
	credential, err := c.loadCredential(credentialID)
	if err != nil {
		return nil, err
	}

	cert, err := certificatePairFromCredential(credential)
	if err != nil {
		return nil, err
	}

	rootLocal := "HSARequestOrderData"
	if orderType == keyRotationOrderPUB {
		rootLocal = "PUBRequestOrderData"
	}

	doc := ebicsxml.SignatureKeyRequestOrderData{
		XMLName: xml.Name{Space: ebicsxml.NamespaceH005, Local: rootLocal},
		SignaturePubKeyInfo: ebicsxml.RawElement{
			InnerXML: signatureKeyInnerXML(cert, "A006"),
		},
		PartnerID: subscriber.PartnerID,
		UserID:    subscriber.UserID,
	}

	data, marshalErr := xml.Marshal(doc)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal %s request order data: %w", orderType, marshalErr)
	}

	return data, nil
}

func (c *Client) buildPairRotationOrderData(
	subscriber *model.EbicsSubscriber,
	authCredentialID, encCredentialID int64,
) ([]byte, error) {
	authCredential, err := c.loadCredential(authCredentialID)
	if err != nil {
		return nil, err
	}
	encCredential, err := c.loadCredential(encCredentialID)
	if err != nil {
		return nil, err
	}

	authCert, err := certificatePairFromCredential(authCredential)
	if err != nil {
		return nil, err
	}
	encCert, err := certificatePairFromCredential(encCredential)
	if err != nil {
		return nil, err
	}

	doc := ebicsxml.KeyPairRequestOrderData{
		XMLName: xml.Name{Space: ebicsxml.NamespaceH005, Local: "HCARequestOrderData"},
		AuthenticationPubKeyInfo: ebicsxml.RawElement{
			InnerXML: typedKeyInnerXML(authCert, "AuthenticationVersion", "X002"),
		},
		EncryptionPubKeyInfo: ebicsxml.RawElement{
			InnerXML: typedKeyInnerXML(encCert, "EncryptionVersion", "E002"),
		},
		PartnerID: subscriber.PartnerID,
		UserID:    subscriber.UserID,
	}

	data, marshalErr := xml.Marshal(doc)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal HCA request order data: %w", marshalErr)
	}

	return data, nil
}

func (c *Client) buildTripleRotationOrderData(
	subscriber *model.EbicsSubscriber,
	authCredentialID, encCredentialID, sigCredentialID int64,
) ([]byte, error) {
	authCredential, err := c.loadCredential(authCredentialID)
	if err != nil {
		return nil, err
	}
	encCredential, err := c.loadCredential(encCredentialID)
	if err != nil {
		return nil, err
	}
	sigCredential, err := c.loadCredential(sigCredentialID)
	if err != nil {
		return nil, err
	}

	authCert, err := certificatePairFromCredential(authCredential)
	if err != nil {
		return nil, err
	}
	encCert, err := certificatePairFromCredential(encCredential)
	if err != nil {
		return nil, err
	}
	sigCert, err := certificatePairFromCredential(sigCredential)
	if err != nil {
		return nil, err
	}

	doc := ebicsxml.KeyTripleRequestOrderData{
		XMLName: xml.Name{Space: ebicsxml.NamespaceH005, Local: "HCSRequestOrderData"},
		AuthenticationPubKeyInfo: ebicsxml.RawElement{
			InnerXML: typedKeyInnerXML(authCert, "AuthenticationVersion", "X002"),
		},
		EncryptionPubKeyInfo: ebicsxml.RawElement{
			InnerXML: typedKeyInnerXML(encCert, "EncryptionVersion", "E002"),
		},
		SignaturePubKeyInfo: ebicsxml.RawElement{
			InnerXML: signatureKeyInnerXML(sigCert, "A006"),
		},
		PartnerID: subscriber.PartnerID,
		UserID:    subscriber.UserID,
	}

	data, marshalErr := xml.Marshal(doc)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal HCS request order data: %w", marshalErr)
	}

	return data, nil
}

func (c *Client) loadCredential(credentialID int64) (*model.Credential, error) {
	credential := &model.Credential{}
	if err := c.db.Get(credential, "id=?", credentialID).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS credential %d: %w", credentialID, err)
	}

	return credential, nil
}
