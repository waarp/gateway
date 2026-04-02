package api

import "time"

// OutEbicsInitializationWorkflow exposes the technical view of an EBICS initialization workflow.
type OutEbicsInitializationWorkflow struct {
	ID                int64          `json:"id" yaml:"id"`
	Status            string         `json:"status" yaml:"status"`
	CurrentStep       string         `json:"currentStep" yaml:"currentStep"`
	IniOperationID    *int64         `json:"iniOperationID,omitempty" yaml:"iniOperationID,omitempty"`
	HiaOperationID    *int64         `json:"hiaOperationID,omitempty" yaml:"hiaOperationID,omitempty"`
	H3KOperationID    *int64         `json:"h3KOperationID,omitempty" yaml:"h3KOperationID,omitempty"`
	LetterGeneratedAt *time.Time     `json:"letterGeneratedAt,omitempty" yaml:"letterGeneratedAt,omitempty"`
	LetterConfirmedAt *time.Time     `json:"letterConfirmedAt,omitempty" yaml:"letterConfirmedAt,omitempty"`
	BankActivationAt  *time.Time     `json:"bankActivationAt,omitempty" yaml:"bankActivationAt,omitempty"`
	Operator          string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason            string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	BankFeedback      string         `json:"bankFeedback,omitempty" yaml:"bankFeedback,omitempty"`
	Evidence          map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

// InEbicsInitializationAction defines an operator action on an EBICS initialization workflow.
type InEbicsInitializationAction struct {
	ClientID int64          `json:"clientID" yaml:"clientID"`
	Action   string         `json:"action" yaml:"action"`
	Operator string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}
