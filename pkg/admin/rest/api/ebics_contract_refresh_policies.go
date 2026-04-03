package api

import "time"

type OutEbicsContractRefreshPolicy struct {
	ID           int64  `json:"id" yaml:"id"`
	Name         string `json:"name" yaml:"name"`
	Enabled      bool   `json:"enabled" yaml:"enabled"`
	ClientID     int64  `json:"clientID" yaml:"clientID"`
	ClientName   string `json:"clientName,omitempty" yaml:"clientName,omitempty"`
	SubscriberID int64  `json:"subscriberID" yaml:"subscriberID"`
	HostID       string `json:"hostID,omitempty" yaml:"hostID,omitempty"`
	PartnerID    string `json:"partnerID,omitempty" yaml:"partnerID,omitempty"`
	UserID       string `json:"userID,omitempty" yaml:"userID,omitempty"`
	//nolint:tagliatelle // preserve existing EBICS HEV acronym in API
	IncludeHEV       bool       `json:"includeHEV" yaml:"includeHEV"`
	IntervalSeconds  int64      `json:"intervalSeconds" yaml:"intervalSeconds"`
	Status           string     `json:"status" yaml:"status"`
	NextRunAt        *time.Time `json:"nextRunAt,omitempty" yaml:"nextRunAt,omitempty"`
	LastAttemptAt    *time.Time `json:"lastAttemptAt,omitempty" yaml:"lastAttemptAt,omitempty"`
	LastSuccessAt    *time.Time `json:"lastSuccessAt,omitempty" yaml:"lastSuccessAt,omitempty"`
	LastError        string     `json:"lastError,omitempty" yaml:"lastError,omitempty"`
	ActivationStatus string     `json:"activationStatus,omitempty" yaml:"activationStatus,omitempty"`
	ActivationReason string     `json:"activationReason,omitempty" yaml:"activationReason,omitempty"`
}

type InEbicsContractRefreshPolicy struct {
	Name         string `json:"name" yaml:"name"`
	Enabled      *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	ClientID     int64  `json:"clientID" yaml:"clientID"`
	SubscriberID int64  `json:"subscriberID" yaml:"subscriberID"`
	//nolint:tagliatelle // preserve existing EBICS HEV acronym in API
	IncludeHEV      *bool `json:"includeHEV,omitempty" yaml:"includeHEV,omitempty"`
	IntervalSeconds int64 `json:"intervalSeconds" yaml:"intervalSeconds"`
}

type PatchEbicsContractRefreshPolicyReqObject struct {
	Name         Nullable[string] `json:"name" yaml:"name"`
	Enabled      Nullable[bool]   `json:"enabled" yaml:"enabled"`
	ClientID     Nullable[int64]  `json:"clientID" yaml:"clientID"`
	SubscriberID Nullable[int64]  `json:"subscriberID" yaml:"subscriberID"`
	//nolint:tagliatelle // preserve existing EBICS HEV acronym in API
	IncludeHEV      Nullable[bool]  `json:"includeHEV" yaml:"includeHEV"`
	IntervalSeconds Nullable[int64] `json:"intervalSeconds" yaml:"intervalSeconds"`
}
