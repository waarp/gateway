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
		Description: "Adds the version table to the database",
		Script:      initVersion(),
	}, {
		Description: "Bump the database version to 0.5.0",
		Script:      bumpVersion("0.0.0", "0.5.0"),
		Version:     "0.5.0",
	},
}
