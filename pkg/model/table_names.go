package model

import "code.waarp.fr/apps/gateway/gateway/pkg/database"

// These are the constants defining the names of the database tables associated
// with the models.
const (
	TableCredentials                  = "credentials"
	TableHistory                      = "transfer_history"
	TableLocAccounts                  = "local_accounts"
	TableLocAgents                    = "local_agents"
	TableRemAccounts                  = "remote_accounts"
	TableRemAgents                    = "remote_agents"
	TableClients                      = "clients"
	TableRules                        = "rules"
	TableRuleAccesses                 = "rule_access"
	TableTasks                        = "tasks"
	TableTransfers                    = "transfers"
	TableTransferInfo                 = "transfer_info"
	TableFileInfo                     = "file_info"
	TableUsers                        = "users"
	TableCloudInstances               = "cloud_instances"
	TableAuthorities                  = "auth_authorities"
	TableAuthHosts                    = "authority_hosts"
	TableCryptoKeys                   = "crypto_keys"
	TableEmailTemplates               = "email_templates"
	TableSMTPCredentials              = "smtp_credentials"
	TableEbicsHosts                   = "ebics_hosts"
	TableEbicsSubscribers             = "ebics_subscribers"
	TableEbicsBankKeys                = "ebics_bank_keys"
	TableEbicsContractViews           = "ebics_contract_views"
	TableEbicsContractViewItems       = "ebics_contract_view_items"
	TableEbicsPayloadProfiles         = "ebics_payload_profiles"
	TableEbicsOperations              = "ebics_operations"
	TableEbicsTransactions            = "ebics_transactions"
	TableEbicsTransactionSegments     = "ebics_transaction_segments"
	TableEbicsKeyLifecycles           = "ebics_key_lifecycles"
	TableEbicsInitializationWorkflows = "ebics_initialization_workflows"
	TableEbicsRTNEvents               = "ebics_rtn_events"
	TableEbicsRTNProviders            = "ebics_rtn_providers"

	ViewNormalizedTransfers = "normalized_transfers"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddInit(&User{})
}
