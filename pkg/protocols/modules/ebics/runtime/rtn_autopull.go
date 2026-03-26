package runtime

import (
	"errors"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var errUnsupportedRTNAutoPullPolicy = errors.New("unsupported RTN auto-pull policy")

const (
	rtnAutoPullPolicyManual       = "MANUAL"
	rtnAutoPullPolicyAuto         = "AUTO"
	rtnAutoPullPolicyAutoFiltered = "AUTO_FILTERED"
	rtnDefaultAutoPullOrderType   = "FDL"
)

// AutoPullPlan describes the normalized Gateway action derived from an RTN event.
type AutoPullPlan struct {
	Enabled       bool
	OrderType     string
	ProfileName   string
	CorrelationID string
	Reason        string
}

// BuildAutoPullPlan derives the auto-pull plan associated with a provider and an RTN event.
func BuildAutoPullPlan(event *model.EbicsRTNEvent, provider *model.EbicsRTNProvider) (*AutoPullPlan, error) {
	if err := CanAutoPull(event, provider); err != nil {
		return nil, err
	}

	plan := &AutoPullPlan{
		Enabled:       true,
		OrderType:     strings.ToUpper(strings.TrimSpace(event.OrderTypeHint)),
		ProfileName:   strings.TrimSpace(event.ProfileID),
		CorrelationID: strings.TrimSpace(event.CorrelationID),
		Reason:        "RTN auto-pull requested by provider policy",
	}

	if plan.OrderType == "" {
		plan.OrderType = rtnDefaultAutoPullOrderType
	}

	if plan.CorrelationID == "" {
		plan.CorrelationID = event.IdempotenceKey
	}

	if plan.ProfileName == "" && provider.AutoPullPolicy == rtnAutoPullPolicyAutoFiltered {
		return nil, database.NewValidationError(
			"the RTN auto-pull policy AUTO_FILTERED requires a profile reference on the event")
	}

	return plan, nil
}

// CanAutoPull validates whether the RTN event can trigger an auto-pull for the given provider.
func CanAutoPull(event *model.EbicsRTNEvent, provider *model.EbicsRTNProvider) error {
	if event == nil {
		return database.NewValidationError("the RTN event is missing")
	}

	if provider == nil {
		return database.NewValidationError("the RTN provider is missing")
	}

	if !provider.Enabled {
		return database.NewValidationError("the RTN provider is disabled")
	}

	switch provider.AutoPullPolicy {
	case rtnAutoPullPolicyManual:
		return database.NewValidationError("the RTN provider is configured for manual processing only")
	case rtnAutoPullPolicyAuto, rtnAutoPullPolicyAutoFiltered:
	default:
		return errUnsupportedRTNAutoPullPolicy
	}

	switch event.Status {
	case "RECEIVED", "PROCESSING", "RETRYABLE":
		return nil
	case "DUPLICATE":
		return database.NewValidationError("a duplicate RTN event cannot trigger an auto-pull")
	case "QUARANTINED":
		return database.NewValidationError("a quarantined RTN event cannot trigger an auto-pull")
	case "FAILED":
		return database.NewValidationError("a failed RTN event cannot trigger an auto-pull")
	default:
		return database.NewValidationErrorf("the RTN event status %q cannot trigger an auto-pull", event.Status)
	}
}
