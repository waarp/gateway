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

func TestAddRuleAccess(t *testing.T) {

	Convey("Testing the rule access 'grant' command", t, func() {
		out = testFile()
		command := &ruleAccessGrantCommand{}

		Convey("Given a gateway with 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			obj1 := &model.LocalAgent{
				Name:        "object 1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(obj1), ShouldBeNil)

			obj2 := &model.RemoteAgent{
				Name:        "object 2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(obj2), ShouldBeNil)

			rule := &model.Rule{
				Name:    "existing rule",
				Comment: "comment about existing rule",
				IsSend:  false,
				Path:    "/existing/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)
			ruleID := fmt.Sprint(rule.ID)

			Convey("When adding a new rule access", func() {
				args := []string{ruleID}

				Convey("Given valid parameters", func() {
					command.ID = obj2.ID
					command.Type = fromTableName(obj2.TableName())

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute(args)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should display a message saying the rule "+
							"access was added", func() {
							So(getOutput(), ShouldEqual, "Access to rule 1 is "+
								"now restricted.\nAccess to rule n°1 "+
								"was successfully granted to remote agent n°1. "+
								"Granted accesses can be consulted at the address: "+
								gw.URL+admin.APIPath+admin.RulesPath+"/"+ruleID+
								admin.RulePermissionPath)
						})

						Convey("Then the new rule access should have been added", func() {
							acc := &model.RuleAccess{
								RuleID:     rule.ID,
								ObjectID:   obj2.ID,
								ObjectType: obj2.TableName(),
							}
							exists, err := db.Exists(acc)
							So(err, ShouldBeNil)
							So(exists, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the rule access already exist", func() {
					existing := &model.RuleAccess{
						RuleID:     rule.ID,
						ObjectID:   obj1.ID,
						ObjectType: obj1.TableName(),
					}
					So(db.Create(existing), ShouldBeNil)

					command.ID = obj1.ID
					command.Type = fromTableName(obj1.TableName())

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute(args)

						Convey("Then it should return an error", func() {
							So(err, ShouldNotBeNil)
						})

						Convey("Then is should display a message saying the access"+
							" already exist", func() {
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"The agent has already been granted access to this rule")
						})

						Convey("Then the old access should still exist", func() {
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

func TestRevokeRuleAccess(t *testing.T) {

	Convey("Testing the rule access 'revoke' command", t, func() {
		out = testFile()
		command := &ruleAccessRevokeCommand{}

		Convey("Given a gateway with 1 rule access", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			rule := &model.Rule{
				Name:    "existing rule",
				Comment: "comment about existing rule",
				IsSend:  false,
				Path:    "/existing/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			obj := &model.LocalAgent{
				Name:        "object",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(obj), ShouldBeNil)

			acc := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   obj.ID,
				ObjectType: obj.TableName(),
			}
			So(db.Create(acc), ShouldBeNil)

			Convey("Given a valid rule ID", func() {
				ruleID := fmt.Sprint(rule.ID)

				Convey("When executing the command", func() {
					command.ID = acc.ObjectID
					command.Type = fromTableName(acc.ObjectType)

					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{ruleID})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the rule was deleted", func() {
						So(getOutput(), ShouldEqual, "Access to rule "+ruleID+
							" is now unrestricted.\nAccess to rule n°"+ruleID+
							" was successfully revoked from local agent n°1.")
					})

					Convey("Then the access should have been removed", func() {
						exists, err := db.Exists(acc)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})
				})
			})

			Convey("Given an invalid rule ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					command.ID = acc.ObjectID
					command.Type = fromTableName(acc.ObjectType)

					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+admin.RulesPath+"/1000"+
							admin.RulePermissionPath+"' does not exist")
					})

					Convey("Then the access should still exist", func() {
						exists, err := db.Exists(acc)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestListRuleAccesses(t *testing.T) {

	Convey("Testing the rule 'list' command", t, func() {
		out = testFile()
		command := &ruleAccessListCommand{}
		_, err := flags.ParseArgs(command, []string{"waarp_gateway"})
		So(err, ShouldBeNil)

		Convey("Given a gateway with 2 rule accesses", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			obj1 := &model.LocalAgent{
				Name:        "object 1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(obj1), ShouldBeNil)

			obj2 := &model.RemoteAgent{
				Name:        "object 2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(obj2), ShouldBeNil)

			rule := &model.Rule{
				Name:    "rule",
				Comment: "rule comment",
				IsSend:  true,
				Path:    "/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			acc1 := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   obj1.ID,
				ObjectType: obj1.TableName(),
			}
			So(db.Create(acc1), ShouldBeNil)

			acc2 := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   obj2.ID,
				ObjectType: obj2.TableName(),
			}
			So(db.Create(acc2), ShouldBeNil)

			Convey("Given a valid rule ID parameter", func() {
				ruleID := fmt.Sprint(rule.ID)

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{ruleID})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the rule accesses", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Permissions:\n"+
							"Access to local agent n°1\n"+
							"Access to remote agent n°1\n",
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
							addr+admin.APIPath+admin.RulesPath+"/1000"+
							admin.RulePermissionPath+"' does not exist")
					})
				})
			})
		})
	})
}
