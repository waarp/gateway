package api

import "time"

// OutEbicsRTNEvent exposes the operational view of a persisted RTN event.
type OutEbicsRTNEvent struct {
	ID                  int64          `json:"id" yaml:"id"`
	Source              string         `json:"source" yaml:"source"`
	EventID             string         `json:"eventID,omitempty" yaml:"eventID,omitempty"`
	CorrelationID       string         `json:"correlationID,omitempty" yaml:"correlationID,omitempty"`
	IdempotenceKey      string         `json:"idempotenceKey" yaml:"idempotenceKey"`
	OrderTypeHint       string         `json:"orderTypeHint,omitempty" yaml:"orderTypeHint,omitempty"`
	ProfileID           string         `json:"profileID,omitempty" yaml:"profileID,omitempty"`
	Status              string         `json:"status" yaml:"status"`
	Attempts            int            `json:"attempts" yaml:"attempts"`
	NextRetryAt         *time.Time     `json:"nextRetryAt,omitempty" yaml:"nextRetryAt,omitempty"`
	ReceivedAt          time.Time      `json:"receivedAt" yaml:"receivedAt"`
	ProcessedAt         *time.Time     `json:"processedAt,omitempty" yaml:"processedAt,omitempty"`
	LastError           string         `json:"lastError,omitempty" yaml:"lastError,omitempty"`
	AutoPullOperationID *int64         `json:"autoPullOperationID,omitempty" yaml:"autoPullOperationID,omitempty"`
	AutoPullTransferID  *int64         `json:"autoPullTransferID,omitempty" yaml:"autoPullTransferID,omitempty"`
	AutoPullOrderType   string         `json:"autoPullOrderType,omitempty" yaml:"autoPullOrderType,omitempty"`
	AutoPullStatus      string         `json:"autoPullStatus,omitempty" yaml:"autoPullStatus,omitempty"`
	AutoPullOutcome     string         `json:"autoPullOutcome,omitempty" yaml:"autoPullOutcome,omitempty"`
	AutoPullRetry       string         `json:"autoPullRetry,omitempty" yaml:"autoPullRetry,omitempty"`
	OperatorAction      string         `json:"operatorAction,omitempty" yaml:"operatorAction,omitempty"`
	OperatorReason      string         `json:"operatorReason,omitempty" yaml:"operatorReason,omitempty"`
	OperatorMetadata    map[string]any `json:"operatorMetadata,omitempty" yaml:"operatorMetadata,omitempty"`
}

// OutEbicsRTNProvider exposes the operational view of an RTN provider.
type OutEbicsRTNProvider struct {
	ID               int64      `json:"id" yaml:"id"`
	Name             string     `json:"name" yaml:"name"`
	Transport        string     `json:"transport" yaml:"transport"`
	Enabled          bool       `json:"enabled" yaml:"enabled"`
	SubscriberID     int64      `json:"subscriberID" yaml:"subscriberID"`
	ClientID         *int64     `json:"clientID,omitempty" yaml:"clientID,omitempty"`
	AutoPullPolicy   string     `json:"autoPullPolicy" yaml:"autoPullPolicy"`
	LastConnectionAt *time.Time `json:"lastConnectionAt,omitempty" yaml:"lastConnectionAt,omitempty"`
	LastError        string     `json:"lastError,omitempty" yaml:"lastError,omitempty"`
}

// InEbicsRTNProvider defines the creation/update contract of an RTN provider.
type InEbicsRTNProvider struct {
	Name           string         `json:"name" yaml:"name"`
	Transport      string         `json:"transport" yaml:"transport"`
	Enabled        *bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	SubscriberID   int64          `json:"subscriberID" yaml:"subscriberID"`
	ClientID       *int64         `json:"clientID,omitempty" yaml:"clientID,omitempty"`
	Configuration  map[string]any `json:"configuration,omitempty" yaml:"configuration,omitempty"`
	AutoPullPolicy string         `json:"autoPullPolicy,omitempty" yaml:"autoPullPolicy,omitempty"`
}

// InEbicsRTNEventAction defines an operator action on an RTN event.
type InEbicsRTNEventAction struct {
	Action   string         `json:"action" yaml:"action"`
	Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
