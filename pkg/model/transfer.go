package model

// Transfer represents one record of the 'transfers' table.
type Transfer struct {
	RuleID      uint64 `json:"ruleID"`
	RemoteID    uint64 `json:"remoteID"`
	AccountID   uint64 `json:"accountID"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}
