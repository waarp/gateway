package api

import "time"

type OutEbicsRTNOutboundProvider struct {
	ID               int64      `json:"id" yaml:"id"`
	Name             string     `json:"name" yaml:"name"`
	Transport        string     `json:"transport" yaml:"transport"`
	Enabled          bool       `json:"enabled" yaml:"enabled"`
	SubscriberID     int64      `json:"subscriberID" yaml:"subscriberID"`
	ActivationStatus string     `json:"activationStatus,omitempty" yaml:"activationStatus,omitempty"`
	ActivationReason string     `json:"activationReason,omitempty" yaml:"activationReason,omitempty"`
	LastConnectionAt *time.Time `json:"lastConnectionAt,omitempty" yaml:"lastConnectionAt,omitempty"`
	LastError        string     `json:"lastError,omitempty" yaml:"lastError,omitempty"`
}

type InEbicsRTNOutboundProvider struct {
	Name          string         `json:"name" yaml:"name"`
	Transport     string         `json:"transport" yaml:"transport"`
	Enabled       *bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	SubscriberID  int64          `json:"subscriberID" yaml:"subscriberID"`
	Configuration map[string]any `json:"configuration,omitempty" yaml:"configuration,omitempty"`
}

type OutEbicsRTNOutboundNotification struct {
	ID                     int64          `json:"id" yaml:"id"`
	ProviderID             int64          `json:"providerID" yaml:"providerID"`
	EventType              string         `json:"eventType" yaml:"eventType"`
	SourceOrderType        string         `json:"sourceOrderType" yaml:"sourceOrderType"`
	CorrelationID          string         `json:"correlationID,omitempty" yaml:"correlationID,omitempty"`
	SubscriberID           int64          `json:"subscriberID" yaml:"subscriberID"`
	ServerReportingSetID   *int64         `json:"serverReportingSetID,omitempty" yaml:"serverReportingSetID,omitempty"`
	ServerReportingItemKey string         `json:"serverReportingItemKey,omitempty" yaml:"serverReportingItemKey,omitempty"`
	Status                 string         `json:"status" yaml:"status"`
	Attempts               int            `json:"attempts" yaml:"attempts"`
	NextRetryAt            *time.Time     `json:"nextRetryAt,omitempty" yaml:"nextRetryAt,omitempty"`
	SentAt                 *time.Time     `json:"sentAt,omitempty" yaml:"sentAt,omitempty"`
	LastError              string         `json:"lastError,omitempty" yaml:"lastError,omitempty"`
	Payload                map[string]any `json:"payload,omitempty" yaml:"payload,omitempty"`
}

type InEbicsRTNOutboundNotification struct {
	ProviderID           int64  `json:"providerID" yaml:"providerID"`
	ServerReportingSetID int64  `json:"serverReportingSetID" yaml:"serverReportingSetID"`
	ItemKey              string `json:"itemKey" yaml:"itemKey"`
}

type InEbicsRTNOutboundNotificationAction struct {
	Action string `json:"action" yaml:"action"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}
