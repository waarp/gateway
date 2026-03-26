package api

import "time"

// OutEbicsKeyLifecycle exposes the technical view of an EBICS key lifecycle.
type OutEbicsKeyLifecycle struct {
	ID                  int64      `json:"id" yaml:"id"`
	KeyUsage            string     `json:"keyUsage" yaml:"keyUsage"`
	RotationType        string     `json:"rotationType" yaml:"rotationType"`
	Status              string     `json:"status" yaml:"status"`
	CurrentCredentialID int64      `json:"currentCredentialID" yaml:"currentCredentialID"`
	NextCredentialID    *int64     `json:"nextCredentialID,omitempty" yaml:"nextCredentialID,omitempty"`
	TriggerOperationID  *int64     `json:"triggerOperationID,omitempty" yaml:"triggerOperationID,omitempty"`
	LastOperationID     *int64     `json:"lastOperationID,omitempty" yaml:"lastOperationID,omitempty"`
	RequestedAt         *time.Time `json:"requestedAt,omitempty" yaml:"requestedAt,omitempty"`
	SentAt              *time.Time `json:"sentAt,omitempty" yaml:"sentAt,omitempty"`
	ActivatedAt         *time.Time `json:"activatedAt,omitempty" yaml:"activatedAt,omitempty"`
	RetiredAt           *time.Time `json:"retiredAt,omitempty" yaml:"retiredAt,omitempty"`
	Operator            string     `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason              string     `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// InEbicsKeyLifecycleAction defines an operator action on an EBICS key lifecycle.
type InEbicsKeyLifecycleAction struct {
	Action   string         `json:"action" yaml:"action"`
	Operator string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}
