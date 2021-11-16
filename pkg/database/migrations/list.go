// Package migrations contains a list of all the database migrations written
// for the gateway.
package migrations

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

// Migrations should be declared here in chronological order. This means that
// new migrations should ALWAYS be added at the end of the list so that the order
// never changes.
//nolint:gochecknoglobals // global var is used by design
var Migrations = []migration.Migration{
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
		Description: "Change the new local agent paths to OS specific paths",
		Script:      ver0_5_0LocalAgentDenormalizePaths{},
	},
	{ // #7
		Description: "Replace the local agent path columns with new send/receive ones",
		Script:      ver0_5_0LocalAgentsPathsRename{},
	},
	{ // #8
		Description: "Disallow reserved names for local servers",
		Script:      ver0_5_0LocalAgentsDisallowReservedNames{},
	},
	{ // #9
		Description: "Add new path columns to the rule table",
		Script:      ver0_5_0RulesPathsRename{},
	},
	{ // #10
		Description: "Change the new rule paths to OS specific paths",
		Script:      ver0_5_0RulePathChanges{},
	},
	{ // #11
		Description: "Add a filesize to the transfers & history tables",
		Script:      ver0_5_0AddFilesize{},
	},
	{ // #12
		Description: "Replace the existing transfer path columns with new ones",
		Script:      ver0_5_0TransferChangePaths{},
	},
	{ // #13
		Description: "Change the transfer's local path to the OS specific format",
		Script:      ver0_5_0TransferFormatLocalPath{},
	},
	{ // #14
		Description: "Replace the existing history filename columns with new local/remote ones",
		Script:      ver0_5_0HistoryPathsChange{},
	},
	{ // #15
		Description: "Decode the (double) base64 encoded local agent password hashes",
		Script:      ver0_5_0LocalAccountsPasswordDecode{},
	},
	{ // #16
		Description: "Rename and change the type of the user 'password' column",
		Script:      ver0_5_0UserPasswordChange{},
	},
}
