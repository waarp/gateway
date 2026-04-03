package api

type OutEbicsRuntimePolicy struct {
	Enabled                     bool  `json:"enabled" yaml:"enabled"`
	MaintenanceIntervalSeconds  int64 `json:"maintenanceIntervalSeconds" yaml:"maintenanceIntervalSeconds"`
	TransactionRetentionSeconds int64 `json:"transactionRetentionSeconds" yaml:"transactionRetentionSeconds"`
	RTNEventRetentionSeconds    int64 `json:"rtnEventRetentionSeconds" yaml:"rtnEventRetentionSeconds"`
}

type PatchEbicsRuntimePolicyReqObject struct {
	Enabled                     Nullable[bool]  `json:"enabled" yaml:"enabled"`
	MaintenanceIntervalSeconds  Nullable[int64] `json:"maintenanceIntervalSeconds" yaml:"maintenanceIntervalSeconds"`
	TransactionRetentionSeconds Nullable[int64] `json:"transactionRetentionSeconds" yaml:"transactionRetentionSeconds"`
	RTNEventRetentionSeconds    Nullable[int64] `json:"rtnEventRetentionSeconds" yaml:"rtnEventRetentionSeconds"`
}
