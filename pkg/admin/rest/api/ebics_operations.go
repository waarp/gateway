package api

import "time"

// OutEbicsOperation exposes the operational view of an EBICS operation.
type OutEbicsOperation struct {
	ID                     int64          `json:"id" yaml:"id"`
	OperationType          string         `json:"operationType" yaml:"operationType"`
	OrderType              string         `json:"orderType" yaml:"orderType"`
	SignatureState         string         `json:"signatureState,omitempty" yaml:"signatureState,omitempty"`
	Direction              string         `json:"direction" yaml:"direction"`
	TransportMode          string         `json:"transportMode" yaml:"transportMode"`
	Status                 string         `json:"status" yaml:"status"`
	Severity               string         `json:"severity" yaml:"severity"`
	TransactionID          string         `json:"transactionID,omitempty" yaml:"transactionID,omitempty"`
	RequestID              string         `json:"requestID,omitempty" yaml:"requestID,omitempty"`
	CorrelationID          string         `json:"correlationID,omitempty" yaml:"correlationID,omitempty"`
	TechnicalReturnCode    string         `json:"technicalReturnCode,omitempty" yaml:"technicalReturnCode,omitempty"`
	TechnicalReturnMessage string         `json:"technicalReturnMessage,omitempty" yaml:"technicalReturnMessage,omitempty"`
	BusinessReturnCode     string         `json:"businessReturnCode,omitempty" yaml:"businessReturnCode,omitempty"`
	BusinessReturnMessage  string         `json:"businessReturnMessage,omitempty" yaml:"businessReturnMessage,omitempty"`
	GatewayOutcome         string         `json:"gatewayOutcome,omitempty" yaml:"gatewayOutcome,omitempty"`
	RetryDecision          string         `json:"retryDecision,omitempty" yaml:"retryDecision,omitempty"`
	ManualActionRequired   bool           `json:"manualActionRequired" yaml:"manualActionRequired"`
	TransferID             *int64         `json:"transferID,omitempty" yaml:"transferID,omitempty"`
	Metadata               map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// OutEbicsOperationLinks exposes the cross-resource references attached to one
// EBICS operation detail.
type OutEbicsOperationLinks struct {
	TransferID     *int64 `json:"transferID,omitempty" yaml:"transferID,omitempty"`
	ContractViewID *int64 `json:"contractViewID,omitempty" yaml:"contractViewID,omitempty"`
	RTNEventID     *int64 `json:"rtnEventID,omitempty" yaml:"rtnEventID,omitempty"`
}

// OutEbicsOperationDetail exposes the detailed operational view of one EBICS
// operation together with its technical correlations.
type OutEbicsOperationDetail struct {
	Operation   *OutEbicsOperation            `json:"operation" yaml:"operation"`
	HostID      string                        `json:"hostID" yaml:"hostID"`
	PartnerID   string                        `json:"partnerID,omitempty" yaml:"partnerID,omitempty"`
	UserID      string                        `json:"userID,omitempty" yaml:"userID,omitempty"`
	StartedAt   *time.Time                    `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	FinishedAt  *time.Time                    `json:"finishedAt,omitempty" yaml:"finishedAt,omitempty"`
	Transaction *OutEbicsTransaction          `json:"transaction,omitempty" yaml:"transaction,omitempty"`
	Segments    []*OutEbicsTransactionSegment `json:"segments,omitempty" yaml:"segments,omitempty"`
	Links       *OutEbicsOperationLinks       `json:"links,omitempty" yaml:"links,omitempty"`
}

// InEbicsServiceRef identifies one EBICS service descriptor in admin actions.
type InEbicsServiceRef struct {
	ServiceName   string `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption string `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope         string `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName       string `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType string `json:"containerType,omitempty" yaml:"containerType,omitempty"`
}

// InEbicsReportingAction defines one client-side reporting/admin read action.
type InEbicsReportingAction struct {
	ClientID          int64                `json:"clientID" yaml:"clientID"`
	EbicsSubscriberID int64                `json:"ebicsSubscriberID" yaml:"ebicsSubscriberID"`
	OrderType         string               `json:"orderType" yaml:"orderType"`
	OrderID           string               `json:"orderID,omitempty" yaml:"orderID,omitempty"`
	Service           *InEbicsServiceRef   `json:"service,omitempty" yaml:"service,omitempty"`
	ServiceFilters    []*InEbicsServiceRef `json:"serviceFilters,omitempty" yaml:"serviceFilters,omitempty"`
	CompleteOrderData bool                 `json:"completeOrderData,omitempty" yaml:"completeOrderData,omitempty"`
	FetchLimit        int                  `json:"fetchLimit,omitempty" yaml:"fetchLimit,omitempty"`
	FetchOffset       int                  `json:"fetchOffset,omitempty" yaml:"fetchOffset,omitempty"`
	Metadata          map[string]any       `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// InEbicsSignatureAction defines one client-side signature action.
type InEbicsSignatureAction struct {
	ClientID          int64              `json:"clientID" yaml:"clientID"`
	EbicsSubscriberID int64              `json:"ebicsSubscriberID" yaml:"ebicsSubscriberID"`
	OrderType         string             `json:"orderType" yaml:"orderType"`
	OrderID           string             `json:"orderID,omitempty" yaml:"orderID,omitempty"`
	Service           *InEbicsServiceRef `json:"service,omitempty" yaml:"service,omitempty"`
	OrderData         []byte             `json:"orderData,omitempty" yaml:"orderData,omitempty"`
	SignatureData     []byte             `json:"signatureData,omitempty" yaml:"signatureData,omitempty"`
	Metadata          map[string]any     `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
