//go:build test_db_mysql
// +build test_db_mysql

package migrations

import (
	"strings"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"

	. "github.com/smartystreets/goconvey/convey"
)

func getMySQLEngine(c C) *migration.Engine {
	db := testhelpers.GetTestMySQLDB(c)

	script := strings.Split(MysqlCreationScript, ";\n")
	for _, cmd := range script[:len(script)-1] {
		_, err := db.Exec(cmd)
		So(err, ShouldBeNil)
	}

	eng, err := migration.NewEngine(db, migration.MySQL, nil)
	So(err, ShouldBeNil)

	return eng
}

func TestMySQLCreationScript(t *testing.T) {
	Convey("Given a MySQL database", t, func(c C) {
		db := testhelpers.GetTestMySQLDB(c)

		Convey("Given the script to initialize version 0.0.0 of the database", func() {
			Convey("When executing the script", func() {
				script := strings.Split(MysqlCreationScript, ";\n")
				for _, cmd := range script[:len(script)-1] {
					_, err := db.Exec(cmd)
					So(err, ShouldBeNil)
				}
			})
		})
	})
}

func TestMySQLMigrations(t *testing.T) {
	const dbType = migration.MySQL

	Convey("Given an un-migrated MySQL database engine", t, func(c C) {
		eng := getSQLiteEngine(c)

		testVer0_4_2RemoveHistoryRemoteIdUnique(eng, dbType)
	})
}
