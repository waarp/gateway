package runtime

import (
	"errors"
	"slices"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var errUnsupportedKeyLifecycleAction = errors.New("unsupported EBICS key lifecycle action")

const (
	keyLifecycleActionPrepareMaterial = "PREPARE_MATERIAL"
	keyLifecycleActionPlanOrder       = "PLAN_ORDER"
	keyLifecycleActionMarkSent        = "MARK_SENT"
	keyLifecycleActionConfirmBank     = "CONFIRM_BANK_ACTIVATION"
	keyLifecycleActionRetire          = "RETIRE"
	keyLifecycleActionCancel          = "CANCEL"
	keyLifecycleActionReject          = "REJECT"
)

type KeyLifecycleAction struct {
	Action   string
	Operator string
	Reason   string
	Evidence map[string]any
}

type KeyLifecycleStore interface {
	GetKeyLifecycleByID(id int64) (*model.EbicsKeyLifecycle, error)
	UpdateKeyLifecycle(lifecycle *model.EbicsKeyLifecycle) error
}

type CredentialGuard interface {
	CheckCredentialUsable(id int64) error
}

// CanApplyKeyLifecycleAction validates whether the action is allowed for the current lifecycle status.
func CanApplyKeyLifecycleAction(lifecycle *model.EbicsKeyLifecycle, action string) error {
	if lifecycle == nil {
		return database.NewValidationError("the EBICS key lifecycle is missing")
	}

	switch action {
	case keyLifecycleActionPrepareMaterial:
		return requireKeyLifecycleStatus(lifecycle.Status, "DRAFT")
	case keyLifecycleActionPlanOrder:
		return requireKeyLifecycleStatus(lifecycle.Status, "MATERIAL_PREPARED")
	case keyLifecycleActionMarkSent:
		return requireKeyLifecycleStatus(lifecycle.Status, "ORDER_PLANNED")
	case keyLifecycleActionConfirmBank:
		return requireKeyLifecycleStatus(lifecycle.Status, "ORDER_SENT", "WAITING_BANK_CONFIRMATION")
	case keyLifecycleActionRetire:
		return requireKeyLifecycleStatus(lifecycle.Status, "ACTIVATED")
	case keyLifecycleActionCancel:
		return requireKeyLifecycleStatus(lifecycle.Status, "DRAFT", "MATERIAL_PREPARED", "ORDER_PLANNED")
	case keyLifecycleActionReject:
		return requireKeyLifecycleStatus(lifecycle.Status, "ORDER_SENT", "WAITING_BANK_CONFIRMATION")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS key lifecycle action", action)
	}
}

// ApplyKeyLifecycleAction applies a controlled transition to an EBICS key lifecycle.
func ApplyKeyLifecycleAction(lifecycle *model.EbicsKeyLifecycle, action KeyLifecycleAction) error {
	if err := CanApplyKeyLifecycleAction(lifecycle, action.Action); err != nil {
		return err
	}

	lifecycle.Operator = action.Operator
	lifecycle.Reason = action.Reason
	lifecycle.EvidenceMap = action.Evidence

	now := time.Now().UTC()

	switch action.Action {
	case keyLifecycleActionPrepareMaterial:
		lifecycle.Status = "MATERIAL_PREPARED"
		if lifecycle.RequestedAt.IsZero() {
			lifecycle.RequestedAt = now
		}
	case keyLifecycleActionPlanOrder:
		lifecycle.Status = "ORDER_PLANNED"
	case keyLifecycleActionMarkSent:
		lifecycle.Status = "ORDER_SENT"
		if lifecycle.RequestedAt.IsZero() {
			lifecycle.RequestedAt = now
		}
		lifecycle.SentAt = now
	case keyLifecycleActionConfirmBank:
		lifecycle.Status = "ACTIVATED"
		if lifecycle.SentAt.IsZero() {
			return database.NewValidationError("the EBICS key lifecycle cannot be activated before the order is sent")
		}
		lifecycle.ActivatedAt = now
	case keyLifecycleActionRetire:
		lifecycle.Status = "RETIRED"
		lifecycle.RetiredAt = now
	case keyLifecycleActionCancel:
		lifecycle.Status = "CANCELLED"
	case keyLifecycleActionReject:
		lifecycle.Status = "REJECTED"
	default:
		return errUnsupportedKeyLifecycleAction
	}

	return nil
}

func requireKeyLifecycleStatus(current string, allowed ...string) error {
	if slices.Contains(allowed, current) {
		return nil
	}

	return database.NewValidationErrorf(
		"the EBICS key lifecycle action is not allowed from status %q", current)
}
