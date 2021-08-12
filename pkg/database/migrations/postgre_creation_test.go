// +build test_full test_db_postgresql

package migrations

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
	"github.com/smartystreets/goconvey/convey"
	. "github.com/smartystreets/goconvey/convey"
)

func getPostgreEngine(c convey.C) *migration.Engine {
	db := testhelpers.GetTestPostgreDB(c)

	_, err := db.Exec(postgresCreationScript)
	So(err, ShouldBeNil)

	eng, err := migration.NewEngine(db, migration.PostgreSQL, nil)
	So(err, ShouldBeNil)

	return eng
}

func TestPostgreSQLCreationScript(t *testing.T) {
	Convey("Given a PostgreSQL database", t, func(c C) {
		db := testhelpers.GetTestPostgreDB(c)

		Convey("Given the script to initialize version 0.0.0 of the database", func() {

			Convey("When executing the script", func() {
				_, err := db.Exec(postgresCreationScript)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
