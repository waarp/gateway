package api

import "time"

// OutEbicsKeyLifecycle exposes the technical view of an EBICS key lifecycle.
type OutEbicsKeyLifecycle struct {
	ID                  int64          `json:"id" yaml:"id"`
	KeyUsage            string         `json:"keyUsage" yaml:"keyUsage"`
	RotationType        string         `json:"rotationType" yaml:"rotationType"`
	CoordinationID      string         `json:"coordinationID,omitempty" yaml:"coordinationID,omitempty"`
	Status              string         `json:"status" yaml:"status"`
	CurrentCredentialID int64          `json:"currentCredentialID" yaml:"currentCredentialID"`
	NextCredentialID    *int64         `json:"nextCredentialID,omitempty" yaml:"nextCredentialID,omitempty"`
	TriggerOperationID  *int64         `json:"triggerOperationID,omitempty" yaml:"triggerOperationID,omitempty"`
	LastOperationID     *int64         `json:"lastOperationID,omitempty" yaml:"lastOperationID,omitempty"`
	RequestedAt         *time.Time     `json:"requestedAt,omitempty" yaml:"requestedAt,omitempty"`
	SentAt              *time.Time     `json:"sentAt,omitempty" yaml:"sentAt,omitempty"`
	ActivatedAt         *time.Time     `json:"activatedAt,omitempty" yaml:"activatedAt,omitempty"`
	RetiredAt           *time.Time     `json:"retiredAt,omitempty" yaml:"retiredAt,omitempty"`
	Operator            string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason              string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence            map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

// InEbicsKeyLifecycleAction defines an operator action on an EBICS key lifecycle.
type InEbicsKeyLifecycleAction struct {
	Action   string         `json:"action" yaml:"action"`
	Operator string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

// InEbicsKeyRotationPrepare defines a coordinated EBICS key rotation preparation request.
//
//nolint:lll // JSON/YAML field names are intentionally explicit for admin APIs.
type InEbicsKeyRotationPrepare struct {
	ClientID                       int64          `json:"clientID" yaml:"clientID"`
	EbicsSubscriberID              int64          `json:"ebicsSubscriberID" yaml:"ebicsSubscriberID"`
	RotationType                   string         `json:"rotationType,omitempty" yaml:"rotationType,omitempty"`
	CoordinationID                 string         `json:"coordinationID,omitempty" yaml:"coordinationID,omitempty"`
	NextAuthenticationCredentialID *int64         `json:"nextAuthenticationCredentialID,omitempty" yaml:"nextAuthenticationCredentialID,omitempty"`
	NextEncryptionCredentialID     *int64         `json:"nextEncryptionCredentialID,omitempty" yaml:"nextEncryptionCredentialID,omitempty"`
	NextSignatureCredentialID      *int64         `json:"nextSignatureCredentialID,omitempty" yaml:"nextSignatureCredentialID,omitempty"`
	SignatureOrderType             string         `json:"signatureOrderType,omitempty" yaml:"signatureOrderType,omitempty"`
	Operator                       string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason                         string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence                       map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

// InEbicsKeyRotationAction defines a coordinated action on a prepared EBICS key rotation.
type InEbicsKeyRotationAction struct {
	ClientID           int64          `json:"clientID" yaml:"clientID"`
	EbicsSubscriberID  int64          `json:"ebicsSubscriberID" yaml:"ebicsSubscriberID"`
	CoordinationID     string         `json:"coordinationID" yaml:"coordinationID"`
	SignatureOrderType string         `json:"signatureOrderType,omitempty" yaml:"signatureOrderType,omitempty"`
	SignatureData      []byte         `json:"signatureData,omitempty" yaml:"signatureData,omitempty"`
	Operator           string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason             string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence           map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

// OutEbicsKeyRotationGroup exposes one coordinated EBICS key rotation group.
type OutEbicsKeyRotationGroup struct {
	CoordinationID string                  `json:"coordinationID" yaml:"coordinationID"`
	Lifecycles     []*OutEbicsKeyLifecycle `json:"lifecycles" yaml:"lifecycles"`
	Operations     []*OutEbicsOperation    `json:"operations,omitempty" yaml:"operations,omitempty"`
}
