package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	logConf := conf.LogConfig{
		Level: "CRITICAL",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

func TestUsersTableName(t *testing.T) {
	Convey("Given a `User` instance", t, func() {
		user := &User{}

		Convey("When calling the 'TableName' method", func() {
			name := user.TableName()

			Convey("Then it should return the name of the users table", func() {
				So(name, ShouldEqual, "users")
			})
		})
	})
}

func TestUsersBeforeInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 user", func() {
			existing := &User{
				Username: "existing",
				Password: []byte("password_existing"),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a user account", func() {
				user := &User{
					Username: "user",
					Password: []byte("password_user"),
				}

				Convey("Given that the new account is valid", func() {

					Convey("When calling the 'BeforeInsert' function", func() {
						So(user.BeforeInsert(db), ShouldBeNil)

						Convey("Then the user's password should be hashed", func() {
							hash, err := hashPassword(user.Password)
							So(err, ShouldBeNil)
							So(string(user.Password), ShouldEqual, string(hash))
						})
					})
				})

				Convey("Given that the new user has an ID", func() {
					user.ID = 1000

					Convey("When calling the 'BeforeInsert' function", func() {
						err := user.BeforeInsert(db)

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the user's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the new user is missing a username", func() {
					user.Username = ""

					Convey("When calling the 'BeforeInsert' function", func() {
						err := user.BeforeInsert(db)

						Convey("Then the error should say that the username is missing", func() {
							So(err, ShouldBeError, "the username "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the new username is already taken", func() {
					user.Username = existing.Username

					Convey("When calling the 'BeforeInsert' function", func() {
						err := user.BeforeInsert(db)

						Convey("Then the error should say that the login is already taken", func() {
							So(err, ShouldBeError, "a user named '"+user.Username+
								"' already exist")
						})
					})
				})
			})
		})
	})
}

func TestUsersBeforeUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 2 users", func() {
			existing := &User{
				Username: "existing",
				Password: []byte("password_existing"),
			}
			So(db.Create(existing), ShouldBeNil)

			old := &User{
				Username: "old",
				Password: []byte("password_old"),
			}
			So(db.Create(old), ShouldBeNil)

			Convey("Given a user account", func() {
				user := &User{
					Username: "new",
					Password: []byte("password_new"),
				}

				Convey("Given that the new account is valid", func() {

					Convey("When calling the 'BeforeUpdate' function", func() {
						So(user.BeforeUpdate(db, old.ID), ShouldBeNil)

						Convey("Then the user's password should be hashed", func() {
							hash, err := hashPassword(user.Password)
							So(err, ShouldBeNil)
							So(string(user.Password), ShouldEqual, string(hash))
						})
					})
				})

				Convey("Given that the new user has an ID", func() {
					user.ID = 1000

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := user.BeforeUpdate(db, old.ID)

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the user's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the new username is already taken", func() {
					user.Username = existing.Username

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := user.BeforeUpdate(db, old.ID)

						Convey("Then the error should say that the login is already taken", func() {
							So(err, ShouldBeError, "a user named '"+user.Username+
								"' already exist")
						})
					})
				})

				Convey("Given that the new username is identical to the old one", func() {
					user.Username = old.Username

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := user.BeforeUpdate(db, old.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

func TestUsersBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()
		owner := database.Owner
		Convey("Given the database contains 1 user for this gateway", func() {
			mine := &User{
				Username: "existing",
				Password: []byte("password_existing"),
			}
			So(db.Create(mine), ShouldBeNil)

			// Change database ownership
			database.Owner = "tata"
			other := &User{
				Username: "old",
				Password: []byte("password_old"),
			}
			So(db.Create(other), ShouldBeNil)
			// Revert database ownership
			database.Owner = owner

			// Delete base admin
			So(db.Delete(&User{Username: "admin"}), ShouldBeNil)

			Convey("When calling BeforeDelete", func() {
				err := mine.BeforeDelete(db)

				Convey("Then it should return an eror", func() {
					So(err, ShouldNotBeNil)

					Convey("Then the error should say ''", func() {
						So(err.Error(), ShouldEqual, "cannot delete gateway last admin")
					})
				})
			})
		})

		Convey("Given the database contains 2 users for this gateway", func() {
			mine := &User{
				Username: "existing",
				Password: []byte("password_existing"),
			}
			So(db.Create(mine), ShouldBeNil)

			other := &User{
				Username: "old",
				Password: []byte("password_old"),
			}
			So(db.Create(other), ShouldBeNil)

			// Delete base admin
			So(db.Delete(&User{Username: "admin"}), ShouldBeNil)

			Convey("When calling BeforeDelete", func() {
				err := mine.BeforeDelete(db)

				Convey("Then it should return No eror", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
