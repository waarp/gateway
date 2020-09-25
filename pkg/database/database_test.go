package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-xorm/xorm"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
)

var (
	sqliteTestDatabase *DB
	sqliteConfig       *conf.ServerConfig
)

func init() {
	BcryptRounds = bcrypt.MinCost

	sqliteConfig = &conf.ServerConfig{}
	sqliteConfig.Database.Type = sqlite
	sqliteConfig.Database.Address = filepath.Join(os.TempDir(), "test.sqlite")
	sqliteConfig.Database.AESPassphrase = fmt.Sprintf("%s%ssqlite_test_passphrase.aes",
		os.TempDir(), string(os.PathSeparator))

	sqliteTestDatabase = &DB{Conf: sqliteConfig}
}

func testSelect(db *DB) {
	bean1 := testValid{ID: 1, String: "str1"}
	bean2 := testValid{ID: 2, String: "str2"}
	bean3 := testValid{ID: 3, String: "str2"}
	bean4 := testValid{ID: 4, String: "str3"}
	bean5 := testValid{ID: 5, String: "str1"}

	shouldContain := func(exec queryFunc, query query, exps ...testValid) {
		Convey("When executing the query", func() {
			rows, err := exec(query)
			So(err, ShouldBeNil)
			defer rows.Close()

			Convey("Then the result should contain the expected elements", func() {
				for _, exp := range exps {
					So(rows.Next(), ShouldBeTrue)
					var bean testValid
					So(rows.Scan(&bean), ShouldBeNil)
					So(bean, ShouldResemble, exp)
				}
				So(rows.Next(), ShouldBeFalse)
			})
		})
	}

	runTests := func(exec queryFunc, db xorm.Interface) {
		query := Select(&testValid{})

		Convey("With a nil bean", func() {
			_, err := exec(Select(nil))
			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})

		Convey("With no conditions", func() {
			shouldContain(exec, query, bean1, bean2, bean3, bean4, bean5)
		})

		Convey("With a '=' condition", func() {
			query.Where(Equal("string", "str2"))
			shouldContain(exec, query, bean2, bean3)
		})

		Convey("With a '<>' condition", func() {
			query.Where(NotEqual("string", "str2"))
			shouldContain(exec, query, bean1, bean4, bean5)
		})

		Convey("With a '>' condition", func() {
			query.Where(GreaterThan("id", 3))
			shouldContain(exec, query, bean4, bean5)
		})

		Convey("With a '<' condition", func() {
			query.Where(LowerThan("id", 3))
			shouldContain(exec, query, bean1, bean2)
		})

		Convey("With a '>=' condition", func() {
			query.Where(GreaterThanOrEqual("id", 3))
			shouldContain(exec, query, bean3, bean4, bean5)
		})

		Convey("With a '<=' condition", func() {
			query.Where(LowerThanOrEqual("id", 3))
			shouldContain(exec, query, bean1, bean2, bean3)
		})

		Convey("With a 'AND' condition", func() {
			query.Where(Equal("id", 3).And(Equal("string", "str2")))
			shouldContain(exec, query, bean3)
		})

		Convey("With a 'OR' condition", func() {
			query.Where(Equal("string", "str3").Or(Equal("string", "str2")))
			shouldContain(exec, query, bean2, bean3, bean4)
		})

		Convey("With an 'IN' condition", func() {
			query.Where(In("string", "str1", "str2"))
			shouldContain(exec, query, bean1, bean2, bean3, bean5)
		})

		Convey("With an 'IN SELECT' condition", func() {
			b1 := &testValid2{ID: 1, String: "str1"}
			b2 := &testValid2{ID: 2, String: "str2"}
			b3 := &testValid2{ID: 3, String: "str3"}
			_, err := db.Insert(b1, b2, b3)
			So(err, ShouldBeNil)

			query.Where(In("string", Expr("SELECT string FROM test_valid_2 WHERE id=2 OR id=3")))
			shouldContain(exec, query, bean2, bean3, bean4)
		})

		Convey("With a limit and offset", func() {
			query.Limit(2, 1)
			shouldContain(exec, query, bean2, bean3)
		})
	}

	Convey("When executing a 'SELECT' query", func() {
		_, err := db.engine.Insert(&bean1, &bean2,
			&bean3, &bean4, &bean5)
		So(err, ShouldBeNil)

		Convey("As a standalone query", func() {
			runTests(db.Query2, db.engine)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(ses.rollback)

			runTests(ses.Query2, ses.session)
		})
	})

}

func testInsert(db *DB) {
	existing := testValid{ID: 1, String: "existing"}
	newElem := testValid{ID: 2, String: "new"}

	runTests := func(exec execFunc, db xorm.Interface) {
		Convey("With a valid record", func() {
			_, err := exec(Insert(&newElem))
			So(err, ShouldBeNil)

			Convey("Then the record should have been inserted", func() {
				var actuals []testValid
				So(db.Find(&actuals), ShouldBeNil)
				So(actuals, ShouldHaveLength, 2)
				exp := testValid{ID: 2, String: "new"}
				So(actuals, ShouldContain, exp)
			})

			Convey("Then the `WriteHook` should have been called", func() {
				So(newElem.Hooks, ShouldEqual, "write hook")
			})
		})

		Convey("With a nil record ", func() {
			_, err := exec(Insert(nil))

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})
	}

	Convey("When executing an 'INSERT' query", func() {
		_, err := db.engine.InsertOne(&existing)
		So(err, ShouldBeNil)

		Convey("As a standalone query", func() {
			runTests(db.Exec, db.engine)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(ses.rollback)

			runTests(ses.Exec, ses.session)
		})
	})
}

func testUpdate(db *DB) {
	toUpdate := testValid{ID: 1, String: "update"}
	other := testValid{ID: 2, String: "other"}

	runTests := func(exec execFunc, db xorm.Interface) {
		Convey("With an existing record", func() {
			toUpdate.String = "updated"
			_, err := exec(Update(&toUpdate))
			So(err, ShouldBeNil)

			Convey("Then the record should have been updated", func() {
				var beans []testValid
				So(db.Find(&beans), ShouldBeNil)
				So(beans, ShouldHaveLength, 2)
				exp := testValid{ID: 1, String: "updated"}
				So(beans, ShouldContain, other)
				So(beans, ShouldContain, exp)
			})

			Convey("Then the `Validate` hook should have been called", func() {
				So(toUpdate.Hooks, ShouldEqual, "write hook")
			})
		})

		Convey("With an unknown record", func() {
			unknown := testValid{ID: 3, String: "fail"}
			_, err := exec(Update(&unknown))

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With an nil record", func() {
			_, err := exec(Update(nil))

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})
	}

	Convey("When calling the 'Update' method", func() {
		_, err := db.engine.Insert(&toUpdate, &other)
		So(err, ShouldBeNil)

		Convey("Using the standalone accessor", func() {
			runTests(db.Exec, db.engine)
		})

		Convey("Using the transaction accessor", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(ses.session.Close)

			runTests(ses.Exec, ses.session)
		})
	})
}

func testDelete(db *DB) {
	toDelete1 := testValid{ID: 1, String: "delete1"}
	toDelete2 := testValid{ID: 2, String: "delete2"}

	runTests := func(exec execFunc, db xorm.Interface) {
		Convey("With no conditions", func() {
			_, err := exec(Delete(&toDelete1))
			So(err, ShouldBeNil)

			Convey("Then the record should no longer be present in the database", func() {
				var beans []testValid
				So(db.Find(&beans), ShouldBeNil)
				So(beans, ShouldHaveLength, 1)
				So(beans, ShouldNotContain, toDelete1)
			})

			Convey("Then the `BeforeDelete` hook should have been called", func() {
				So(toDelete1.Hooks, ShouldEqual, "delete hook")
			})
		})

		Convey("With a nil record", func() {
			_, err := exec(Delete(nil))

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})
	}

	Convey("When calling the 'Delete' method", func() {
		_, err := db.engine.Insert(&toDelete1, &toDelete2)
		So(err, ShouldBeNil)

		Convey("Using the standalone accessor", func() {
			runTests(db.Exec, db.engine)
		})

		Convey("Using the session accessor", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(ses.session.Close)

			runTests(ses.Exec, ses.session)
		})

	})
}

func testTrans(db *DB) {
	bean := testBean{
		ID:     1,
		String: "test trans",
	}

	Convey("Given a new transaction", func() {
		ses, err := db.BeginTransaction()
		So(err, ShouldBeNil)

		Convey("Given a pending insertion operation", func() {
			err := ses.Create(&bean)
			So(err, ShouldBeNil)

			Convey("When calling the 'Commit' method", func() {
				err := ses.Commit()

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the new insertion should have been committed", func() {
						exists, err := db.engine.Exist(&bean)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("When calling the 'Rollback' method", func() {
				ses.Rollback()

				Convey("Then the new insertion should NOT have been committed", func() {
					exists, err := db.engine.Exist(&bean)
					So(err, ShouldBeNil)
					So(exists, ShouldBeFalse)
				})
			})
		})
	})
}

func testDatabase(db *DB) {
	So(db.engine.CreateTables(&testBean{}, &testValid{}, &testValid2{}), ShouldBeNil)
	Reset(func() {
		So(db.engine.DropTables(&testBean{}, &testValid{}, &testValid2{}), ShouldBeNil)
	})

	testSelect(db)
	testInsert(db)
	testUpdate(db)
	testDelete(db)
	testTrans(db)
}

func TestSqlite(t *testing.T) {
	db := sqliteTestDatabase
	defer func() {
		if err := db.engine.Close(); err != nil {
			t.Logf("Failed to close database: %s", err)
		}
		if err := os.Remove(sqliteConfig.Database.AESPassphrase); err != nil {
			t.Logf("Failed to delete passphrase file: %s", err)
		}
		if err := os.Remove(sqliteConfig.Database.Address); err != nil {
			t.Logf("Failed to delete sqlite file: %s", err)
		}
	}()
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}

	Convey("Given a Sqlite service", t, func() {
		testDatabase(db)
	})
}

func TestDatabaseStartWithNoPassPhraseFile(t *testing.T) {
	gcm := GCM
	GCM = nil
	defer func() { GCM = gcm }()

	Convey("Given a test database", t, func() {
		db := &DB{
			Conf: &conf.ServerConfig{
				Database: conf.DatabaseConfig{
					Type:          "sqlite",
					AESPassphrase: filepath.Join(os.TempDir(), "test_no_passphrase.aes"),
				},
			},
		}

		Convey("When the database service is started", func() {
			So(db.Start(), ShouldBeNil)
			Reset(func() { _ = os.Remove(db.Conf.Database.AESPassphrase) })

			Convey("Then there is a new passphrase file", func() {
				stats, err := os.Stat(db.Conf.Database.AESPassphrase)
				So(err, ShouldBeNil)
				So(stats, ShouldNotBeNil)

				Convey("Then the permissions of the files are secure", func() {
					So(stats.Mode().Perm(), ShouldEqual, aesFilePerm)
				})
			})
		})
	})
}
