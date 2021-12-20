package model

// Cryptos is the type representing a slice of Crypto.
type Cryptos []*Crypto

func (*Cryptos) TableName() string { return TableCrypto }
func (*Cryptos) Elem() string      { return "crypto credential" }

// TransferInfoList is the type representing a slice of TransferInfo.
type TransferInfoList []*TransferInfo

func (*TransferInfoList) TableName() string { return TableTransferInfo }
func (*TransferInfoList) Elem() string      { return "transfer info" }

// HistoryEntries is the type representing a slice of HistoryEntry.
type HistoryEntries []*HistoryEntry

func (*HistoryEntries) TableName() string { return TableHistory }
func (*HistoryEntries) Elem() string      { return "history entry" }

// LocalAccounts is the type representing a slice of LocalAccount.
type LocalAccounts []*LocalAccount

func (*LocalAccounts) TableName() string { return TableLocAccounts }
func (*LocalAccounts) Elem() string      { return "local account" }

// LocalAgents is the type representing a slice of LocalAgents.
type LocalAgents []*LocalAgent

func (*LocalAgents) TableName() string { return TableLocAgents }
func (*LocalAgents) Elem() string      { return "server" }

// RemoteAccounts is the type representing a slice of RemoteAccounts.
type RemoteAccounts []*RemoteAccount

func (*RemoteAccounts) TableName() string { return TableRemAccounts }
func (*RemoteAccounts) Elem() string      { return "remote account" }

// RemoteAgents is the type representing a slice of RemoteAgent.
type RemoteAgents []*RemoteAgent

func (*RemoteAgents) TableName() string { return TableRemAgents }
func (*RemoteAgents) Elem() string      { return "partner" }

// Rules is the type representing a slice of Rule.
type Rules []*Rule

func (*Rules) TableName() string { return TableRules }
func (*Rules) Elem() string      { return "rule" }

// RuleAccesses is the type representing a slice of RuleAccess.
type RuleAccesses []*RuleAccess

func (*RuleAccesses) TableName() string { return TableRuleAccesses }
func (*RuleAccesses) Elem() string      { return "rule permission" }

// Tasks is the type representing a slice of Task.
type Tasks []*Task

func (*Tasks) TableName() string { return TableTasks }
func (*Tasks) Elem() string      { return "task" }

// Transfers is the type representing a slice of Transfer.
type Transfers []*Transfer

func (*Transfers) TableName() string { return TableTransfers }
func (*Transfers) Elem() string      { return "transfer" } //nolint:goconst // this is not the same constant

// Users is the type representing a slice of User.
type Users []*User

func (*Users) TableName() string { return TableUsers }
func (*Users) Elem() string      { return "user" }

/*
// FileInfoList is the type representing a slice of FileInfo.
type FileInfoList []FileInfo

func (*FileInfoList) TableName() string { return TableFileInfo }
func (*FileInfoList) Elem() string      { return "file info" }
*/

// NormalizedTransfers is the type representing a slice of NormalizedTransferView.
type NormalizedTransfers []*NormalizedTransferView

// TableName returns the name of the users table.
func (*NormalizedTransfers) TableName() string { return ViewNormalizedTransfers }

// Elem returns the name of 1 element of the users table.
func (*NormalizedTransfers) Elem() string { return "normalized transfer" }
