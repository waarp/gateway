package main

import (
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

func userInfoString(u *rest.OutUser) string {
	return "● User " + u.Username + "\n"
}

func TestGetUser(t *testing.T) {

	Convey("Testing the user 'get' command", t, func() {
		out = testFile()
		command := &userGet{}

		Convey("Given a gateway with 1 user", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			user := &model.User{
				Username: "user",
				Password: []byte("password"),
			}
			So(db.Create(user), ShouldBeNil)

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
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "user 'toto' not found")
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

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given valid flags", func() {
				args := []string{"-u", "user", "-p", "password"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the user was added", func() {
						So(getOutput(), ShouldEqual, "The user "+command.Username+
							" was successfully added.\n")
					})

					Convey("Then the new partner should have been added", func() {
						user := &model.User{
							Username: command.Username,
						}
						So(db.Get(user), ShouldBeNil)
						So(bcrypt.CompareHashAndPassword(user.Password,
							[]byte(command.Password)), ShouldBeNil)
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

		Convey("Given a gateway with 1 user", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			user := &model.User{
				Username: "user",
				Password: []byte("password"),
			}
			So(db.Create(user), ShouldBeNil)

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
						exists, err := db.Exists(user)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
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
						So(db.Get(user), ShouldBeNil)
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

		Convey("Given a gateway with 1 user", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			user := &model.User{
				Username: "user",
				Password: []byte("password"),
			}
			So(db.Create(user), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{"-u", "new_user", "-p", "new_password", user.Username}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the user was updated", func() {
						So(getOutput(), ShouldEqual, "The user new_user "+
							"was successfully updated.\n")
					})

					Convey("Then the old user should have been removed", func() {
						exists, err := db.Exists(user)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})

					Convey("Then the new user should exist", func() {
						update := &model.User{
							ID:       user.ID,
							Username: command.Username,
						}
						So(db.Get(update), ShouldBeNil)
						So(bcrypt.CompareHashAndPassword(update.Password,
							[]byte(command.Password)), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid username", func() {
				args := []string{"-u", "new_user", "-p", "new_password", "toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "user 'toto' not found")
					})

					Convey("Then the partner should stay unchanged", func() {
						So(db.Get(user), ShouldBeNil)
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

		Convey("Given a gateway with 2 users", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			So(db.Execute("DELETE FROM users WHERE username='admin'"), ShouldBeNil)

			user1 := &model.User{
				Username: "user1",
				Password: []byte("password"),
			}
			So(db.Create(user1), ShouldBeNil)
			commandLine.Args.Address = "http://user1:password@" + gw.Listener.Addr().String()

			user2 := &model.User{
				Username: "user2",
				Password: []byte("password"),
			}
			So(db.Create(user2), ShouldBeNil)

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
