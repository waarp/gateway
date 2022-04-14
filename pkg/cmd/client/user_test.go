package wg

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func userInfoString(u *api.OutUser) string {
	return "● User " + u.Username + "\n" +
		"    Permissions:\n" +
		"    ├─Transfers: " + u.Perms.Transfers + "\n" +
		"    ├─Servers:   " + u.Perms.Servers + "\n" +
		"    ├─Partners:  " + u.Perms.Partners + "\n" +
		"    ├─Rules:     " + u.Perms.Rules + "\n" +
		"    └─Users:     " + u.Perms.Users + "\n"
}

func TestGetUser(t *testing.T) {
	Convey("Testing the user 'get' command", t, func() {
		out = testFile()
		command := &userGet{}

		Convey("Given a gateway with 1 user", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			user := &model.User{
				Username:     "toto",
				PasswordHash: hash("password"),
				Permissions: model.PermTransfersRead |
					model.PermServersRead |
					model.PermPartnersRead |
					model.PermRulesRead |
					model.PermUsersRead,
			}
			So(db.Insert(user).Run(), ShouldBeNil)

			Convey("Given a valid username", func() {
				args := []string{user.Username}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the user's info", func() {
						u := rest.FromUser(user)
						So(getOutput(), ShouldEqual, userInfoString(u))
					})
				})
			})

			Convey("Given an invalid username", func() {
				args := []string{"tata"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "user 'tata' not found")
					})
				})
			})
		})
	})
}

func TestAddUser(t *testing.T) {
	Convey("Testing the user 'add' command", t, func() {
		out = testFile()
		command := &userAdd{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{"-u", "user", "-p", "password", "-r", "T=r,S=r,P=r"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the user was added", func() {
						So(getOutput(), ShouldEqual, "The user "+command.Username+
							" was successfully added.\n")
					})

					Convey("Then the new partner should have been added", func() {
						var users model.Users
						So(db.Select(&users).Run(), ShouldBeNil)
						So(len(users), ShouldEqual, 2)

						So(bcrypt.CompareHashAndPassword([]byte(users[1].PasswordHash),
							[]byte("password")), ShouldBeNil)
						exp := model.User{
							Owner:        conf.GlobalConfig.GatewayName,
							ID:           2,
							Username:     "user",
							PasswordHash: users[1].PasswordHash,
							Permissions: model.PermTransfersRead |
								model.PermServersRead | model.PermPartnersRead,
						}
						So(users[1], ShouldResemble, exp)
					})
				})
			})
		})
	})
}

func TestDeleteUser(t *testing.T) {
	Convey("Testing the user 'delete' command", t, func() {
		out = testFile()
		command := &userDelete{}

		Convey("Given a gateway with 1 user", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			user := &model.User{
				Username:     "user",
				PasswordHash: hash("password"),
			}
			So(db.Insert(user).Run(), ShouldBeNil)

			Convey("Given a valid username", func() {
				args := []string{user.Username}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the user was deleted", func() {
						So(getOutput(), ShouldEqual, "The user "+user.Username+
							" was successfully deleted.\n")
					})

					Convey("Then the user should have been removed", func() {
						var users model.Users
						So(db.Select(&users).Run(), ShouldBeNil)
						So(users, ShouldNotContain, *user)
					})
				})
			})

			Convey("Given an invalid username", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "user 'toto' not found")
					})

					Convey("Then the partner should still exist", func() {
						var users model.Users
						So(db.Select(&users).Run(), ShouldBeNil)
						So(users, ShouldContain, *user)
					})
				})
			})
		})
	})
}

func TestUpdateUser(t *testing.T) {
	Convey("Testing the user 'delete' command", t, func() {
		out = testFile()
		command := &userUpdate{}

		Convey("Given a gateway with 1 user", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			user := &model.User{
				Username:     "user",
				PasswordHash: hash("password"),
				Permissions: model.PermTransfersRead |
					model.PermServersRead |
					model.PermPartnersRead |
					model.PermRulesRead |
					model.PermUsersRead,
			}
			So(db.Insert(user).Run(), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{
					user.Username, "-u", "new_user",
					"-p", "new_password", "-r", "T+w,S-rw,P=wd,R+w-r,U=w",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the user was updated", func() {
						So(getOutput(), ShouldEqual, "The user new_user "+
							"was successfully updated.\n")
					})

					Convey("Then the new user should exist", func() {
						var users model.Users
						So(db.Select(&users).Run(), ShouldBeNil)
						So(len(users), ShouldEqual, 2)

						So(bcrypt.CompareHashAndPassword([]byte(users[1].PasswordHash),
							[]byte("new_password")), ShouldBeNil)
						exp := model.User{
							Owner:        conf.GlobalConfig.GatewayName,
							ID:           user.ID,
							Username:     "new_user",
							PasswordHash: users[1].PasswordHash,
							Permissions: model.PermTransfersRead | model.PermTransfersWrite |
								model.PermPartnersWrite | model.PermPartnersDelete |
								model.PermRulesWrite |
								model.PermUsersWrite,
						}
						So(users, ShouldContain, exp)
					})
				})
			})

			Convey("Given an invalid username", func() {
				args := []string{"toto", "-u", "new_user", "-p", "new_password"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "user 'toto' not found")
					})

					Convey("Then the partner should stay unchanged", func() {
						var users model.Users
						So(db.Select(&users).Run(), ShouldBeNil)
						So(users, ShouldContain, *user)
					})
				})
			})
		})
	})
}

func TestListUser(t *testing.T) {
	Convey("Testing the user 'list' command", t, func() {
		out = testFile()
		command := &userList{}

		Convey("Given a gateway with 2 users", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			user1 := &model.User{
				Username:     "user1",
				PasswordHash: hash("password"),
				Permissions:  model.PermUsersRead,
			}
			So(db.Insert(user1).Run(), ShouldBeNil)
			var err error
			addr, err = url.Parse("http://user1:password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			user2 := &model.User{
				Username:     "user2",
				PasswordHash: hash("password"),
			}
			So(db.Insert(user2).Run(), ShouldBeNil)

			u1 := rest.FromUser(user1)
			u2 := rest.FromUser(user2)

			Convey("Given no parameters", func() {
				args := []string{}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the users' info", func() {
						So(getOutput(), ShouldEqual, "Users:\n"+
							userInfoString(u1)+userInfoString(u2))
					})
				})
			})

			Convey("Given a 'limit' parameter of 1", func() {
				args := []string{"-l", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should only display 1 user's info", func() {
						So(getOutput(), ShouldEqual, "Users:\n"+
							userInfoString(u1))
					})
				})
			})

			Convey("Given an 'offset' parameter of 1", func() {
				args := []string{"-o", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should NOT display the 1st user's info", func() {
						So(getOutput(), ShouldEqual, "Users:\n"+
							userInfoString(u2))
					})
				})
			})

			Convey("Given a 'sort' parameter of 'username-'", func() {
				args := []string{"-s", "username-"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should the users' info in reverse", func() {
						So(getOutput(), ShouldEqual, "Users:\n"+
							userInfoString(u2)+userInfoString(u1))
					})
				})
			})
		})
	})
}
