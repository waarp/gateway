// +build test_full test_db_postgresql

package migrations

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPostgreSQLMigrations(t *testing.T) {
	const dbType = migration.PostgreSQL

	Convey("Given an un-migrated PostgreSQL database engine", t, func(c C) {
		eng := getPostgreEngine(c)

		testVer0_5_0RemoveRulePathSlash(eng, dbType)
		testVer0_5_0CheckRulePathAncestor(eng, dbType)
	})
}
