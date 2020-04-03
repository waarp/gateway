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
	Convey("Given a user entry", t, func() {
		user := &User{
			Username: "name",
			Password: []byte("password"),
		}

		Convey("When calling the `BeforeInsert` hook", func() {
			err := user.BeforeInsert(nil)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the user's password should be hashed", func() {
				hash, err := hashPassword(user.Password)
				So(err, ShouldBeNil)
				So(string(user.Password), ShouldEqual, string(hash))
			})
		})
	})
}

func TestUsersBeforeUpdate(t *testing.T) {
	Convey("Given a user entry", t, func() {
		user := &User{
			Username: "name",
			Password: []byte("password"),
		}

		Convey("When calling the `BeforeUpdate` hook", func() {
			err := user.BeforeUpdate(nil)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the user's password should be hashed", func() {
				hash, err := hashPassword(user.Password)
				So(err, ShouldBeNil)
				So(string(user.Password), ShouldEqual, string(hash))
			})
		})
	})
}

func TestUsersValidateInsert(t *testing.T) {
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

					Convey("When calling the 'ValidateInsert' function", func() {
						err := user.ValidateInsert(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new user has an ID", func() {
					user.ID = 1000

					Convey("When calling the 'ValidateInsert' function", func() {
						err := user.ValidateInsert(db)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the user's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the new user is missing a username", func() {
					user.Username = ""

					Convey("When calling the 'ValidateInsert' function", func() {
						err := user.ValidateInsert(db)

						Convey("Then the error should say that the username is missing", func() {
							So(err, ShouldBeError, "the username "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the new username is already taken", func() {
					user.Username = existing.Username

					Convey("When calling the 'ValidateInsert' function", func() {
						err := user.ValidateInsert(db)

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

func TestUsersValidateUpdate(t *testing.T) {
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
					Owner:    database.Owner,
					Username: "new",
					Password: []byte("password_new"),
				}

				Convey("Given that the new account is valid", func() {

					Convey("When calling the 'ValidateUpdate' function", func() {
						err := user.ValidateUpdate(db, old.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new user has an ID", func() {
					user.ID = 1000

					Convey("When calling the 'ValidateUpdate' function", func() {
						err := user.ValidateUpdate(db, old.ID)

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the user's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the new username is already taken", func() {
					user.Username = existing.Username

					Convey("When calling the 'ValidateUpdate' function", func() {
						err := user.ValidateUpdate(db, old.ID)

						Convey("Then the error should say that the login is already taken", func() {
							So(err, ShouldBeError, "a user named '"+user.Username+
								"' already exist")
						})
					})
				})

				Convey("Given that the new username is identical to the old one", func() {
					user.Username = old.Username

					Convey("When calling the 'ValidateUpdate' function", func() {
						err := user.ValidateUpdate(db, old.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}
