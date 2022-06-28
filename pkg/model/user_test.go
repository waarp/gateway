package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
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
		db := database.TestDatabase(c)

		Convey("Given the database contains 1 user", func() {
			existing := &User{
				Username:     "existing",
				PasswordHash: hash("password_existing"),
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a user account", func() {
				user := &User{
					Username:     "user",
					PasswordHash: hash("password_user"),
					Permissions:  PermPartnersRead,
				}

				Convey("Given that the new account is valid", func() {
					Convey("When calling the 'BeforeWrite' function", func() {
						So(user.BeforeWrite(db), ShouldBeNil)

						Convey("Then the user's password should be hashed", func() {
							So(bcrypt.CompareHashAndPassword([]byte(user.PasswordHash),
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
		db := database.TestDatabase(c)
		owner := conf.GlobalConfig.GatewayName

		Convey("Given the database contains 1 user for this gateway", func() {
			mine := &User{
				Username:     "existing",
				PasswordHash: hash("password_existing"),
				Permissions:  PermAll,
			}
			So(db.Insert(mine).Run(), ShouldBeNil)

			mine2 := &User{
				Username:     "existing2",
				PasswordHash: hash("password_existing2"),
			}
			So(db.Insert(mine2).Run(), ShouldBeNil)

			// Change database ownership
			conf.GlobalConfig.GatewayName = "tata"
			other := &User{
				Username:     "old",
				PasswordHash: hash("password_old"),
			}
			So(db.Insert(other).Run(), ShouldBeNil)
			// Revert database ownership
			conf.GlobalConfig.GatewayName = owner

			// Delete base admin
			So(db.DeleteAll(&User{}).Where("username=?", "admin").Run(), ShouldBeNil)

			Convey("When calling BeforeDelete", func() {
				err := db.Transaction(func(ses *database.Session) database.Error {
					return mine.BeforeDelete(ses)
				})

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)

					Convey("Then the error should say the last admin cannot be deleted", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"cannot delete the last gateway admin"))
					})
				})
			})
		})

		Convey("Given the database contains 2 users for this gateway", func() {
			mine := &User{
				Username:     "existing",
				PasswordHash: hash("password_existing"),
			}
			So(db.Insert(mine).Run(), ShouldBeNil)

			other := &User{
				Username:     "old",
				PasswordHash: hash("password_old"),
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

func TestUserInit(t *testing.T) {
	Convey("Given the user init function", t, func(c C) {
		init := (&User{}).Init

		Convey("Given a database with no admins", func() {
			db := database.TestDatabaseNoInit(c)

			// Add a user from another gateway
			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "tata"
			other := &User{
				Username:     "other",
				PasswordHash: hash("password_other"),
				Permissions:  PermAll,
			}
			So(db.Insert(other).Run(), ShouldBeNil)
			conf.GlobalConfig.GatewayName = owner

			Convey("When calling the user init function", func() {
				So(init(db), ShouldBeNil)

				Convey("Then it should have added the default admin user", func() {
					var users Users
					So(db.Select(&users).Where("owner=?", conf.GlobalConfig.GatewayName).Run(), ShouldBeNil)
					So(users, ShouldHaveLength, 1)

					So(users[0].Username, ShouldEqual, "admin")
					So(bcrypt.CompareHashAndPassword([]byte(users[0].PasswordHash),
						[]byte("admin_password")), ShouldBeNil)
					So(users[0].Permissions, ShouldEqual, PermAll)
				})
			})
		})

		Convey("Given an database with an admin", func() {
			db := database.TestDatabaseNoInit(c)

			// Add another admin
			other := &User{
				Username:     "other",
				PasswordHash: hash("password_other"),
				Permissions:  PermUsersWrite,
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			Convey("When calling the user init function", func() {
				So(init(db), ShouldBeNil)

				Convey("Then it should NOT have added the default admin user", func() {
					var users Users
					So(db.Select(&users).Where("owner=?", conf.GlobalConfig.GatewayName).Run(), ShouldBeNil)
					So(users, ShouldHaveLength, 1)

					So(users[0], ShouldResemble, *other)
				})
			})
		})
	})
}
