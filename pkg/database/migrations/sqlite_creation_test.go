package migrations

import (
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"github.com/smartystreets/goconvey/convey"
	. "github.com/smartystreets/goconvey/convey"
)

func tempFilename() string {
	f, err := ioutil.TempFile(os.TempDir(), "test_migration_database_*.db")
	So(err, ShouldBeNil)
	So(f.Close(), ShouldBeNil)
	So(os.Remove(f.Name()), ShouldBeNil)
	return f.Name()
}

func getSQLiteEngine(c convey.C) *migration.Engine {
	db := testhelpers.GetTestSqliteDB(c)

	_, err := db.Exec(sqliteCreationScript)
	So(err, ShouldBeNil)

	eng, err := migration.NewEngine(db, migration.SQLite, nil)
	So(err, ShouldBeNil)

	return eng
}

func TestSqliteCreationScript(t *testing.T) {
	Convey("Given a SQLite database", t, func(c C) {
		db := testhelpers.GetTestSqliteDB(c)

		Convey("Given the script to initialize version 0.0.0 of the database", func() {

			Convey("When executing the script", func() {
				_, err := db.Exec(sqliteCreationScript)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
