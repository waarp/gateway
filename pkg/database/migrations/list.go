// Package migrations contains a list of all the database migrations written
// for the gateway.
package migrations

import (
	"code.waarp.fr/lib/migration"
)

type Change = migration.Script

func noop(migration.Actions) error { return nil }

// Migrations should be declared here in chronological order. This means that
// new migrations should ALWAYS be added at the end of the list so that the order
// never changes.
//
//nolint:gochecknoglobals // global var is used by design
var Migrations = []Change{
	{ // #0
		Description: "Initialize the database",
		Up:          ver0_4_0InitDatabaseUp,
		Down:        ver0_4_0InitDatabaseDown,
	},
	{ // #1
		Description: "Remove the UNIQUE constraint on the history table's remote ID",
		Up:          ver0_4_2RemoveHistoryRemoteIDUniqueUp,
		Down:        ver0_4_2RemoveHistoryRemoteIDUniqueDown,
	},
	{ // #2
		Description: "Remove the leading / from all rule paths",
		Up:          ver0_5_0RemoveRulePathSlashUp,
		Down:        ver0_5_0RemoveRulePathSlashDown,
	},
	{ // #3
		Description: "Check that no rule path is the parent of another",
		Up:          ver0_5_0CheckRulePathParentUp,
		Down:        noop, // nothing to do
	},
	{ // #4
		Description: "Change the new local agent paths to OS specific paths",
		Up:          ver0_5_0LocalAgentDenormalizePathsUp,
		Down:        ver0_5_0LocalAgentDenormalizePathsDown,
	},
	{ // #5
		Description: "Replace the local agent path columns with new send/receive ones",
		Up:          ver0_5_0LocalAgentsPathsRenameUp,
		Down:        ver0_5_0LocalAgentsPathsRenameDown,
	},
	{ // #6
		Description: "Disallow reserved names for local servers",
		Up:          ver0_5_0LocalAgentsDisallowReservedNamesUp,
		Down:        noop, // nothing to do
	},
	{ // #7
		Description: "Add new path columns to the rule table",
		Up:          ver0_5_0RulesPathsRenameUp,
		Down:        ver0_5_0RulesPathsRenameDown,
	},
	{ // #8
		Description: "Change the new rule paths to OS specific paths",
		Up:          ver0_5_0RulePathChangesUp,
		Down:        ver0_5_0RulePathChangesDown,
	},
	{ // #9
		Description: "Add a filesize to the transfers & history tables",
		Up:          ver0_5_0AddFilesizeUp,
		Down:        ver0_5_0AddFilesizeDown,
	},
	{ // #10
		Description: "Replace the existing transfer path columns with new ones",
		Up:          ver0_5_0TransferChangePathsUp,
		Down:        ver0_5_0TransferChangePathsDown,
	},
	{ // #11
		Description: "Change the transfer's local path to the OS specific format",
		Up:          ver0_5_0TransferFormatLocalPathUp,
		Down:        ver0_5_0TransferFormatLocalPathDown,
	},
	{ // #12
		Description: "Replace the existing history filename columns with new local/remote ones",
		Up:          ver0_5_0HistoryPathsChangeUp,
		Down:        ver0_5_0HistoryPathsChangeDown,
	},
	{ // #13
		Description: "Decode the (double) base64 encoded local agent password hashes",
		Up:          ver0_5_0LocalAccountsPasswordDecodeUp,
		Down:        ver0_5_0LocalAccountsPasswordDecodeDown,
	},
	{ // #14
		Description: "Rename and change the type of the user 'password' column",
		Up:          ver0_5_0UserPasswordChangeUp,
		Down:        ver0_5_0UserPasswordChangeDown,
	},
	{ // #15
		Description: "Fill the remote_transfer_id column where it is empty",
		Up:          ver0_5_2FillRemoteTransferIDUp,
		Down:        ver0_5_2FillRemoteTransferIDDown,
	},
	{ // #16
		Description: "Add a 'is_history' column to the transfer info table",
		Up:          ver0_6_0AddTransferInfoIsHistoryUp,
		Down:        ver0_6_0AddTransferInfoIsHistoryDown,
	},
	{ // #17
		Description: `Add an "enabled" column to the local agents table`,
		Up:          ver0_7_0AddLocalAgentEnabledColumnUp,
		Down:        ver0_7_0AddLocalAgentEnabledColumnDown,
	},
	{ // #18
		Description: "Revamp the 'users' table",
		Up:          ver0_7_0RevampUsersTableUp,
		Down:        ver0_7_0RevampUsersTableDown,
	},
	{ // #19
		Description: "Revamp the 'local_agents' table",
		Up:          ver0_7_0RevampLocalAgentsTableUp,
		Down:        ver0_7_0RevampLocalAgentsTableDown,
	},
	{ // #20
		Description: "Revamp the 'remote_agents' table",
		Up:          ver0_7_0RevampRemoteAgentsTableUp,
		Down:        ver0_7_0RevampRemoteAgentsTableDown,
	},
	{ // #21
		Description: "Revamp the 'local_accounts' table",
		Up:          ver0_7_0RevampLocalAccountsTableUp,
		Down:        ver0_7_0RevampLocalAccountsTableDown,
	},
	{ // #22
		Description: "Revamp the 'remote_accounts' table",
		Up:          ver0_7_0RevampRemoteAccountsTableUp,
		Down:        ver0_7_0RevampRemoteAccountsTableDown,
	},
	{ // #23
		Description: "Revamp the 'rules' table",
		Up:          ver0_7_0RevampRulesTableUp,
		Down:        ver0_7_0RevampRulesTableDown,
	},
	{ // #24
		Description: "Revamp the 'tasks' table",
		Up:          ver0_7_0RevampTasksTableUp,
		Down:        ver0_7_0RevampTasksTableDown,
	},
	{ // #25
		Description: "Revamp the 'transfer_history' table",
		Up:          ver0_7_0RevampHistoryTableUp,
		Down:        ver0_7_0RevampHistoryTableDown,
	},
	{ // #26
		Description: "Revamp the 'transfers' table",
		Up:          ver0_7_0RevampTransfersTableUp,
		Down:        ver0_7_0RevampTransfersTableDown,
	},
	{ // #27
		Description: "Revamp the 'transfer_info' table",
		Up:          ver0_7_0RevampTransferInfoTableUp,
		Down:        ver0_7_0RevampTransferInfoTableDown,
	},
	{ // #28
		Description: "Revamp the 'crypto' table",
		Up:          ver0_7_0RevampCryptoTableUp,
		Down:        ver0_7_0RevampCryptoTableDown,
	},
	{ // #29
		Description: "Revamp the 'rule_access' table",
		Up:          ver0_7_0RevampRuleAccessTableUp,
		Down:        ver0_7_0RevampRuleAccessTableDown,
	},
	{ // #30
		Description: "Add a unique constraint to the local agent 'address' column",
		Up:          ver0_7_0AddLocalAgentsAddressUniqueUp,
		Down:        ver0_7_0AddLocalAgentsAddressUniqueDown,
	},
	{ // #31
		Description: "Add a normalized transfer view which combines transfers & history",
		Up:          ver0_7_0AddNormalizedTransfersViewUp,
		Down:        ver0_7_0AddNormalizedTransfersViewDown,
	},
	{ // #32
		Description: "Split the R66 protocol into R66 (plain) & R66-TLS",
		Up:          ver0_7_5SplitR66TLSUp,
		Down:        ver0_7_5SplitR66TLSDown,
	},
	// ######################### 0.7.1 #########################
	// ######################### 0.7.2 #########################
	// ######################### 0.7.3 #########################
	// ######################### 0.7.4 #########################
	// ######################### 0.7.5 #########################
	{ // #33
		Description: "Drop the normalized transfer view",
		Up:          ver0_8_0DropNormalizedTransfersViewUp,
		Down:        ver0_8_0DropNormalizedTransfersViewDown,
	},
	{ // #34
		Description: `Add a "filename" column to the transfers table`,
		Up:          ver0_8_0AddTransferFilenameUp,
		Down:        ver0_8_0AddTransferFilenameDown,
	},
	{ // #35
		Description: `Add a "filename" column to the history table`,
		Up:          ver0_8_0AddHistoryFilenameUp,
		Down:        ver0_8_0AddHistoryFilenameDown,
	},
	{ // #36
		Description: "Restore and update the normalized transfer view with the new filename",
		Up:          ver0_8_0UpdateNormalizedTransfersViewUp,
		Down:        ver0_8_0UpdateNormalizedTransfersViewDown,
	},
	{ // #37
		Description: `Add a "cloud_instances" table`,
		Up:          ver0_9_0AddCloudInstancesUp,
		Down:        ver0_9_0AddCloudInstancesDown,
	},
	{ // #38
		Description: `Converts all the transfers' local paths to URLs`,
		Up:          ver0_9_0LocalPathToURLUp,
		Down:        ver0_9_0LocalPathToURLDown,
	},
	{ // #39
		Description: `Replaces the local agent "enabled" column by a "disabled" one`,
		Up:          ver0_9_0FixLocalServerEnabledUp,
		Down:        ver0_9_0FixLocalServerEnabledDown,
	},
	{ // #40
		Description: `Add a "clients" table and fill it with default clients`,
		Up:          ver0_9_0AddClientsTableUp,
		Down:        ver0_9_0AddClientsTableDown,
	},
	{ // #41
		Description: `Add an "owner" column to the "remote_agents" table`,
		Up:          ver0_9_0AddRemoteAgentOwnerUp,
		Down:        ver0_9_0AddRemoteAgentOwnerDown,
	},
	{ // #42
		Description: "Duplicate all the partners and their children",
		Up:          ver0_9_0DuplicateRemoteAgentsUp,
		Down:        ver0_9_0DuplicateRemoteAgentsDown,
	},
	{ // #43
		Description: "Relink the transfer agent IDs to the correct instances",
		Up:          ver0_9_0RelinkTransfersUp,
		Down:        ver0_9_0RelinkTransfersDown,
	},
	{ // #44
		Description: `Add a "client_id" column to the "transfers" table`,
		Up:          ver0_9_0AddTransferClientIDUp,
		Down:        ver0_9_0AddTransferClientIDDown,
	},
	{ // #45
		Description: "Add the 'client' column to the history",
		Up:          ver0_9_0AddHistoryClientUp,
		Down:        ver0_9_0AddHistoryClientDown,
	},
	{ // #46
		Description: `Restore and modify the "normalized_transfers" view`,
		Up:          ver0_9_0RestoreNormalizedTransfersViewUp,
		Down:        ver0_9_0RestoreNormalizedTransfersViewDown,
	},
	{ // #47
		Description: "Add the new 'credentials' table",
		Up:          ver0_9_0AddCredTableUp,
		Down:        ver0_9_0AddCredTableDown,
	},
	{ // #48
		Description: "Fill the new 'credentials' table",
		Up:          ver0_9_0FillCredTableUp,
		Down:        ver0_9_0FillCredTableDown,
	},
	{ // #49
		Description: "Remove the old 'crypto_credentials' table & the account password columns",
		Up:          ver0_9_0RemoveOldAuthsUp,
		Down:        ver0_9_0RemoveOldAuthsDown,
	},
	{ // #50
		Description: "Extracts the R66 server credentials from the proto config to the credentials table",
		Up:          ver0_9_0MoveR66ServerPswdUp,
		Down:        ver0_9_0MoveR66ServerPswdDown,
	},
	{ // #51
		Description: `Add the "auth_authorities" & "authority_hosts" tables`,
		Up:          ver0_9_0AddAuthoritiesTableUp,
		Down:        ver0_9_0AddAuthoritiesTableDown,
	},
	{ // #52
		Description: `Add the "snmp_targets" table`,
		Up:          ver0_10_0AddSNMPMonitorsUp,
		Down:        ver0_10_0AddSNMPMonitorsDown,
	},
	{ // #53
		Description: `Add an "ip_addresses" column to the "local_accounts" table`,
		Up:          ver0_10_0AddLocalAccountIPAddrUp,
		Down:        ver0_10_0AddLocalAccountIPAddrDown,
	},
	{ // #54
		Description: `Add indexes on the transfer & history "start" columns`,
		Up:          ver0_10_0AddTransferStartIndexUp,
		Down:        ver0_10_0AddTransferStartIndexDown,
	},
	{ // #55
		Description: `Add the "snmp_server_conf" table`,
		Up:          ver0_11_0AddSNMPServerConfigUp,
		Down:        ver0_11_0AddSNMPServerConfigDown,
	},
	{ // #56
		Description: `Add the "crypto_keys" table`,
		Up:          ver0_12_0AddCryptoKeysUp,
		Down:        ver0_12_0AddCryptoKeysDown,
	},
	{ // #57
		Description: `Remove the unique constraint on the remote transfer ID`,
		Up:          ver0_12_0DropRemoteTransferIdUniqueUp,
		Down:        ver0_12_0DropRemoteTransferIdUniqueDown,
	},
	{ // #58
		Description: `Add an "owner" column to the "crypto_keys" table`,
		Up:          ver0_12_1AddCryptoKeysOwnerUp,
		Down:        ver0_12_1AddCryptoKeysOwnerDown,
	},
}
