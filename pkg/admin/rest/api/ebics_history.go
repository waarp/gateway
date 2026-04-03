package api

import "time"

// OutEbicsHistoryEntry exposes one append-only non-payload EBICS history entry.
type OutEbicsHistoryEntry struct {
	ID                     int64          `json:"id" yaml:"id"`
	HistoryType            string         `json:"historyType" yaml:"historyType"`
	OperationType          string         `json:"operationType" yaml:"operationType"`
	Action                 string         `json:"action,omitempty" yaml:"action,omitempty"`
	OrderType              string         `json:"orderType,omitempty" yaml:"orderType,omitempty"`
	Direction              string         `json:"direction,omitempty" yaml:"direction,omitempty"`
	TransportMode          string         `json:"transportMode,omitempty" yaml:"transportMode,omitempty"`
	Status                 string         `json:"status" yaml:"status"`
	Severity               string         `json:"severity,omitempty" yaml:"severity,omitempty"`
	TechnicalReturnCode    string         `json:"technicalReturnCode,omitempty" yaml:"technicalReturnCode,omitempty"`
	TechnicalReturnMessage string         `json:"technicalReturnMessage,omitempty" yaml:"technicalReturnMessage,omitempty"`
	BusinessReturnCode     string         `json:"businessReturnCode,omitempty" yaml:"businessReturnCode,omitempty"`
	BusinessReturnMessage  string         `json:"businessReturnMessage,omitempty" yaml:"businessReturnMessage,omitempty"`
	GatewayOutcome         string         `json:"gatewayOutcome,omitempty" yaml:"gatewayOutcome,omitempty"`
	RetryDecision          string         `json:"retryDecision,omitempty" yaml:"retryDecision,omitempty"`
	ClientID               *int64         `json:"clientID,omitempty" yaml:"clientID,omitempty"`
	HostID                 string         `json:"hostID" yaml:"hostID"`
	PartnerID              string         `json:"partnerID,omitempty" yaml:"partnerID,omitempty"`
	UserID                 string         `json:"userID,omitempty" yaml:"userID,omitempty"`
	OperationID            *int64         `json:"operationID,omitempty" yaml:"operationID,omitempty"`
	TransferID             *int64         `json:"transferID,omitempty" yaml:"transferID,omitempty"`
	WorkflowID             *int64         `json:"workflowID,omitempty" yaml:"workflowID,omitempty"`
	LifecycleID            *int64         `json:"lifecycleID,omitempty" yaml:"lifecycleID,omitempty"`
	CoordinationID         string         `json:"coordinationID,omitempty" yaml:"coordinationID,omitempty"`
	RequestID              string         `json:"requestID,omitempty" yaml:"requestID,omitempty"`
	CorrelationID          string         `json:"correlationID,omitempty" yaml:"correlationID,omitempty"`
	TransactionID          string         `json:"transactionID,omitempty" yaml:"transactionID,omitempty"`
	Operator               string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason                 string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence               map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Metadata               map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	StartedAt              *time.Time     `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	FinishedAt             *time.Time     `json:"finishedAt,omitempty" yaml:"finishedAt,omitempty"`
	CreatedAt              time.Time      `json:"createdAt" yaml:"createdAt"`
}
