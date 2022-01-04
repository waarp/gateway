package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

func testSelectForUpdate(db *DB) {
	bean1 := testValid{ID: 1, String: "str1"}
	bean2 := testValid{ID: 2, String: "str2"}
	bean3 := testValid{ID: 3, String: "str2"}

	db2 := &DB{}
	So(db2.Start(), ShouldBeNil)
	Reset(func() { So(db2.engine.Close(), ShouldBeNil) })

	transRes := make(chan Error, 1)
	trans2 := func(ready chan<- bool) {
		close(ready)
		transRes <- db2.Transaction(func(ses *Session) Error {
			var beans validList
			if err := ses.SelectForUpdate(&beans).Where("string=? AND id<>0", "str2").Run(); err != nil {
				return err
			}

			if len(beans) != 0 {
				return NewValidationError("%+v should be empty", beans)
			}

			return nil
		})
	}

	Convey("When executing a 'SELECT FOR UPDATE' query", func() {
		_, err := db.engine.Insert(&bean1, &bean2, &bean3)
		So(err, ShouldBeNil)

		tErr1 := db.Transaction(func(ses *Session) Error {
			var beans validList
			err := ses.SelectForUpdate(&beans).Where("string=?", "str2").Run()
			So(err, ShouldBeNil)

			ready := make(chan bool)
			go trans2(ready)
			<-ready

			err2 := ses.Exec("UPDATE test_valid SET string=? WHERE string=?",
				"new_str2", "str2")
			So(err2, ShouldBeNil)

			return nil
		})

		So(tErr1, ShouldBeNil)
		tErr2 := <-transRes
		So(tErr2, ShouldBeNil)

		var res []testValid
		So(db.engine.Find(&res), ShouldBeNil)
		So(res, ShouldContain, testValid{ID: 1, String: "str1"})
		So(res, ShouldContain, testValid{ID: 2, String: "new_str2"})
		So(res, ShouldContain, testValid{ID: 3, String: "new_str2"})
	})
}

func testIterate(db *DB) {
	bean1 := testValid{ID: 1, String: "str1"}
	bean2 := testValid{ID: 2, String: "str2"}
	bean3 := testValid{ID: 3, String: "str2"}
	bean4 := testValid{ID: 4, String: "str3"}
	bean5 := testValid{ID: 5, String: "str1"}

	shouldContain := func(query *IterateQuery, exps ...testValid) {
		Convey("When executing the query", func() {
			rows, err := query.Run()
			So(err, ShouldBeNil)
			Reset(rows.Close)

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

	runTests := func(db ReadAccess) {
		query := db.Iterate(&testValid{}).OrderBy("id", true)

		Convey("With no conditions", func() {
			shouldContain(query, bean1, bean2, bean3, bean4, bean5)
		})

		Convey("With a '=' condition", func() {
			query.Where("string=?", "str2")
			shouldContain(query, bean2, bean3)
		})

		Convey("With a '<>' condition", func() {
			query.Where("string<>?", "str2")
			shouldContain(query, bean1, bean4, bean5)
		})

		Convey("With a '>' condition", func() {
			query.Where("id>?", 3)
			shouldContain(query, bean4, bean5)
		})

		Convey("With a '<' condition", func() {
			query.Where("id<?", 3)
			shouldContain(query, bean1, bean2)
		})

		Convey("With a '>=' condition", func() {
			query.Where("id>=?", 3)
			shouldContain(query, bean3, bean4, bean5)
		})

		Convey("With a '<=' condition", func() {
			query.Where("id<=?", 3)
			shouldContain(query, bean1, bean2, bean3)
		})

		Convey("With a 'AND' condition", func() {
			query.Where("id=? AND string=?", 3, "str2")
			shouldContain(query, bean3)
		})

		Convey("With a 'OR' condition", func() {
			query.Where("string=? OR string=?", "str3", "str2")
			shouldContain(query, bean2, bean3, bean4)
		})

		Convey("With an 'IN' condition", func() {
			query.Where("string IN (?,?)", "str1", "str2")
			shouldContain(query, bean1, bean2, bean3, bean5)
		})

		Convey("With an 'IN SELECT' condition", func() {
			b1 := &testValid2{ID: 1, String: "str1"}
			b2 := &testValid2{ID: 2, String: "str2"}
			b3 := &testValid2{ID: 3, String: "str3"}
			_, err := db.getUnderlying().Insert(b1, b2, b3)
			So(err, ShouldBeNil)

			query.Where("string IN (SELECT string FROM test_valid_2 WHERE id=? OR id=?)", 2, 3)
			shouldContain(query, bean2, bean3, bean4)
		})

		Convey("With a limit and offset", func() {
			query.Limit(2, 1)
			shouldContain(query, bean2, bean3)
		})

		Convey("With a 'DISTINCT' clause", func() {
			query.Distinct("string").OrderBy("string", true)
			shouldContain(query, testValid{String: bean1.String},
				testValid{String: bean2.String}, testValid{String: bean4.String})
		})
	}

	Convey("When executing a 'ITERATE' query", func() {
		_, err := db.engine.Insert(&bean1, &bean2,
			&bean3, &bean4, &bean5)
		So(err, ShouldBeNil)

		Convey("As a Standalone query", func() {
			runTests(db)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(func() { _ = ses.session.Close() })

			runTests(ses)
		})
	})
}

func testSelect(db *DB) {
	bean1 := testValid{ID: 1, String: "str1"}
	bean2 := testValid{ID: 2, String: "str2"}
	bean3 := testValid{ID: 3, String: "str2"}
	bean4 := testValid{ID: 4, String: "str3"}
	bean5 := testValid{ID: 5, String: "str1"}

	shouldContain := func(query *SelectQuery, res *validList, exps ...testValid) {
		Convey("When executing the query", func() {
			So(query.Run(), ShouldBeNil)

			Convey("Then the result should contain the expected elements", func() {
				So(res, ShouldHaveLength, len(exps))
				for i, r := range *res {
					So(r, ShouldResemble, exps[i])
				}
			})
		})
	}

	runTests := func(db ReadAccess) {
		var res validList
		query := db.Select(&res).OrderBy("id", true)

		Convey("With no conditions", func() {
			shouldContain(query, &res, bean1, bean2, bean3, bean4, bean5)
		})

		Convey("With a '=' condition", func() {
			query.Where("string=?", "str2")
			shouldContain(query, &res, bean2, bean3)
		})

		Convey("With a '<>' condition", func() {
			query.Where("string<>?", "str2")
			shouldContain(query, &res, bean1, bean4, bean5)
		})

		Convey("With a '>' condition", func() {
			query.Where("id>?", 3)
			shouldContain(query, &res, bean4, bean5)
		})

		Convey("With a '<' condition", func() {
			query.Where("id<?", 3)
			shouldContain(query, &res, bean1, bean2)
		})

		Convey("With a '>=' condition", func() {
			query.Where("id>=?", 3)
			shouldContain(query, &res, bean3, bean4, bean5)
		})

		Convey("With a '<=' condition", func() {
			query.Where("id<=?", 3)
			shouldContain(query, &res, bean1, bean2, bean3)
		})

		Convey("With a 'AND' condition", func() {
			query.Where("id=? AND string=?", 3, "str2")
			shouldContain(query, &res, bean3)
		})

		Convey("With a 'OR' condition", func() {
			query.Where("string=? OR string=?", "str3", "str2")
			shouldContain(query, &res, bean2, bean3, bean4)
		})

		Convey("With an 'IN' condition", func() {
			query.Where("string IN (?,?)", "str1", "str2")
			shouldContain(query, &res, bean1, bean2, bean3, bean5)
		})

		Convey("With an 'IN SELECT' condition", func() {
			b1 := &testValid2{ID: 1, String: "str1"}
			b2 := &testValid2{ID: 2, String: "str2"}
			b3 := &testValid2{ID: 3, String: "str3"}
			_, err := db.getUnderlying().Insert(b1, b2, b3)
			So(err, ShouldBeNil)

			query.Where("string IN (SELECT string FROM test_valid_2 WHERE id=? OR id=?)", 2, 3)
			shouldContain(query, &res, bean2, bean3, bean4)
		})

		Convey("With a limit and offset", func() {
			query.Limit(2, 1)
			shouldContain(query, &res, bean2, bean3)
		})

		Convey("With a 'DISTINCT' clause", func() {
			query.Distinct("string").OrderBy("string", true)
			shouldContain(query, &res, testValid{String: bean1.String},
				testValid{String: bean2.String}, testValid{String: bean4.String})
		})
	}

	Convey("When executing a 'SELECT' query", func() {
		_, err := db.engine.Insert(&bean1, &bean2,
			&bean3, &bean4, &bean5)
		So(err, ShouldBeNil)

		Convey("As a Standalone query", func() {
			runTests(db)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(func() { _ = ses.session.Close() })

			runTests(ses)
		})
	})
}

func testInsert(db *DB) {
	existing := testValid{ID: 1, String: "existing"}
	newElem := testValid{ID: 2, String: "new"}

	runTests := func(db Access) {
		Convey("With a valid record", func() {
			So(db.Insert(&newElem).Run(), ShouldBeNil)

			Convey("Then the record should have been inserted", func() {
				var actuals []testValid
				So(db.getUnderlying().Find(&actuals), ShouldBeNil)
				So(actuals, ShouldHaveLength, 2)
				exp := testValid{ID: 2, String: "new"}
				So(actuals, ShouldContain, exp)
			})

			Convey("Then the `WriteHook` should have been called", func() {
				So(newElem.Hooks, ShouldEqual, "write hook")
			})
		})

		Convey("Given that the write hook fails", func() {
			newElem := testWriteFail{ID: 2}
			err := db.Insert(&newElem).Run()

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then the record should NOT have been inserted", func() {
				var actuals []testValid
				So(db.getUnderlying().Find(&actuals), ShouldBeNil)
				So(actuals, ShouldContain, existing)
			})

			Convey("Then the `WriteHook` should have been called", func() {
				So(newElem.Hooks, ShouldEqual, "write hook")
			})
		})
	}

	Convey("When executing an 'Insert' query", func() {
		_, err := db.engine.InsertOne(&existing)
		So(err, ShouldBeNil)

		Convey("As a Standalone query", func() {
			runTests(db)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(func() { _ = ses.session.Close() })

			runTests(ses)
		})
	})
}

func testGet(db *DB) {
	toGet := testValid{ID: 1, String: "update"}
	other := testValid{ID: 2, String: "other"}

	runTests := func(db Access) {
		Convey("With an existing record", func() {
			get := testValid{}
			So(db.Get(&get, "id=?", toGet.ID).Run(), ShouldBeNil)

			Convey("Then the record should have been retrieved", func() {
				So(get, ShouldResemble, toGet)
			})
		})

		Convey("With an unknown record", func() {
			unknown := testValid{}
			err := db.Get(&unknown, "id=?", 3).Run()

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	}

	Convey("When calling the 'Get' method", func() {
		_, err := db.engine.Insert(&toGet, &other)
		So(err, ShouldBeNil)

		Convey("As a Standalone query", func() {
			runTests(db)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(func() { _ = ses.session.Close() })

			runTests(ses)
		})
	})
}

func testUpdate(db *DB) {
	toUpdate := testValid{ID: 1, String: "update"}
	other := testValid{ID: 2, String: "other"}

	runTests := func(db Access) {
		Convey("With an existing record", func() {
			toUpdate.String = "updated"
			So(db.Update(&toUpdate).Run(), ShouldBeNil)

			Convey("Then the record should have been updated", func() {
				var beans []testValid
				So(db.getUnderlying().Find(&beans), ShouldBeNil)
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
			err := db.Update(&unknown).Run()

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With an a columns condition", func() {
			toUpdate.ID = 1
			toUpdate.String = "updated"
			So(db.Update(&toUpdate).Cols("id").Run(), ShouldBeNil)

			Convey("Then only the set columns should have been updated", func() {
				var beans []testValid
				So(db.getUnderlying().Find(&beans), ShouldBeNil)
				So(beans, ShouldHaveLength, 2)
				exp := testValid{ID: 1, String: "update"}
				So(beans, ShouldContain, other)
				So(beans, ShouldContain, exp)
			})

			Convey("Then the `Validate` hook should have been called", func() {
				So(toUpdate.Hooks, ShouldEqual, "write hook")
			})
		})
	}

	Convey("When calling the 'Update' method", func() {
		_, err := db.engine.Insert(&toUpdate, &other)
		So(err, ShouldBeNil)

		Convey("As a Standalone query", func() {
			runTests(db)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(func() { _ = ses.session.Close() })

			runTests(ses)
		})
	})
}

func testDelete(db *DB) {
	toDelete1 := testValid{ID: 1, String: "delete1"}
	toDelete2 := testValid{ID: 2, String: "delete2"}
	toDeleteFail := testDeleteFail{ID: 1}

	runTests := func(db Access) {
		Convey("With a valid entry", func() {
			So(db.Delete(&toDelete1).Run(), ShouldBeNil)

			Convey("Then the record should no longer be present in the database", func() {
				var beans []testValid
				So(db.getUnderlying().Find(&beans), ShouldBeNil)
				So(beans, ShouldHaveLength, 1)
				So(beans, ShouldNotContain, toDelete1)
			})

			Convey("Then the `BeforeDelete` hook should have been called", func() {
				So(toDelete1.Hooks, ShouldEqual, "delete hook")
			})
		})

		Convey("Given that the delete hook fails", func() {
			err := db.Delete(&toDeleteFail).Run()

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then the record should still be present in the database", func() {
				var beans []testDeleteFail
				So(db.getUnderlying().Find(&beans), ShouldBeNil)
				So(beans, ShouldNotBeEmpty)
				So(beans[0], ShouldResemble, testDeleteFail{ID: 1})

				if _, ok := db.(*Standalone); ok {
					Convey("Then the hook changes should have been reverted", func() {
						So(beans, ShouldHaveLength, 1)
					})
				}
			})

			Convey("Then the `BeforeDelete` hook should have been called", func() {
				So(toDeleteFail.Hooks, ShouldEqual, "delete hook")
			})
		})

		Convey("With an unknown record", func() {
			unknown := testValid{ID: 3, String: "fail"}
			err := db.Delete(&unknown).Run()

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	}

	Convey("When calling the 'Delete' method", func() {
		_, err := db.engine.Insert(&toDelete1, &toDelete2, &toDeleteFail)
		So(err, ShouldBeNil)

		Convey("As a Standalone query", func() {
			runTests(db)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(func() { _ = ses.session.Close() })

			runTests(ses)
		})
	})
}

func testDeleteAll(db *DB) {
	toDelete1 := testValid{ID: 1, String: "delete1"}
	toDelete2 := testValid{ID: 2, String: "delete2"}
	toDelete3 := testValid{ID: 3, String: "delete2"}
	toDelete4 := testValid{ID: 4, String: "delete3"}

	runTests := func(db Access) {
		Convey("With no conditions", func() {
			So(db.DeleteAll(&toDelete1).Run(), ShouldBeNil)

			Convey("Then all records should have been deleted", func() {
				var beans []testValid
				So(db.getUnderlying().Find(&beans), ShouldBeNil)
				So(beans, ShouldBeEmpty)
			})
		})
	}

	Convey("When calling the 'DeleteAll' method", func() {
		_, err := db.engine.Insert(&toDelete1, &toDelete2, &toDelete3, &toDelete4)
		So(err, ShouldBeNil)

		Convey("As a Standalone query", func() {
			runTests(db)
		})

		Convey("Inside a transaction", func() {
			ses := db.newSession()
			So(ses.session.Begin(), ShouldBeNil)
			Reset(func() { _ = ses.session.Close() })

			runTests(ses)
		})
	})
}

func testTransaction(db *DB) {
	bean := testValid{
		ID:     1,
		String: "test transaction",
	}

	Convey("Given a valid transaction", func() {
		trans := func(ses *Session) Error {
			return ses.Insert(&bean).Run()
		}

		Convey("When executing the transaction", func() {
			So(db.Transaction(trans), ShouldBeNil)

			Convey("Then the new insertion should have been committed", func() {
				exists, err := db.engine.Exist(&bean)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})
	})

	Convey("Given an invalid transaction", func() {
		trans := func(ses *Session) Error {
			So(ses.Insert(&bean).Run(), ShouldBeNil)

			return NewInternalError(fmt.Errorf("transaction failed")) //nolint:goerr113 // this is a test
		}

		Convey("When executing the transaction", func() {
			So(db.Transaction(trans), ShouldBeError)

			Convey("Then the new insertion should NOT have been committed", func() {
				exists, err := db.engine.Exist(&bean)
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})
	})
}

func testResetIncrement(db *DB) {
	bean1 := testValid{String: "str1"}
	bean2 := testValid{String: "str2"}

	Convey("When calling the 'ResetIncrement' method", func() {
		_, err := db.engine.Insert(&bean1, &bean2)
		So(err, ShouldBeNil)
		_, err = db.engine.NoAutoCondition().Where("id=id").Delete(&testValid{})
		So(err, ShouldBeNil)

		Convey("Inside a transaction", func() {
			Convey("Given that the table is empty", func() {
				ses := db.newSession()
				So(ses.session.Begin(), ShouldBeNil)
				Reset(func() { _ = ses.session.Close() })

				So(ses.ResetIncrement(&testValid{}), ShouldBeNil)
				So(ses.session.Commit(), ShouldBeNil)

				Convey("Then it should have reset the increment", func() {
					bean3 := testValid{String: "str3"}
					_, err := db.engine.Insert(&bean3)
					So(err, ShouldBeNil)

					So(bean3.ID, ShouldEqual, 1)
				})
			})

			Convey("Given that the table is NOT empty", func() {
				bean3 := testValid{String: "str3"}
				_, err := db.engine.Insert(&bean3)
				So(err, ShouldBeNil)
				So(bean3.ID, ShouldEqual, 3)

				ses := db.newSession()
				So(ses.session.Begin(), ShouldBeNil)
				Reset(func() { _ = ses.session.Close() })

				So(ses.ResetIncrement(&testValid{}), ShouldBeError, fmt.Sprintf(
					"cannot reset the increment on table %q while there are still rows in it",
					bean3.TableName()))
			})
		})
	})
}

func testDatabase(db *DB) {
	So(db.engine.CreateTables(&testValid{}, &testValid2{}, &testWriteFail{},
		&testDeleteFail{}), ShouldBeNil)
	Reset(func() {
		So(db.engine.DropTables(&testValid{}, &testValid2{}, &testWriteFail{},
			&testDeleteFail{}), ShouldBeNil)
	})

	testSelectForUpdate(db)
	testIterate(db)
	testSelect(db)
	testGet(db)
	testInsert(db)
	testUpdate(db)
	testDelete(db)
	testDeleteAll(db)
	testTransaction(db)
	testResetIncrement(db)
}

func TestSqlite(t *testing.T) {
	conf.GlobalConfig.Log.Level = "CRITICAL"
	conf.GlobalConfig.Log.LogTo = "stdout"
	conf.GlobalConfig.Database.Type = SQLite
	conf.GlobalConfig.Database.Address = filepath.Join(os.TempDir(), "test.db")
	conf.GlobalConfig.Database.AESPassphrase = filepath.Join(os.TempDir(), "sqlite_test_passphrase.aes")

	db := &DB{}
	defer func() {
		if err := db.engine.Close(); err != nil {
			t.Logf("Failed to close database: %s", err)
		}

		if err := os.Remove(conf.GlobalConfig.Database.AESPassphrase); err != nil {
			t.Logf("Failed to delete passphrase file: %s", err)
		}

		if err := os.Remove(conf.GlobalConfig.Database.Address); err != nil {
			t.Logf("Failed to delete sqlite file: %s", err)
		}
	}()

	if err := db.start(false); err != nil {
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

	conf.GlobalConfig.Log.Level = "CRITICAL"
	conf.GlobalConfig.Log.LogTo = "stdout"
	conf.GlobalConfig.Database.Type = SQLite
	conf.GlobalConfig.Database.Address = filepath.Join(os.TempDir(), "test_no_passphrase.db")
	conf.GlobalConfig.Database.AESPassphrase = filepath.Join(os.TempDir(), "test_no_passphrase.aes")

	Convey("Given a test database", t, func() {
		db := &DB{}

		Convey("When the database service is started", func() {
			So(db.Start(), ShouldBeNil)
			Reset(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				So(db.Stop(ctx), ShouldBeNil)
				So(os.Remove(conf.GlobalConfig.Database.Address), ShouldBeNil)
				So(os.Remove(conf.GlobalConfig.Database.AESPassphrase), ShouldBeNil)
			})

			Convey("Then there is a new passphrase file", func() {
				stats, err := os.Stat(conf.GlobalConfig.Database.AESPassphrase)
				So(err, ShouldBeNil)
				So(stats, ShouldNotBeNil)

				Convey("Then the permissions of the files are secure", func() {
					So(stats.Mode().Perm(), ShouldEqual, aesFilePerm)
				})
			})
		})
	})
}

func TestDatabaseStartVersionMismatch(t *testing.T) {
	conf.GlobalConfig.Log.Level = "CRITICAL"
	conf.GlobalConfig.Log.LogTo = "stdout"
	conf.GlobalConfig.Database.Type = SQLite
	conf.GlobalConfig.Database.Address = filepath.Join(os.TempDir(), "test_version_mismatch.db")
	conf.GlobalConfig.Database.AESPassphrase = filepath.Join(os.TempDir(), "test_version_mismatch.aes")

	Convey("Given a test database", t, func() {
		db := &DB{}
		So(db.Start(), ShouldBeNil)
		Reset(func() {
			_ = os.Remove(conf.GlobalConfig.Database.Address)
			_ = os.Remove(conf.GlobalConfig.Database.AESPassphrase)
		})

		Convey("Given that the database version does not match the program", func() {
			ver := &version{Current: "0.0.0"}
			_, err := db.engine.Table(ver.TableName()).Update(ver)
			So(err, ShouldBeNil)
			So(db.Stop(context.Background()), ShouldBeNil)

			Convey("When starting the database", func() {
				err := db.Start()

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, "database version mismatch")
				})
			})
		})
	})
}
