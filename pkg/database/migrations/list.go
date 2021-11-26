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
	VersionTag  string
}

// Migrations should be declared here in chronological order. This means that
// new migrations should ALWAYS be added at the end of the list so that the order
// never changes.
//nolint:gochecknoglobals // global var is used by design
var Migrations = []change{
	{ // #0
		Description: "Starting version",
		Script:      bumpVersion{from: "", to: "0.4.0"}, // should never be called
		VersionTag:  "0.4.0",
	},
	{ // #1
		Description: "Bump the database version to 0.4.1",
		Script:      bumpVersion{from: "0.4.0", to: "0.4.1"},
		VersionTag:  "0.4.1",
	},
	{ // #2
		Description: "Remove the UNIQUE constraint on the history table's remote ID",
		Script:      ver0_4_2RemoveHistoryRemoteIDUnique{},
	},
	{ // #3
		Description: "Bump the database version to 0.4.2",
		Script:      bumpVersion{from: "0.4.1", to: "0.4.2"},
		VersionTag:  "0.4.2",
	},
	{ // #4
		Description: "Bump the database version to 0.4.3",
		Script:      bumpVersion{from: "0.4.2", to: "0.4.3"},
		VersionTag:  "0.4.3",
	},
	{ // #5
		Description: "Bump the database version to 0.4.4",
		Script:      bumpVersion{from: "0.4.3", to: "0.4.4"},
		VersionTag:  "0.4.4",
	},
	{ // #6
		Description: "Remove the leading / from all rule paths",
		Script:      ver0_5_0RemoveRulePathSlash{},
	},
	{ // #7
		Description: "Check that no rule path is the parent of another",
		Script:      ver0_5_0CheckRulePathParent{},
	},
	{ // #8
		Description: "Change the new local agent paths to OS specific paths",
		Script:      ver0_5_0LocalAgentDenormalizePaths{},
	},
	{ // #9
		Description: "Replace the local agent path columns with new send/receive ones",
		Script:      ver0_5_0LocalAgentsPathsRename{},
	},
	{ // #10
		Description: "Disallow reserved names for local servers",
		Script:      ver0_5_0LocalAgentsDisallowReservedNames{},
	},
	{ // #11
		Description: "Add new path columns to the rule table",
		Script:      ver0_5_0RulesPathsRename{},
	},
	{ // #12
		Description: "Change the new rule paths to OS specific paths",
		Script:      ver0_5_0RulePathChanges{},
	},
	{ // #13
		Description: "Add a filesize to the transfers & history tables",
		Script:      ver0_5_0AddFilesize{},
	},
	{ // #14
		Description: "Replace the existing transfer path columns with new ones",
		Script:      ver0_5_0TransferChangePaths{},
	},
	{ // #15
		Description: "Change the transfer's local path to the OS specific format",
		Script:      ver0_5_0TransferFormatLocalPath{},
	},
	{ // #16
		Description: "Replace the existing history filename columns with new local/remote ones",
		Script:      ver0_5_0HistoryPathsChange{},
	},
	{ // #17
		Description: "Decode the (double) base64 encoded local agent password hashes",
		Script:      ver0_5_0LocalAccountsPasswordDecode{},
	},
	{ // #18
		Description: "Rename and change the type of the user 'password' column",
		Script:      ver0_5_0UserPasswordChange{},
	},
	{ // #19
		Description: "Bump the database version to 0.5.0",
		Script:      bumpVersion{from: "0.4.4", to: "0.5.0"},
		VersionTag:  "0.5.0",
	},
	{ // #20
		Description: "Bump the database version to 0.5.1",
		Script:      bumpVersion{from: "0.5.0", to: "0.5.1"},
		VersionTag:  "0.5.1",
	},
	{ // #21
		Description: "Fill the remote_transfer_id column where it is empty",
		Script:      ver0_5_2FillRemoteTransferID{},
	},
	{ // #22
		Description: "Bump the database version to 0.5.2",
		Script:      bumpVersion{from: "0.5.1", to: "0.5.2"},
	},
	{ // #23
		Description: "Add a 'is_history' column to the transfer info table",
		Script:      ver0_6_0AddTransferInfoIsHistory{},
	},
}
