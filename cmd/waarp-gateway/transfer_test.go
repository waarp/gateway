package main

import (
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAddTransfer(t *testing.T) {

	Convey("Testing the partner 'add' command", t, func() {
		command := &transferAddCommand{}

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			Convey("Given a valid remote agents", func() {
				p := model.RemoteAgent{
					Name:        "test",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"port":1}`),
				}
				err := db.Create(&p)
				So(err, ShouldBeNil)

				// TODO delete Given a valid Certificate
				Convey("Given a valid certificate", func() {
					c := model.Cert{
						Name:        "test",
						PublicKey:   []byte("test"),
						Certificate: []byte("test"),
						OwnerType:   "remote_agents",
						OwnerID:     p.ID,
					}
					err := db.Create(&c)
					So(err, ShouldBeNil)

					Convey("Given a valid account", func() {
						a := model.RemoteAccount{
							Login:         "login",
							Password:      []byte("password"),
							RemoteAgentID: p.ID,
						}
						err := db.Create(&a)
						So(err, ShouldBeNil)

						Convey("Given a valid rule", func() {
							r := model.Rule{
								Name:  "rule",
								IsGet: false,
							}
							err := db.Create(&r)
							So(err, ShouldBeNil)

							Convey("Given valid flags", func() {
								command.ServerID = p.ID
								command.AccountID = a.ID
								command.RuleID = r.ID
								command.File = "test"

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should NOT return an error", func() {
										So(err, ShouldBeNil)
									})
								})
							})

							Convey("Given no rule", func() {
								command.ServerID = p.ID
								command.AccountID = a.ID
								command.File = "test"

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should return an error", func() {
										So(err, ShouldNotBeNil)
									})
								})
							})

							Convey("Given no account", func() {
								command.ServerID = p.ID
								command.RuleID = r.ID
								command.File = "test"

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should return an error", func() {
										So(err, ShouldNotBeNil)
									})
								})
							})

							Convey("Given no remote", func() {
								command.AccountID = a.ID
								command.RuleID = r.ID
								command.File = "test"

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should return an error", func() {
										So(err, ShouldNotBeNil)
									})
								})
							})

							Convey("Given no File", func() {
								command.ServerID = p.ID
								command.AccountID = a.ID
								command.RuleID = r.ID

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should return an error", func() {
										So(err, ShouldNotBeNil)
									})
								})
							})

							Convey("Given another remote agent", func() {
								p2 := model.RemoteAgent{
									Name:        "dummy",
									Protocol:    "sftp",
									ProtoConfig: []byte(`{"port":1}`),
								}
								err := db.Create(&p2)
								So(err, ShouldBeNil)

								Convey("Given an account link to another remote agent", func() {
									a2 := model.RemoteAccount{
										Login:         "login",
										Password:      []byte("password"),
										RemoteAgentID: p2.ID,
									}
									err := db.Create(&a2)
									So(err, ShouldBeNil)

									Convey("Given an incorrect account", func() {
										command.ServerID = p.ID
										command.AccountID = a2.ID
										command.RuleID = r.ID
										command.File = "test"

										Convey("When executing the command", func() {
											addr := gw.Listener.Addr().String()
											dsn := "http://admin:admin_password@" + addr
											auth.DSN = dsn

											err := command.Execute(nil)

											Convey("Then it should return an error", func() {
												So(err, ShouldNotBeNil)
											})
										})
									})

								})
							})
						})
					})
				})
			})
		})
	})

}
