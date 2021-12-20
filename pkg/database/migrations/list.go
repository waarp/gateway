// Package migrations contains a list of all the database migrations written
// for the gateway.
package migrations

import (
	"code.waarp.fr/lib/migration"
)

type script interface {
	Up(db migration.Actions) error
	Down(db migration.Actions) error
}

type change struct {
	Description string
	Script      script
}

// Migrations should be declared here in chronological order. This means that
// new migrations should ALWAYS be added at the end of the list so that the order
// never changes.
//
//nolint:gochecknoglobals // global var is used by design
var Migrations = []change{
	{ // #0
		Description: "Initialize the database",
		Script:      ver0_4_0InitDatabase{},
	},
	{ // #1
		Description: "Remove the UNIQUE constraint on the history table's remote ID",
		Script:      ver0_4_2RemoveHistoryRemoteIDUnique{},
	},
	{ // #2
		Description: "Remove the leading / from all rule paths",
		Script:      ver0_5_0RemoveRulePathSlash{},
	},
	{ // #3
		Description: "Check that no rule path is the parent of another",
		Script:      ver0_5_0CheckRulePathParent{},
	},
	{ // #4
		Description: "Change the new local agent paths to OS specific paths",
		Script:      ver0_5_0LocalAgentDenormalizePaths{},
	},
	{ // #5
		Description: "Replace the local agent path columns with new send/receive ones",
		Script:      ver0_5_0LocalAgentsPathsRename{},
	},
	{ // #6
		Description: "Disallow reserved names for local servers",
		Script:      ver0_5_0LocalAgentsDisallowReservedNames{},
	},
	{ // #7
		Description: "Add new path columns to the rule table",
		Script:      ver0_5_0RulesPathsRename{},
	},
	{ // #8
		Description: "Change the new rule paths to OS specific paths",
		Script:      ver0_5_0RulePathChanges{},
	},
	{ // #9
		Description: "Add a filesize to the transfers & history tables",
		Script:      ver0_5_0AddFilesize{},
	},
	{ // #10
		Description: "Replace the existing transfer path columns with new ones",
		Script:      ver0_5_0TransferChangePaths{},
	},
	{ // #11
		Description: "Change the transfer's local path to the OS specific format",
		Script:      ver0_5_0TransferFormatLocalPath{},
	},
	{ // #12
		Description: "Replace the existing history filename columns with new local/remote ones",
		Script:      ver0_5_0HistoryPathsChange{},
	},
	{ // #13
		Description: "Decode the (double) base64 encoded local agent password hashes",
		Script:      ver0_5_0LocalAccountsPasswordDecode{},
	},
	{ // #14
		Description: "Rename and change the type of the user 'password' column",
		Script:      ver0_5_0UserPasswordChange{},
	},
	{ // #15
		Description: "Fill the remote_transfer_id column where it is empty",
		Script:      ver0_5_2FillRemoteTransferID{},
	},
	{ // #16
		Description: "Add a 'is_history' column to the transfer info table",
		Script:      ver0_6_0AddTransferInfoIsHistory{},
	},
	{ // #17
		Description: `Add an "enabled" column to the local agents table`,
		Script:      ver0_7_0AddLocalAgentEnabledColumn{},
	},
	{ // #18
		Description: "Revamp the 'users' table",
		Script:      ver0_7_0RevampUsersTable{},
	},
	{ // #19
		Description: "Revamp the 'local_agents' table",
		Script:      ver0_7_0RevampLocalAgentsTable{},
	},
	{ // #20
		Description: "Revamp the 'remote_agents' table",
		Script:      ver0_7_0RevampRemoteAgentsTable{},
	},
	{ // #21
		Description: "Revamp the 'local_accounts' table",
		Script:      ver0_7_0RevampLocalAccountsTable{},
	},
	{ // #22
		Description: "Revamp the 'remote_accounts' table",
		Script:      ver0_7_0RevampRemoteAccountsTable{},
	},
	{ // #23
		Description: "Revamp the 'rules' table",
		Script:      ver0_7_0RevampRulesTable{},
	},
	{ // #24
		Description: "Revamp the 'tasks' table",
		Script:      ver0_7_0RevampTasksTable{},
	},
	{ // #25
		Description: "Revamp the 'transfer_history' table",
		Script:      ver0_7_0RevampHistoryTable{},
	},
	{ // #26
		Description: "Revamp the 'transfers' table",
		Script:      ver0_7_0RevampTransfersTable{},
	},
	{ // #27
		Description: "Revamp the 'transfer_info' table",
		Script:      ver0_7_0RevampTransferInfoTable{},
	},
	{ // #28
		Description: "Revamp the 'crypto' table",
		Script:      ver0_7_0RevampCryptoTable{},
	},
	{ // #29
		Description: "Revamp the 'rule_access' table",
		Script:      ver0_7_0RevampRuleAccessTable{},
	},
	{ // #30
		Description: "Add a unique constraint to the local agent 'address' column",
		Script:      ver0_7_0AddLocalAgentsAddressUnique{},
	},
	{ // 31
		Description: "Add a normalized transfer view which combines transfers & history",
		Script:      ver0_7_0AddNormalizedTransfersView{},
	},
}
