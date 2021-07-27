// Package migrations contains a list of all the database migrations written
// for the gateway.
package migrations

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
)

// Migrations should be declared here in chronological order. This means that
// new migrations should ALWAYS be added at the end of the list so that the order
// never changes.
var Migrations = []migration.Migration{
	{
		Description: "Bump the database version to 0.4.0",
		Script:      bumpVersion{from: "0.0.0", to: "0.4.0"},
		VersionTag:  "0.4.0",
	}, {
		Description: "Bump the database version to 0.4.1",
		Script:      bumpVersion{from: "0.4.0", to: "0.4.1"},
		VersionTag:  "0.4.1",
	},
}
