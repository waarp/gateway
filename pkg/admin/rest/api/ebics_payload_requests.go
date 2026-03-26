package api

// InEbicsPayloadRequest defines the payload submission contract for EBICS orders.
type InEbicsPayloadRequest struct {
	Profile    string            `json:"profile,omitempty" yaml:"profile,omitempty"`
	Rule       string            `json:"rule,omitempty" yaml:"rule,omitempty"`
	Subscriber InSubscriberRef   `json:"subscriber" yaml:"subscriber"`
	File       *InPayloadFile    `json:"file,omitempty" yaml:"file,omitempty"`
	Target     *InPayloadTarget  `json:"target,omitempty" yaml:"target,omitempty"`
	Service    *InPayloadService `json:"service,omitempty" yaml:"service,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// InSubscriberRef identifies the EBICS subscriber targeted by the request.
type InSubscriberRef struct {
	HostID    string `json:"hostID" yaml:"hostID"`
	PartnerID string `json:"partnerID" yaml:"partnerID"`
	UserID    string `json:"userID" yaml:"userID"`
}

// InPayloadFile describes the source file for upload-oriented payload requests.
type InPayloadFile struct {
	Path       string `json:"path" yaml:"path"`
	OutputName string `json:"outputName,omitempty" yaml:"outputName,omitempty"`
}

// InPayloadTarget describes the output target for download-oriented payload requests.
type InPayloadTarget struct {
	Directory string `json:"directory,omitempty" yaml:"directory,omitempty"`
}

// InPayloadService carries the service envelope fields required by EBICS payload orders.
type InPayloadService struct {
	OrderType     string `json:"orderType,omitempty" yaml:"orderType,omitempty"`
	ServiceName   string `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption string `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope         string `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName       string `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType string `json:"containerType,omitempty" yaml:"containerType,omitempty"`
}

// OutEbicsPayloadSubmission returns the identifiers created for a payload submission.
type OutEbicsPayloadSubmission struct {
	OperationID            int64   `json:"operationID" yaml:"operationID"`
	OrderType              string  `json:"orderType" yaml:"orderType"`
	Status                 string  `json:"status" yaml:"status"`
	CorrelationID          string  `json:"correlationID,omitempty" yaml:"correlationID,omitempty"`
	TransferID             *int64  `json:"transferID,omitempty" yaml:"transferID,omitempty"`
	ContractViewID         *int64  `json:"contractViewID,omitempty" yaml:"contractViewID,omitempty"`
	MatchedContractItemIDs []int64 `json:"matchedContractItemIDs,omitempty" yaml:"matchedContractItemIDs,omitempty"`
}

// InEbicsPayloadAction defines an operator action on a payload-bound EBICS operation.
type InEbicsPayloadAction struct {
	Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
