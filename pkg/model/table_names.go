package model

import "code.waarp.fr/apps/gateway/gateway/pkg/database"

// These are the constants defining the names of the database tables associated
// with the models.
const (
	TableCrypto       = "crypto_credentials"
	TableHistory      = "transfer_history"
	TableLocAccounts  = "local_accounts"
	TableLocAgents    = "local_agents"
	TableRemAccounts  = "remote_accounts"
	TableRemAgents    = "remote_agents"
	TableRules        = "rules"
	TableRuleAccesses = "rule_access"
	TableTasks        = "tasks"
	TableTransfers    = "transfers"
	TableTransferInfo = "transfer_info"
	TableFileInfo     = "file_info"
	TableUsers        = "users"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&User{})
	database.AddTable(&LocalAgent{})
	database.AddTable(&LocalAccount{})
	database.AddTable(&RemoteAgent{})
	database.AddTable(&RemoteAccount{})
	database.AddTable(&Crypto{})
	database.AddTable(&Rule{})
	database.AddTable(&Task{})
	database.AddTable(&RuleAccess{})
	database.AddTable(&Transfer{})
	database.AddTable(&HistoryEntry{})
	// database.AddTable(&FileInfo{})
	database.AddTable(&TransferInfo{})
}
