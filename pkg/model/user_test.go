package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

func TestUsersTableName(t *testing.T) {
	Convey("Given a `User` instance", t, func() {
		user := &User{}

		Convey("When calling the 'TableName' method", func() {
			name := user.TableName()

			Convey("Then it should return the name of the users table", func() {
				So(name, ShouldEqual, TableUsers)
			})
		})
	})
}

func TestUsersBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains 1 user", func() {
			existing := &User{
				Username: "existing",
				Password: []byte("password_existing"),
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a user account", func() {
				user := &User{
					Username:    "user",
					Password:    []byte("password_user"),
					Permissions: PermPartnersRead,
				}

				Convey("Given that the new account is valid", func() {

					Convey("When calling the 'BeforeWrite' function", func() {
						So(user.BeforeWrite(db), ShouldBeNil)

						Convey("Then the user's password should be hashed", func() {
							So(bcrypt.CompareHashAndPassword(user.Password,
								[]byte("password_user")), ShouldBeNil)
						})
					})
				})

				Convey("Given that the new user is missing a username", func() {
					user.Username = ""

					Convey("When calling the 'BeforeWrite' function", func() {
						err := user.BeforeWrite(db)

						Convey("Then the error should say that the username is missing", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the username cannot be empty"))
						})
					})
				})

				Convey("Given that the new username is already taken", func() {
					user.Username = existing.Username

					Convey("When calling the 'BeforeWrite' function", func() {
						err := user.BeforeWrite(db)

						Convey("Then the error should say that the login is already taken", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"a user named '%s' already exist", user.Username))
						})
					})
				})
			})
		})
	})
}

func TestUsersBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		owner := conf.GlobalConfig.GatewayName
		Convey("Given the database contains 1 user for this gateway", func() {
			mine := &User{
				Username: "existing",
				Password: []byte("password_existing"),
			}
			So(db.Insert(mine).Run(), ShouldBeNil)

			// Change database ownership
			conf.GlobalConfig.GatewayName = "tata"
			other := &User{
				Username: "old",
				Password: []byte("password_old"),
			}
			So(db.Insert(other).Run(), ShouldBeNil)
			// Revert database ownership
			conf.GlobalConfig.GatewayName = owner

			// Delete base admin
			So(db.DeleteAll(&User{}).Where("username='admin'").Run(), ShouldBeNil)

			Convey("When calling BeforeDelete", func() {
				err := db.Transaction(func(ses *database.Session) database.Error {
					return mine.BeforeDelete(ses)
				})

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)

					Convey("Then the error should say the last admin cannot be deleted", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"cannot delete gateway last admin"))
					})
				})
			})
		})

		Convey("Given the database contains 2 users for this gateway", func() {
			mine := &User{
				Username: "existing",
				Password: []byte("password_existing"),
			}
			So(db.Insert(mine).Run(), ShouldBeNil)

			other := &User{
				Username: "old",
				Password: []byte("password_old"),
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			// Delete base admin
			So(db.DeleteAll(&User{}).Where("username='admin'").Run(), ShouldBeNil)

			Convey("When calling BeforeDelete", func() {
				err := db.Transaction(func(ses *database.Session) database.Error {
					return mine.BeforeDelete(ses)
				})

				Convey("Then it should return No error", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
