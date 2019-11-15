package model

import (
	"fmt"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRuleAccessTableName(t *testing.T) {
	Convey("Given a RuleAccess instance", t, func() {
		ruleAccess := &RuleAccess{}

		Convey("When calling the `TableName` method", func() {
			name := ruleAccess.TableName()

			Convey("Then it should return the name of the rule_access table", func() {
				So(name, ShouldEqual, "rule_access")
			})
		})
	})
}

func TestRuleAccessValidateInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a rule entry", func() {
			r := &Rule{
				Name:   "Test",
				IsSend: true,
			}
			So(db.Create(r), ShouldBeNil)

			rAgent := &RemoteAgent{
				Name:        "Test",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/sftp/root"}`),
			}
			So(db.Create(rAgent), ShouldBeNil)

			rAccount := &RemoteAccount{
				RemoteAgentID: rAgent.ID,
				Login:         "Test",
				Password:      []byte("dummy"),
			}
			So(db.Create(rAccount), ShouldBeNil)

			lAgent := &LocalAgent{
				Name:        "Test",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/sftp/root"}`),
			}
			So(db.Create(lAgent), ShouldBeNil)

			lAccount := &LocalAccount{
				LocalAgentID: lAgent.ID,
				Login:        "Test",
				Password:     []byte("dummy"),
			}
			So(db.Create(lAccount), ShouldBeNil)

			Convey("Given a RuleAccess with an invalid RuleID", func() {
				ra := &RuleAccess{
					RuleID: 0,
				}

				Convey("When calling the `ValidateInsert` method", func() {
					err := ra.ValidateInsert(db)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then the error should say 'No rule found'", func() {
						So(err.Error(), ShouldEqual, fmt.Sprintf("No rule found with ID %d", 0))
					})
				})
			})

			Convey("Given a RuleAccess with an invalid ObjectType", func() {
				ra := &RuleAccess{
					RuleID:     r.ID,
					ObjectType: "dummy",
				}

				Convey("When calling the `ValidateInsert` method", func() {
					err := ra.ValidateInsert(db)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then the error should say 'No rule found'", func() {
						So(err.Error(), ShouldEqual,
							fmt.Sprintf("The rule_access's object type must be one of %s",
								"[local_agents remote_agents local_accounts remote_accounts]"))
					})
				})
			})

			for _, obj_type := range []string{"local_agents", "local_accounts", "remote_agents", "remote_accounts"} {
				Convey(fmt.Sprintf("Given a RuleAccess with an invalid %s ID", obj_type), func() {
					ra := &RuleAccess{
						RuleID:     r.ID,
						ObjectType: obj_type,
						ObjectID:   0,
					}

					Convey("When calling the `ValidateInsert` method", func() {
						err := ra.ValidateInsert(db)

						Convey("Then it should return an error", func() {
							So(err, ShouldNotBeNil)
						})

						Convey("Then the error should say 'No rule found'", func() {
							So(err.Error(), ShouldEqual,
								fmt.Sprintf("No %s found with ID '%d'", ra.ObjectType, ra.ObjectID))
						})
					})
				})

				Convey(fmt.Sprintf("Given a RuleAccess with an valid %s ID", obj_type), func() {
					id := uint64(0)
					switch obj_type {
					case "local_agents":
						id = lAgent.ID
					case "local_accounts":
						id = lAccount.ID
					case "remote_agents":
						id = rAgent.ID
					case "remote_accounts":
						id = rAccount.ID
					}

					ra := &RuleAccess{
						RuleID:     r.ID,
						ObjectType: obj_type,
						ObjectID:   id,
					}

					Convey("When calling the `ValidateInsert` method", func() {
						err := ra.ValidateInsert(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			}
		})
	})
}

func TestRuleAccessValidateUpdate(t *testing.T) {
	Convey("Given a RuleAccess instance", t, func() {
		ruleAccess := &RuleAccess{}

		Convey("When calling the `ValidateUpdate` method", func() {
			err := ruleAccess.ValidateUpdate(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then the error should say that operation is unallowed", func() {
				So(err.Error(), ShouldEqual, "Unallowed operation")
			})
		})
	})
}
