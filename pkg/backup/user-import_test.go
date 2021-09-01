package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func shouldBeHashOf(hash, pswd string) {
	So(bcrypt.CompareHashAndPassword([]byte(hash), []byte(pswd)), ShouldBeNil)
}

func TestImportUsers(t *testing.T) {
	Convey("Given a database with some users", t, func(c C) {
		db := database.TestDatabase(c)
		So(db.DeleteAll(&model.User{}).Run(), ShouldBeNil)

		dbUser1a := &model.User{
			Username:     "user1",
			PasswordHash: hash("password1a"),
			Permissions:  model.PermTransfersRead,
		}
		So(db.Insert(dbUser1a).Run(), ShouldBeNil)

		owner := conf.GlobalConfig.GatewayName
		conf.GlobalConfig.GatewayName = "toto"
		dbUser1b := &model.User{
			Username:     dbUser1a.Username,
			PasswordHash: hash("password1b"),
			Permissions:  model.PermAll,
		}
		So(db.Insert(dbUser1b).Run(), ShouldBeNil)
		conf.GlobalConfig.GatewayName = owner

		Convey("Given a new user to import", func() {
			user := file.User{
				Username:    "user2",
				Password:    "password2",
				Permissions: file.Permissions{Servers: "-w-"},
			}
			users := []file.User{user}

			Convey("When importing the users", func() {
				So(importUsers(discard(), db, users), ShouldBeNil)

				Convey("Then it should have inserted the users in database", func() {
					var dbUsers model.Users
					So(db.Select(&dbUsers).OrderBy("id", true).Run(), ShouldBeNil)
					So(dbUsers, ShouldHaveLength, 3)

					So(dbUsers[0], ShouldResemble, *dbUser1a)
					So(dbUsers[1], ShouldResemble, *dbUser1b)

					So(dbUsers[2].Username, ShouldEqual, user.Username)
					shouldBeHashOf(dbUsers[2].PasswordHash, user.Password)
					So(dbUsers[2].Permissions, ShouldEqual, model.PermServersWrite)
				})
			})
		})

		Convey("Given an existing user to import", func() {
			user := file.User{
				Username:    dbUser1a.Username,
				Password:    "password2",
				Permissions: file.Permissions{Servers: "-w-"},
			}
			users := []file.User{user}

			Convey("When importing the users", func() {
				So(importUsers(discard(), db, users), ShouldBeNil)

				Convey("Then it should have updated the users", func() {
					var dbUsers model.Users
					So(db.Select(&dbUsers).OrderBy("id", true).Run(), ShouldBeNil)
					So(dbUsers, ShouldHaveLength, 2)

					So(dbUsers[1], ShouldResemble, *dbUser1b)

					So(dbUsers[0].Username, ShouldEqual, dbUser1a.Username)
					shouldBeHashOf(dbUsers[0].PasswordHash, user.Password)
					So(dbUsers[0].Permissions, ShouldEqual, model.PermServersWrite)
				})
			})
		})
	})
}
