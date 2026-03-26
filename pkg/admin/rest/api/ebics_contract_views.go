package api

import "time"

type OutEbicsContractView struct {
	ID              int64     `json:"id" yaml:"id"`
	HostID          string    `json:"hostID" yaml:"hostID"`
	PartnerID       string    `json:"partnerID,omitempty" yaml:"partnerID,omitempty"`
	UserID          string    `json:"userID,omitempty" yaml:"userID,omitempty"`
	SourceOrderType string    `json:"sourceOrderType" yaml:"sourceOrderType"`
	VersionTag      string    `json:"versionTag,omitempty" yaml:"versionTag,omitempty"`
	Status          string    `json:"status" yaml:"status"`
	FetchedAt       time.Time `json:"fetchedAt" yaml:"fetchedAt"`
}

type OutEbicsContractViewItem struct {
	ID                 int64  `json:"id" yaml:"id"`
	ItemType           string `json:"itemType" yaml:"itemType"`
	ItemKey            string `json:"itemKey" yaml:"itemKey"`
	OrderType          string `json:"orderType,omitempty" yaml:"orderType,omitempty"`
	ServiceName        string `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption      string `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope              string `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName            string `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType      string `json:"containerType,omitempty" yaml:"containerType,omitempty"`
	AdminOrderType     string `json:"adminOrderType,omitempty" yaml:"adminOrderType,omitempty"`
	AuthorisationLevel string `json:"authorisationLevel,omitempty" yaml:"authorisationLevel,omitempty"`
	AccountID          string `json:"accountID,omitempty" yaml:"accountID,omitempty"`
	MaxAmountValue     string `json:"maxAmountValue,omitempty" yaml:"maxAmountValue,omitempty"`
	MaxAmountCurrency  string `json:"maxAmountCurrency,omitempty" yaml:"maxAmountCurrency,omitempty"`
	IsEnabled          bool   `json:"isEnabled" yaml:"isEnabled"`
}
