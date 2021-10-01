package model

// Cryptos is the type representing a slice of Crypto.
type Cryptos []Crypto

// TableName returns the name of the certificates table.
func (*Cryptos) TableName() string { return TableCrypto }

// Elem returns the name of 1 element of the certificates table.
func (*Cryptos) Elem() string { return "certificate" }

// TransferInfoList is the type representing a slice of ExtInfo.
type TransferInfoList []TransferInfo

// TableName returns the name of the transfer info table.
func (*TransferInfoList) TableName() string { return TableTransferInfo }

// Elem returns the name of 1 element of the transfer info table.
func (*TransferInfoList) Elem() string { return "transfer info" }

// HistoryEntries is the type representing a slice of HistoryEntry.
type HistoryEntries []HistoryEntry

// TableName returns the name of the transfer history table.
func (*HistoryEntries) TableName() string { return TableHistory }

// Elem returns the name of 1 element of the transfer history table.
func (*HistoryEntries) Elem() string { return "history entry" }

// LocalAccounts is the type representing a slice of LocalAccount.
type LocalAccounts []LocalAccount

// TableName returns the name of the local accounts table.
func (*LocalAccounts) TableName() string { return TableLocAccounts }

// Elem returns the name of 1 element of the local accounts table.
func (*LocalAccounts) Elem() string { return "local account" }

// LocalAgents is the type representing a slice of LocalAgents.
type LocalAgents []LocalAgent

// TableName returns the name of the local agents table.
func (*LocalAgents) TableName() string { return TableLocAgents }

// Elem returns the name of 1 element of the local agents table.
func (*LocalAgents) Elem() string { return "server" }

// RemoteAccounts is the type representing a slice of RemoteAccounts.
type RemoteAccounts []RemoteAccount

// TableName returns the name of the remote accounts table.
func (*RemoteAccounts) TableName() string { return TableRemAccounts }

// Elem returns the name of 1 element of the remote accounts table.
func (*RemoteAccounts) Elem() string { return "remote account" }

// RemoteAgents is the type representing a slice of RemoteAgent.
type RemoteAgents []RemoteAgent

// TableName returns the name of the remote agents table.
func (*RemoteAgents) TableName() string { return TableRemAgents }

// Elem returns the name of 1 element of the remote agents table.
func (*RemoteAgents) Elem() string { return "partner" }

// Rules is the type representing a slice of Rule.
type Rules []Rule

// TableName returns the name of the rules table.
func (*Rules) TableName() string { return TableRules }

// Elem returns the name of 1 element of the rules table.
func (*Rules) Elem() string { return "rule" }

// RuleAccesses is the type representing a slice of RuleAccess.
type RuleAccesses []RuleAccess

// TableName returns the name of the rule access table.
func (*RuleAccesses) TableName() string { return TableRuleAccesses }

// Elem returns the name of 1 element of the rule access table.
func (*RuleAccesses) Elem() string { return "rule permission" }

// Tasks is the type representing a slice of Task.
type Tasks []Task

// TableName returns the name of the task table.
func (*Tasks) TableName() string { return TableTasks }

// Elem returns the name of 1 element of the task table.
func (*Tasks) Elem() string { return "task" }

// Transfers is the type representing a slice of Transfer.
type Transfers []Transfer

// TableName returns the name of the transfers table.
func (*Transfers) TableName() string { return TableTransfers }

// Elem returns the name of 1 element of the transfers table.
//nolint:goconst // this is the same as a constant
func (*Transfers) Elem() string { return "transfer" }

// Users is the type representing a slice of User.
type Users []User

// TableName returns the name of the users table.
func (*Users) TableName() string { return TableUsers }

// Elem returns the name of 1 element of the users table.
func (*Users) Elem() string { return "user" }
