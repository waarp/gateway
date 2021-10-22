//go:build test_full || test_db_postgresql
// +build test_full test_db_postgresql

package migrations

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
)

func TestPostgreSQLMigrations(t *testing.T) {
	const dbType = migration.PostgreSQL

	Convey("Given an un-migrated PostgreSQL database engine", t, func(c C) {
		eng := getPostgreEngine(c)

		testVer0_4_2RemoveHistoryRemoteIdUnique(eng, dbType)
	})
}
