package model

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsInitializationStatusDraft                       = "DRAFT"
	ebicsInitializationStatusRunning                     = "RUNNING"
	ebicsInitializationStatusWaitingExternalConfirmation = "WAITING_EXTERNAL_CONFIRMATION"
	ebicsInitializationStatusWaitingBankActivation       = "WAITING_BANK_ACTIVATION"
	ebicsInitializationStatusActivated                   = "ACTIVATED"
	ebicsInitializationStatusCancelled                   = "CANCELLED"
	ebicsInitializationStatusRejected                    = "REJECTED"

	ebicsInitializationStepDraft                     = "DRAFT"
	ebicsInitializationStepINIPlanned                = "INI_PLANNED"
	ebicsInitializationStepINISent                   = "INI_SENT"
	ebicsInitializationStepHIAPlanned                = "HIA_PLANNED"
	ebicsInitializationStepHIASent                   = "HIA_SENT"
	ebicsInitializationStepH3KPlanned                = "H3K_PLANNED"
	ebicsInitializationStepH3KSent                   = "H3K_SENT"
	ebicsInitializationStepWaitingLetterConfirmation = "WAITING_LETTER_CONFIRMATION"
	ebicsInitializationStepWaitingBankActivation     = "WAITING_BANK_ACTIVATION"
	ebicsInitializationStepActivated                 = "ACTIVATED"
	ebicsInitializationStepCancelled                 = "CANCELLED"
)

// EbicsInitializationWorkflow stores the durable initialization workflow of an EBICS subscriber.
type EbicsInitializationWorkflow struct {
	ID                int64  `xorm:"<- id AUTOINCR"`
	Owner             string `xorm:"owner"`
	EbicsSubscriberID int64  `xorm:"ebics_subscriber_id"`

	Status            string        `xorm:"status"`
	CurrentStep       string        `xorm:"current_step"`
	IniOperationID    sql.NullInt64 `xorm:"ini_operation_id"`
	HiaOperationID    sql.NullInt64 `xorm:"hia_operation_id"`
	H3KOperationID    sql.NullInt64 `xorm:"h3k_operation_id"`
	LetterGeneratedAt time.Time     `xorm:"letter_generated_at DATETIME(6) UTC"`
	LetterConfirmedAt time.Time     `xorm:"letter_confirmed_at DATETIME(6) UTC"`
	BankActivationAt  time.Time     `xorm:"bank_activation_at DATETIME(6) UTC"`
	Operator          string        `xorm:"operator"`
	Reason            string        `xorm:"reason"`
	BankFeedback      string        `xorm:"bank_feedback"`
	Evidence          string        `xorm:"evidence"`
	CreatedAt         time.Time     `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt         time.Time     `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	EvidenceMap map[string]any `xorm:"-"`
}

// TableName returns the persistent table name for EBICS initialization workflows.
func (*EbicsInitializationWorkflow) TableName() string { return TableEbicsInitializationWorkflows }

// Appellation returns the display name used in validation messages.
func (*EbicsInitializationWorkflow) Appellation() string { return NameEbicsInitializationWorkflow }

// GetID returns the database identifier of the workflow.
func (w *EbicsInitializationWorkflow) GetID() int64 { return w.ID }

// BeforeDelete prevents removing an EBICS initialization workflow that still
// represents an active or auditable state.
func (w *EbicsInitializationWorkflow) BeforeDelete(database.Access) error {
	if isProtectedEbicsInitializationDeleteStatus(w.Status) {
		return database.NewValidationError(
			"this EBICS initialization workflow is still active and cannot be deleted")
	}

	return nil
}

// BeforeWrite normalizes and validates an EBICS initialization workflow before persistence.
func (w *EbicsInitializationWorkflow) BeforeWrite(db database.Access) error {
	w.Owner = conf.GlobalConfig.GatewayName
	w.Status = strings.ToUpper(strings.TrimSpace(w.Status))
	w.CurrentStep = strings.ToUpper(strings.TrimSpace(w.CurrentStep))
	w.Operator = strings.TrimSpace(w.Operator)
	w.Reason = strings.TrimSpace(w.Reason)
	w.BankFeedback = strings.TrimSpace(w.BankFeedback)
	w.Evidence = strings.TrimSpace(w.Evidence)

	if w.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS subscriber reference is missing")
	}

	if err := validateEbicsInitializationStatus(w.Status); err != nil {
		return err
	}

	if err := validateEbicsInitializationStep(w.CurrentStep); err != nil {
		return err
	}

	if err := w.hydrateEvidence(); err != nil {
		return err
	}

	if err := validateEbicsInitializationCoherence(w); err != nil {
		return err
	}

	return validateEbicsInitializationRefs(db, w)
}

func (w *EbicsInitializationWorkflow) hydrateEvidence() error {
	if w.EvidenceMap != nil {
		serialized, err := serializeStringMap(w.EvidenceMap)
		if err != nil {
			return fmt.Errorf("failed to serialize EBICS initialization evidence: %w", err)
		}

		w.Evidence = serialized
	} else if w.Evidence == "" {
		w.Evidence = emptyJSONObject
	}

	evidence, err := deserializeStringMap(w.Evidence)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS initialization evidence: %w", err)
	}

	w.EvidenceMap = evidence

	return nil
}

// AfterRead hydrates the transient evidence map after a database read.
func (w *EbicsInitializationWorkflow) AfterRead(database.ReadAccess) error {
	evidence, err := deserializeStringMap(w.Evidence)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS initialization evidence after read: %w", err)
	}

	w.EvidenceMap = evidence

	return nil
}

// AfterInsert refreshes transient state after insertion.
func (w *EbicsInitializationWorkflow) AfterInsert(db database.Access) error { return w.AfterRead(db) }

// AfterUpdate refreshes transient state after update.
func (w *EbicsInitializationWorkflow) AfterUpdate(db database.Access) error { return w.AfterRead(db) }

func validateEbicsInitializationStatus(value string) error {
	switch value {
	case ebicsInitializationStatusDraft, ebicsInitializationStatusRunning,
		ebicsInitializationStatusWaitingExternalConfirmation,
		ebicsInitializationStatusWaitingBankActivation,
		ebicsInitializationStatusActivated, ebicsInitializationStatusCancelled,
		ebicsInitializationStatusRejected:
		return nil
	case "":
		return database.NewValidationError("the EBICS initialization status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS initialization status", value)
	}
}

func validateEbicsInitializationStep(value string) error {
	switch value {
	case ebicsInitializationStepDraft, ebicsInitializationStepINIPlanned,
		ebicsInitializationStepINISent, ebicsInitializationStepHIAPlanned,
		ebicsInitializationStepHIASent, ebicsInitializationStepH3KPlanned,
		ebicsInitializationStepH3KSent, ebicsInitializationStepWaitingLetterConfirmation,
		ebicsInitializationStepWaitingBankActivation, ebicsInitializationStepActivated,
		ebicsInitializationStepCancelled:
		return nil
	case "":
		return database.NewValidationError("the EBICS initialization step is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS initialization step", value)
	}
}

func validateEbicsInitializationCoherence(workflow *EbicsInitializationWorkflow) error {
	if workflow.CurrentStep == ebicsInitializationStepActivated &&
		workflow.Status != ebicsInitializationStatusActivated {
		return database.NewValidationError(
			"an activated EBICS initialization step requires an activated workflow status")
	}

	if workflow.BankActivationAt.IsZero() {
		if workflow.Status == ebicsInitializationStatusActivated ||
			workflow.CurrentStep == ebicsInitializationStepActivated {
			return database.NewValidationError(
				"an activated EBICS initialization workflow requires a bank activation timestamp")
		}
	} else if len(workflow.EvidenceMap) == 0 {
		return database.NewValidationError(
			"an EBICS bank activation confirmation requires structured evidence")
	}

	if workflow.CurrentStep == ebicsInitializationStepDraft &&
		workflow.Status == ebicsInitializationStatusActivated {
		return database.NewValidationError(
			"an EBICS initialization workflow cannot move directly from draft to activated")
	}

	if workflow.CurrentStep == ebicsInitializationStepINISent &&
		workflow.Status == ebicsInitializationStatusActivated {
		return database.NewValidationError(
			"an EBICS initialization workflow cannot move directly from INI sent to activated")
	}

	if workflow.CurrentStep == ebicsInitializationStepWaitingLetterConfirmation &&
		workflow.Status == ebicsInitializationStatusActivated &&
		workflow.BankActivationAt.IsZero() {
		return database.NewValidationError(
			"an EBICS initialization workflow cannot activate without explicit bank confirmation")
	}

	if workflow.HiaOperationID.Valid && !workflow.IniOperationID.Valid {
		return database.NewValidationError(
			"an EBICS initialization workflow cannot reference HIA before INI")
	}

	if workflow.H3KOperationID.Valid && !workflow.HiaOperationID.Valid {
		return database.NewValidationError(
			"an EBICS initialization workflow cannot reference H3K before HIA")
	}

	return nil
}

func validateEbicsInitializationRefs(db database.Access, workflow *EbicsInitializationWorkflow) error {
	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", workflow.EbicsSubscriberID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", workflow.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber for initialization workflow: %w", err)
	}

	if workflow.IniOperationID.Valid {
		if err := validateEbicsOperationExists(db, workflow.IniOperationID.Int64, "INI"); err != nil {
			return err
		}
	}

	if workflow.HiaOperationID.Valid {
		if err := validateEbicsOperationExists(db, workflow.HiaOperationID.Int64, "HIA"); err != nil {
			return err
		}
	}

	if workflow.H3KOperationID.Valid {
		if err := validateEbicsOperationExists(db, workflow.H3KOperationID.Int64, "H3K"); err != nil {
			return err
		}
	}

	return nil
}

func validateCredentialExists(db database.Access, credentialID int64, role string) error {
	var credential Credential
	if err := db.Get(&credential, "id=?", credentialID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the %s credential %d does not exist", role, credentialID)
		}

		return fmt.Errorf("failed to retrieve %s credential: %w", role, err)
	}

	return nil
}

func validateEbicsOperationExists(db database.Access, operationID int64, label string) error {
	var operation EbicsOperation
	if err := db.Get(&operation, "id=?", operationID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the %s EBICS operation %d does not exist", label, operationID)
		}

		return fmt.Errorf("failed to retrieve %s EBICS operation: %w", label, err)
	}

	return nil
}

func isProtectedEbicsInitializationDeleteStatus(status string) bool {
	switch status {
	case ebicsInitializationStatusDraft, ebicsInitializationStatusCancelled, ebicsInitializationStatusRejected:
		return false
	default:
		return true
	}
}
