package runtime

import (
	"errors"
	"slices"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var errUnsupportedInitializationAction = errors.New("unsupported EBICS initialization action")

const (
	initializationActionSendINI               = "SEND_INI"
	initializationActionSendHIA               = "SEND_HIA"
	initializationActionSendH3K               = "SEND_H3K"
	initializationActionConfirmLetter         = "CONFIRM_LETTER"
	initializationActionConfirmBankActivation = "CONFIRM_BANK_ACTIVATION"
	initializationActionCancel                = "CANCEL"

	initializationStatusActivated = "ACTIVATED"
	initializationStatusCancelled = "CANCELLED"
)

type InitializationAction struct {
	Action   string
	Operator string
	Reason   string
	Evidence map[string]any
}

type InitializationStore interface {
	GetInitializationByID(id int64) (*model.EbicsInitializationWorkflow, error)
	UpdateInitialization(workflow *model.EbicsInitializationWorkflow) error
}

// CanApplyInitializationAction validates whether the action is allowed for the current workflow step.
func CanApplyInitializationAction(workflow *model.EbicsInitializationWorkflow, action string) error {
	if workflow == nil {
		return database.NewValidationError("the EBICS initialization workflow is missing")
	}

	switch action {
	case initializationActionSendINI:
		return requireInitializationStep(workflow.CurrentStep, "DRAFT", "INI_PLANNED")
	case initializationActionSendHIA:
		return requireInitializationStep(workflow.CurrentStep, "INI_SENT", "HIA_PLANNED")
	case initializationActionSendH3K:
		return requireInitializationStep(workflow.CurrentStep, "HIA_SENT", "H3K_PLANNED")
	case initializationActionConfirmLetter:
		return requireInitializationStep(workflow.CurrentStep, "WAITING_LETTER_CONFIRMATION")
	case initializationActionConfirmBankActivation:
		return requireInitializationStep(workflow.CurrentStep, "WAITING_BANK_ACTIVATION")
	case initializationActionCancel:
		return requireInitializationStatus(
			workflow.Status,
			"RUNNING",
			"WAITING_EXTERNAL_CONFIRMATION",
			"WAITING_BANK_ACTIVATION",
		)
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS initialization action", action)
	}
}

// ApplyInitializationAction applies a controlled transition to an EBICS initialization workflow.
func ApplyInitializationAction(workflow *model.EbicsInitializationWorkflow, action InitializationAction) error {
	if err := CanApplyInitializationAction(workflow, action.Action); err != nil {
		return err
	}

	workflow.Operator = action.Operator
	workflow.Reason = action.Reason
	workflow.EvidenceMap = action.Evidence

	now := time.Now().UTC()

	switch action.Action {
	case initializationActionSendINI:
		workflow.Status = "RUNNING"
		workflow.CurrentStep = "INI_SENT"
	case initializationActionSendHIA:
		workflow.Status = "RUNNING"
		workflow.CurrentStep = "HIA_SENT"
	case initializationActionSendH3K:
		workflow.Status = "WAITING_EXTERNAL_CONFIRMATION"
		workflow.CurrentStep = "WAITING_LETTER_CONFIRMATION"
		if workflow.LetterGeneratedAt.IsZero() {
			workflow.LetterGeneratedAt = now
		}
	case initializationActionConfirmLetter:
		workflow.Status = "WAITING_BANK_ACTIVATION"
		workflow.CurrentStep = "WAITING_BANK_ACTIVATION"
		workflow.LetterConfirmedAt = now
	case initializationActionConfirmBankActivation:
		if len(action.Evidence) == 0 {
			return database.NewValidationError(
				"the EBICS bank activation confirmation requires structured evidence")
		}

		workflow.Status = initializationStatusActivated
		workflow.CurrentStep = "ACTIVATED"
		workflow.BankActivationAt = now
	case initializationActionCancel:
		workflow.Status = initializationStatusCancelled
		workflow.CurrentStep = initializationStatusCancelled
	default:
		return errUnsupportedInitializationAction
	}

	return nil
}

func requireInitializationStep(current string, allowed ...string) error {
	if slices.Contains(allowed, current) {
		return nil
	}

	return database.NewValidationErrorf(
		"the EBICS initialization action is not allowed from step %q", current)
}

func requireInitializationStatus(current string, allowed ...string) error {
	if slices.Contains(allowed, current) {
		return nil
	}

	return database.NewValidationErrorf(
		"the EBICS initialization action is not allowed from status %q", current)
}
