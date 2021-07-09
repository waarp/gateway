// +build test_full test_db_mysql

package migrations

import (
	"strings"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMySQLCreationScript(t *testing.T) {
	Convey("Given a MySQL database", t, func(c C) {
		db := testhelpers.GetTestMySQLDB(c)

		Convey("Given the script to initialize version 0.0.0 of the database", func() {

			Convey("When executing the script", func() {
				script := strings.Split(mysqlCreationScript, ";\n")
				for _, cmd := range script[:len(script)-1] {
					_, err := db.Exec(cmd)
					So(err, ShouldBeNil)
				}
			})
		})
	})
}
