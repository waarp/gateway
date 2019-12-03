package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetRule(t *testing.T) {

	Convey("Testing the rule 'get' command", t, func() {
		out = testFile()
		command := &ruleGetCommand{}

		Convey("Given a gateway with 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			rule := &model.Rule{
				Name:    "test rule",
				Comment: "this is a test rule",
				IsSend:  false,
				Path:    "/test/rule/path",
			}

			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid rule ID", func() {
				id := fmt.Sprint(rule.ID)

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the rule's info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Rule n°1:\n"+
							"      Name: "+rule.Name+"\n"+
							"   Comment: "+rule.Comment+"\n"+
							" Direction: RECEIVE\n"+
							"      Path: "+rule.Path+"\n",
						)
					})
				})
			})

			Convey("Given an invalid rule ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+admin.RulesPath+
							"/1000' does not exist")

					})
				})
			})
		})
	})
}

func TestAddRule(t *testing.T) {

	Convey("Testing the rule 'add' command", t, func() {
		out = testFile()
		command := &ruleAddCommand{}

		Convey("Given a gateway with 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			existing := &model.Rule{
				Name:    "existing rule",
				Comment: "comment about existing rule",
				IsSend:  false,
				Path:    "/existing/rule/path",
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("When adding a new rule", func() {
				command.Name = "new rule"
				command.Comment = "comment about new rule"
				command.Direction = "RECEIVE"
				command.Path = "/new/rule/path"

				Convey("Given valid parameters", func() {

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should display a message saying the rule was added", func() {
							So(getOutput(), ShouldEqual, "The rule '"+command.Name+
								"' was successfully added. It can be consulted at "+
								"the address: "+gw.URL+admin.APIPath+
								admin.RulesPath+"/2\n")
						})

						Convey("Then the new rule should have been added", func() {
							rule := &model.Rule{
								ID:      2,
								Name:    command.Name,
								Comment: command.Comment,
								IsSend:  command.Direction == "SEND",
								Path:    command.Path,
							}
							exists, err := db.Exists(rule)
							So(err, ShouldBeNil)
							So(exists, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the rule's name already exist", func() {
					command.Name = existing.Name

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute(nil)

						Convey("Then it should return an error", func() {
							So(err, ShouldNotBeNil)
						})

						Convey("Then is should display a message saying the rule already exist", func() {
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"A rule named '"+command.Name+"' with send = "+
								fmt.Sprint(command.Direction == "SEND")+
								" already exist")
						})

						Convey("Then the new rule should not have been added", func() {
							rule := &model.Rule{
								Comment: command.Comment,
								IsSend:  command.Direction == "SEND",
								Path:    command.Path,
							}
							exists, err := db.Exists(rule)
							So(err, ShouldBeNil)
							So(exists, ShouldBeFalse)
						})

						Convey("Then the old rule should still exist", func() {
							exists, err := db.Exists(existing)
							So(err, ShouldBeNil)
							So(exists, ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}

func TestDeleteRule(t *testing.T) {

	Convey("Testing the rule 'delete' command", t, func() {
		out = testFile()
		command := &ruleDeleteCommand{}

		Convey("Given a gateway with 1 rule", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			existing := &model.Rule{
				Name:    "existing rule",
				Comment: "comment about existing rule",
				IsSend:  false,
				Path:    "/existing/rule/path",
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a valid rule ID", func() {
				id := fmt.Sprint(existing.ID)

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the rule was deleted", func() {
						So(getOutput(), ShouldEqual, "The rule n°"+id+
							" was successfully deleted from the database\n")
					})

					Convey("Then the rule should have been removed", func() {
						exists, err := db.Exists(existing)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})
				})
			})

			Convey("Given an invalid rule ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+admin.RulesPath+
							"/1000' does not exist")
					})

					Convey("Then the rule should still exist", func() {
						exists, err := db.Exists(existing)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestListRules(t *testing.T) {

	Convey("Testing the rule 'list' command", t, func() {
		out = testFile()
		command := &ruleListCommand{}
		_, err := flags.ParseArgs(command, []string{"waarp_gateway"})
		So(err, ShouldBeNil)

		Convey("Given a gateway with 2 rules", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			rule1 := &model.Rule{
				Name:    "rule 1",
				Comment: "rule 1 comment",
				IsSend:  true,
				Path:    "/rule/1/path",
			}
			So(db.Create(rule1), ShouldBeNil)

			rule2 := &model.Rule{
				Name:    "rule 2",
				Comment: "rule 2 comment",
				IsSend:  true,
				Path:    "/rule/2/path",
			}
			So(db.Create(rule2), ShouldBeNil)

			Convey("Given no parameters", func() {

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the rule' info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Rules:\n"+
							"Rule n°1:\n"+
							"      Name: "+rule1.Name+"\n"+
							"   Comment: "+rule1.Comment+"\n"+
							" Direction: SEND\n"+
							"      Path: "+rule1.Path+"\n"+
							"Rule n°2:\n"+
							"      Name: "+rule2.Name+"\n"+
							"   Comment: "+rule2.Comment+"\n"+
							" Direction: SEND\n"+
							"      Path: "+rule2.Path+"\n",
						)
					})
				})
			})

			Convey("Given a 'limit' parameter of 1", func() {
				command.Limit = 1

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display only 1 rule's info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Rules:\n"+
							"Rule n°1:\n"+
							"      Name: "+rule1.Name+"\n"+
							"   Comment: "+rule1.Comment+"\n"+
							" Direction: SEND\n"+
							"      Path: "+rule1.Path+"\n",
						)
					})
				})
			})

			Convey("Given an 'offset' parameter of 1", func() {
				command.Offset = 1

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should NOT display the 1st rule", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Rules:\n"+
							"Rule n°2:\n"+
							"      Name: "+rule2.Name+"\n"+
							"   Comment: "+rule2.Comment+"\n"+
							" Direction: SEND\n"+
							"      Path: "+rule2.Path+"\n",
						)
					})
				})
			})

			Convey("Given the 'desc' flag is set", func() {
				command.DescOrder = true

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the rules' info in reverse", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Rules:\n"+
							"Rule n°2:\n"+
							"      Name: "+rule2.Name+"\n"+
							"   Comment: "+rule2.Comment+"\n"+
							" Direction: SEND\n"+
							"      Path: "+rule2.Path+"\n"+
							"Rule n°1:\n"+
							"      Name: "+rule1.Name+"\n"+
							"   Comment: "+rule1.Comment+"\n"+
							" Direction: SEND\n"+
							"      Path: "+rule1.Path+"\n",
						)
					})
				})
			})
		})
	})
}
