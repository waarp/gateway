package model

import "code.waarp.fr/apps/gateway/gateway/pkg/database"

// These are the constants defining the names of the database tables associated
// with the models.
const (
	TableCredentials    = "credentials"
	TableHistory        = "transfer_history"
	TableLocAccounts    = "local_accounts"
	TableLocAgents      = "local_agents"
	TableRemAccounts    = "remote_accounts"
	TableRemAgents      = "remote_agents"
	TableClients        = "clients"
	TableRules          = "rules"
	TableRuleAccesses   = "rule_access"
	TableTasks          = "tasks"
	TableTransfers      = "transfers"
	TableTransferInfo   = "transfer_info"
	TableFileInfo       = "file_info"
	TableUsers          = "users"
	TableCloudInstances = "cloud_instances"
	TableAuthorities    = "auth_authorities"
	TableAuthHosts      = "authority_hosts"
	TableCryptoKeys     = "crypto_keys"

	ViewNormalizedTransfers = "normalized_transfers"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddInit(&User{})
}
