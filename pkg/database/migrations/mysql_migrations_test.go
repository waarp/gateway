//go:build test_db_mysql

package migrations

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/lib/migration"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func getMySQLEngine(c C) *testEngine {
	logger := testhelpers.TestLogger(c, "test_mysql_engine")
	db := testhelpers.GetTestMySQLDB(c)

	script := strings.Split(MysqlCreationScript, ";\n")
	for _, cmd := range script {
		_, err := db.Exec(cmd)
		So(err, ShouldBeNil)
	}

	eng, err := migration.NewEngine(db, migration.MySQL, logger, nil)
	So(err, ShouldBeNil)

	return &testEngine{Engine: eng, DB: db}
}

func TestMySQLCreationScript(t *testing.T) {
	Convey("Given a MySQL database", t, func(c C) {
		db := testhelpers.GetTestMySQLDB(c)

		Convey("Given the change to initialize version 0.0.0 of the database", func() {
			Convey("When executing the change", func() {
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
	Convey("Given an un-migrated MySQL database engine", t, func(c C) {
		testMigrations(getMySQLEngine(c), migration.MySQL)
	})
}
