// Package migrations contains a list of all the database migrations written
// for the gateway.
package migrations

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
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
	}, {
		Description: "Bump the database version to 0.4.1",
		Script:      bumpVersion{from: "0.4.0", to: "0.4.1"},
		VersionTag:  "0.4.1",
	}, {
		Description: "Remove the UNIQUE constraint on the history table's remote ID",
		Script:      ver0_4_2RemoveHistoryRemoteIDUnique{},
	}, {
		Description: "Bump the database version to 0.4.2",
		Script:      bumpVersion{from: "0.4.1", to: "0.4.2"},
		VersionTag:  "0.4.2",
	}, {
		Description: "Bump the database version to 0.4.3",
		Script:      bumpVersion{from: "0.4.2", to: "0.4.3"},
		VersionTag:  "0.4.3",
	}, {
		Description: "Bump the database version to 0.4.4",
		Script:      bumpVersion{from: "0.4.3", to: "0.4.4"},
		VersionTag:  "0.4.4",
	},
}
