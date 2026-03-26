package api

type OutEbicsOperation struct {
	ID                     int64          `json:"id" yaml:"id"`
	OperationType          string         `json:"operationType" yaml:"operationType"`
	OrderType              string         `json:"orderType" yaml:"orderType"`
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
