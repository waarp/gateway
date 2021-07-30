package migrations

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
	. "github.com/smartystreets/goconvey/convey"
)

func testVer0_5_0RemoveRulePathSlash(eng *migration.Engine, dialect string) {
	Convey("Given the 0.5.0 rule slash removal script", func() {
		setupDatabaseUpTo(eng, ver0_5_0RemoveRulePathSlash{})

		query := `INSERT INTO rules (name, comment, send, path, in_path, 
            out_path, work_path) VALUES (?, ?, ?, ?, ?, ?, ?)`
		if dialect == migration.PostgreSQL {
			query = `INSERT INTO rules (name, comment, send, path, in_path, 
				out_path, work_path) VALUES ($1, $2, $3, $4, $5, $6, $7)`
		}

		_, err := eng.DB.Exec(query, "send", "", true, "/send_path", "", "", "")
		So(err, ShouldBeNil)
		_, err = eng.DB.Exec(query, "recv", "", false, "/recv_path", "", "", "")
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0RemoveRulePathSlash{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have removed the leading slash", func() {
				rows, err := eng.DB.Query(`SELECT path FROM rules ORDER BY id`)
				So(err, ShouldBeNil)
				defer rows.Close()

				var path1, path2 string
				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&path1), ShouldBeNil)
				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&path2), ShouldBeNil)

				So(path1, ShouldEqual, "send_path")
				So(path2, ShouldEqual, "recv_path")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0RemoveRulePathSlash{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have restored the leading slash", func() {
					rows, err := eng.DB.Query(`SELECT path FROM rules ORDER BY id`)
					So(err, ShouldBeNil)
					defer rows.Close()

					var path1, path2 string
					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&path1), ShouldBeNil)
					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&path2), ShouldBeNil)

					So(path1, ShouldEqual, "/send_path")
					So(path2, ShouldEqual, "/recv_path")
				})
			})
		})
	})
}

func testVer0_5_0CheckRulePathAncestor(eng *migration.Engine, dialect string) {
	Convey("Given the 0.5.0 rule path ancestry check", func() {
		setupDatabaseUpTo(eng, ver0_5_0CheckRulePathParent{})

		query := `INSERT INTO rules (name, comment, send, path, in_path, 
            out_path, work_path) VALUES (?, ?, ?, ?, ?, ?, ?)`
		if dialect == migration.PostgreSQL {
			query = `INSERT INTO rules (name, comment, send, path, in_path, 
				out_path, work_path) VALUES ($1, $2, $3, $4, $5, $6, $7)`
		}

		_, err := eng.DB.Exec(query, "send", "", true, "dir/send_path", "", "", "")
		So(err, ShouldBeNil)
		_, err = eng.DB.Exec(query, "recv", "", false, "dir/recv_path", "", "", "")
		So(err, ShouldBeNil)

		Convey("Given that no rule path is another one's parent", func() {
			Convey("When applying the migration", func() {
				err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0CheckRulePathParent{}}})

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that a rule path IS another one's parent", func() {
			query := `UPDATE rules SET path=? WHERE name=?`
			if dialect == migration.PostgreSQL {
				query = `UPDATE rules SET path=$1 WHERE name=$2`
			}
			_, err := eng.DB.Exec(query, "dir", "recv")
			So(err, ShouldBeNil)

			Convey("When applying the migration", func() {
				err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0CheckRulePathParent{}}})

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, "the path of the rule 'recv' (dir) "+
						"must be changed so that it is no longer a parent of the "+
						"path of rule 'send' (dir/send_path)")
				})
			})
		})
	})
}
