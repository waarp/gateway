// +build test_full test_db_postgresql

package migrations

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

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
