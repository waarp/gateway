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
	{
		Description: "Starting version",
		Script:      bumpVersion{from: "", to: "0.4.0"}, // should never be called
		VersionTag:  "0.4.0",
	},
	{
		Description: "Bump the database version to 0.4.1",
		Script:      bumpVersion{from: "0.4.0", to: "0.4.1"},
		VersionTag:  "0.4.1",
	},
	{
		Description: "Remove the UNIQUE constraint on the history table's remote ID",
		Script:      ver0_4_2RemoveHistoryRemoteIDUnique{},
	},
	{
		Description: "Bump the database version to 0.4.2",
		Script:      bumpVersion{from: "0.4.1", to: "0.4.2"},
		VersionTag:  "0.4.2",
	},
	{
		Description: "Bump the database version to 0.4.3",
		Script:      bumpVersion{from: "0.4.2", to: "0.4.3"},
		VersionTag:  "0.4.3",
	},
	{
		Description: "Change the new local agent paths to OS specific paths",
		Script:      ver0_5_0LocalAgentChangePaths{},
	},
	{
		Description: "Disallow reserved names for local servers",
		Script:      ver0_5_0LocalAgentDisallowReservedNames{},
	},
	{
		Description: "Add new path columns to the rule table",
		Script:      ver0_5_0RuleNewPathCols{},
	},
	{
		Description: "Change the new rule paths to OS specific paths",
		Script:      ver0_5_0RulePathChanges{},
	},
	{
		Description: "Add a filesize to the transfers & history tables",
		Script:      ver0_5_0AddFilesize{},
	},
	{
		Description: "Replace the existing transfer path columns with new ones",
		Script:      ver0_5_0TransferChangePaths{},
	},
	{
		Description: "Change the transfer's local path to the OS specific format",
		Script:      ver0_5_0TransferFormatLocalPath{},
	},
	{
		Description: "Replace the existing history filename columns with new ones",
		Script:      ver0_5_0HistoryPathsChange{},
	},
	{
		Description: "Decode the (double) base64 encoded local agent password hashes",
		Script:      ver0_5_0LocalAccountsPasswordDecode{},
	},
	{
		Description: "Rename and change the type of the user 'password' column",
		Script:      ver0_5_0UserPasswordChange{},
	},
}
