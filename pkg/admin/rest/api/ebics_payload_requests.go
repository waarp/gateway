package api

type InEbicsPayloadRequest struct {
	Profile    string            `json:"profile,omitempty" yaml:"profile,omitempty"`
	Rule       string            `json:"rule,omitempty" yaml:"rule,omitempty"`
	Subscriber InSubscriberRef   `json:"subscriber" yaml:"subscriber"`
	File       *InPayloadFile    `json:"file,omitempty" yaml:"file,omitempty"`
	Target     *InPayloadTarget  `json:"target,omitempty" yaml:"target,omitempty"`
	Service    *InPayloadService `json:"service,omitempty" yaml:"service,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type InSubscriberRef struct {
	HostID    string `json:"hostID" yaml:"hostID"`
	PartnerID string `json:"partnerID" yaml:"partnerID"`
	UserID    string `json:"userID" yaml:"userID"`
}

type InPayloadFile struct {
	Path       string `json:"path" yaml:"path"`
	OutputName string `json:"outputName,omitempty" yaml:"outputName,omitempty"`
}

type InPayloadTarget struct {
	Directory string `json:"directory,omitempty" yaml:"directory,omitempty"`
}

type InPayloadService struct {
	OrderType     string `json:"orderType,omitempty" yaml:"orderType,omitempty"`
	ServiceName   string `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption string `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope         string `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName       string `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType string `json:"containerType,omitempty" yaml:"containerType,omitempty"`
}

type OutEbicsPayloadSubmission struct {
	OperationID            int64   `json:"operationID" yaml:"operationID"`
	OrderType              string  `json:"orderType" yaml:"orderType"`
	Status                 string  `json:"status" yaml:"status"`
	CorrelationID          string  `json:"correlationID,omitempty" yaml:"correlationID,omitempty"`
	TransferID             *int64  `json:"transferID,omitempty" yaml:"transferID,omitempty"`
	ContractViewID         *int64  `json:"contractViewID,omitempty" yaml:"contractViewID,omitempty"`
	MatchedContractItemIDs []int64 `json:"matchedContractItemIDs,omitempty" yaml:"matchedContractItemIDs,omitempty"`
}
