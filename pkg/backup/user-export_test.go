package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestExportUsers(t *testing.T) {
	Convey("Given a database with some users", t, func(c C) {
		db := database.TestDatabase(c)
		So(db.DeleteAll(&model.User{}).Run(), ShouldBeNil)

		user1 := &model.User{
			Username:     "user1",
			PasswordHash: hash("password1"),
			Permissions: model.PermTransfersRead | model.PermTransfersWrite |
				model.PermServersWrite |
				model.PermPartnersRead | model.PermPartnersDelete |
				model.PermRulesRead | model.PermRulesWrite | model.PermRulesDelete |
				model.PermUsersWrite | model.PermUsersDelete |
				model.PermAdminRead,
		}
		So(db.Insert(user1).Run(), ShouldBeNil)

		// Change owner for this insert
		owner := conf.GlobalConfig.GatewayName
		conf.GlobalConfig.GatewayName = "tata"

		So(db.Insert(&model.User{
			Username:     "other",
			PasswordHash: hash("other_password"),
			Permissions:  model.PermAll,
		}).Run(), ShouldBeNil)
		// Revert database owner
		conf.GlobalConfig.GatewayName = owner

		Convey("When calling the exportUsers function", func() {
			res, err := exportUsers(discard(), db)

			Convey("Then it should return no error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then it should return 1 user", func() {
				So(len(res), ShouldEqual, 1)

				Convey("Then this user should be equivalent to the DB one", func() {
					So(res[0], ShouldResemble, file.User{
						Username:     "user1",
						PasswordHash: user1.PasswordHash,
						Permissions: file.Permissions{
							Transfers:      "rw-",
							Servers:        "-w-",
							Partners:       "r-d",
							Rules:          "rwd",
							Users:          "-wd",
							Administration: "r--",
						},
					})
				})
			})
		})
	})
}
