package main

import (
	"fmt"
	"io/ioutil"
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
		command := &userGetCommand{}

		Convey("Given a gateway with 1 user", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			user := &model.User{
				Username: "user",
				Password: []byte("password"),
			}
			So(db.Create(user), ShouldBeNil)

			u := rest.FromUser(user)

			Convey("Given a valid user ID", func() {
				id := fmt.Sprint(user.ID)

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the user's info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)

						So(string(cont), ShouldEqual, userInfoString(u))
					})
				})
			})

			Convey("Given an invalid user ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.UsersPath+
							"/1000' does not exist")

					})
				})
			})

			Convey("Given no user ID", func() {

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "missing user ID")
					})
				})
			})
		})
	})
}

func TestAddUser(t *testing.T) {

	Convey("Testing the user 'add' command", t, func() {
		out = testFile()
		command := &userAddCommand{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			Convey("Given valid flags", func() {
				command.Username = "user"
				command.Password = "password"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the user was added", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldStartWith, "The user '"+command.Username+
							"' was successfully added. It can be consulted at "+
							"the address: "+gw.URL+admin.APIPath+
							rest.UsersPath+"/")
					})

					Convey("Then the new partner should have been added", func() {
						user := &model.User{
							Username: command.Username,
						}
						So(db.Get(user), ShouldBeNil)

						err = bcrypt.CompareHashAndPassword(user.Password, []byte(command.Password))
						So(err, ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestDeleteUser(t *testing.T) {

	Convey("Testing the user 'delete' command", t, func() {
		out = testFile()
		command := &userDeleteCommand{}

		Convey("Given a gateway with 1 user", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			user := &model.User{
				Username: "user",
				Password: []byte("password"),
			}
			So(db.Create(user), ShouldBeNil)

			Convey("Given a valid user ID", func() {
				id := fmt.Sprint(user.ID)

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the user was deleted", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "The user n°"+id+
							" was successfully deleted from the database\n")
					})

					Convey("Then the user should have been removed", func() {
						exists, err := db.Exists(&model.User{ID: user.ID})
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})
				})
			})

			Convey("Given an invalid ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.UsersPath+
							"/1000' does not exist")
					})

					Convey("Then the partner should still exist", func() {
						exists, err := db.Exists(&model.User{ID: user.ID})
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestUpdateUser(t *testing.T) {

	Convey("Testing the user 'delete' command", t, func() {
		out = testFile()
		command := &userUpdateCommand{}

		Convey("Given a gateway with 1 user", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			user := &model.User{
				Username: "user",
				Password: []byte("password"),
			}
			So(db.Create(user), ShouldBeNil)

			Convey("Given a valid user ID", func() {
				id := fmt.Sprint(user.ID)

				Convey("Given all valid flags", func() {
					command.Username = "new_user"
					command.Password = "new_password"

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should display a message saying the user was updated", func() {
							_, err = out.Seek(0, 0)
							So(err, ShouldBeNil)
							cont, err := ioutil.ReadAll(out)
							So(err, ShouldBeNil)
							So(string(cont), ShouldEqual, "The user n°"+id+
								" was successfully updated\n")
						})

						Convey("Then the old user should have been removed", func() {
							exists, err := db.Exists(user)
							So(err, ShouldBeNil)
							So(exists, ShouldBeFalse)
						})

						Convey("Then the new user should exist", func() {
							newAccount := &model.User{
								ID:       user.ID,
								Username: command.Username,
							}
							So(db.Get(newAccount), ShouldBeNil)

							err = bcrypt.CompareHashAndPassword(newAccount.Password, []byte(command.Password))
							So(err, ShouldBeNil)
						})
					})
				})
			})

			Convey("Given an invalid ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.UsersPath+
							"/1000' does not exist")
					})

					Convey("Then the partner should stay unchanged", func() {
						exists, err := db.Exists(user)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestListUser(t *testing.T) {

	Convey("Testing the user 'list' command", t, func() {
		out = testFile()
		command := &userListCommand{}
		_, err := flags.ParseArgs(command, []string{"waarp_gateway"})
		So(err, ShouldBeNil)

		Convey("Given a gateway with 2 users", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			user1 := &model.User{
				Username: "user1",
				Password: []byte("password"),
			}
			So(db.Create(user1), ShouldBeNil)

			user2 := &model.User{
				Username: "user2",
				Password: []byte("password"),
			}
			So(db.Create(user2), ShouldBeNil)

			u1 := rest.FromUser(user1)
			u2 := rest.FromUser(user2)

			Convey("Given no parameters", func() {

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the users' info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)

						a := &rest.OutUser{Username: "admin"}
						So(string(cont), ShouldEqual, "Users:\n"+userInfoString(a)+
							userInfoString(u1)+userInfoString(u2))
					})
				})
			})
		})
	})
}
