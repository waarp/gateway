package model

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func TestRuleAccessTableName(t *testing.T) {
	Convey("Given a RuleAccess instance", t, func() {
		ruleAccess := &RuleAccess{}

		Convey("When calling the `TableName` method", func() {
			name := ruleAccess.TableName()

			Convey("Then it should return the name of the rule_access table", func() {
				So(name, ShouldEqual, TableRuleAccesses)
			})
		})
	})
}

func TestIsRuleAuthorized(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a rule entry", func() {
			r := Rule{
				Name:   "rulename",
				Path:   "/test_path",
				IsSend: true,
			}
			So(db.Insert(&r).Run(), ShouldBeNil)

			rAgent := RemoteAgent{
				Name:     "partner",
				Protocol: testProtocol,
				Address:  "localhost:1111",
			}
			So(db.Insert(&rAgent).Run(), ShouldBeNil)

			rAccount := RemoteAccount{
				RemoteAgentID: rAgent.ID,
				Login:         "toto",
				Password:      "sesame",
			}
			So(db.Insert(&rAccount).Run(), ShouldBeNil)

			lAgent := LocalAgent{
				Name:     "server",
				Protocol: testProtocol,
				Address:  "localhost:2222",
			}
			So(db.Insert(&lAgent).Run(), ShouldBeNil)

			lAccount := LocalAccount{
				LocalAgentID: lAgent.ID,
				Login:        "titi",
				PasswordHash: hash("sesame"),
			}
			So(db.Insert(&lAccount).Run(), ShouldBeNil)

			Convey("Given a local_agent authorized for the rule", func() {
				lAccess := RuleAccess{
					RuleID:     r.ID,
					ObjectType: TableLocAgents,
					ObjectID:   lAgent.ID,
				}
				So(db.Insert(&lAccess).Run(), ShouldBeNil)

				Convey("Given a non authorized transfer", func() {
					t := &Transfer{
						IsServer:  false,
						RuleID:    r.ID,
						AgentID:   rAgent.ID,
						AccountID: rAccount.ID,
					}

					Convey("When calling the `IsRuleAuthorized` method", func() {
						auth, err := IsRuleAuthorized(db, t)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should not authorized", func() {
							So(auth, ShouldBeFalse)
						})
					})
				})
			})
		})
	})
}

func TestRuleAccessBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a rule entry", func() {
			r := &Rule{
				Name:   "rulename",
				IsSend: true,
				Path:   "path",
			}
			So(db.Insert(r).Run(), ShouldBeNil)

			rAgent := RemoteAgent{
				Name:     "partner",
				Protocol: testProtocol,
				Address:  "localhost:1111",
			}
			So(db.Insert(&rAgent).Run(), ShouldBeNil)

			rAccount := RemoteAccount{
				RemoteAgentID: rAgent.ID,
				Login:         "toto",
				Password:      "sesame",
			}
			So(db.Insert(&rAccount).Run(), ShouldBeNil)

			lAgent := LocalAgent{
				Name:     "server",
				Protocol: testProtocol,
				Address:  "localhost:2222",
			}
			So(db.Insert(&lAgent).Run(), ShouldBeNil)

			lAccount := LocalAccount{
				LocalAgentID: lAgent.ID,
				Login:        "titi",
				PasswordHash: hash("sesame"),
			}
			So(db.Insert(&lAccount).Run(), ShouldBeNil)

			Convey("Given a RuleAccess with an invalid RuleID", func() {
				ra := &RuleAccess{
					RuleID: 1000,
				}

				Convey("When calling the `BeforeWrite` method", func() {
					err := ra.BeforeWrite(db)

					Convey("Then the error should say 'No rule found'", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"no rule found with ID %d", ra.RuleID))
					})
				})
			})

			Convey("Given a RuleAccess with an invalid ObjectType", func() {
				ra := &RuleAccess{
					RuleID:     r.ID,
					ObjectType: "dummy",
				}

				Convey("When calling the `BeforeWrite` method", func() {
					err := ra.BeforeWrite(db)

					Convey("Then the error should say 'No rule found'", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"the rule_access's object type must be one of %s",
							validOwnerTypes))
					})
				})
			})

			for _, objType := range []string{
				TableLocAgents, TableLocAccounts,
				TableRemAgents, TableRemAccounts,
			} {
				Convey(fmt.Sprintf("Given a RuleAccess with an invalid %s ID", objType), func() {
					ra := &RuleAccess{
						RuleID:     r.ID,
						ObjectType: objType,
						ObjectID:   1000,
					}

					Convey("When calling the `BeforeWrite` method", func() {
						err := ra.BeforeWrite(db)

						Convey("Then the error should say 'No rule found'", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"no %s found with ID %d", ra.ObjectType, ra.ObjectID))
						})
					})
				})

				Convey(fmt.Sprintf("Given a RuleAccess with an valid %s ID", objType), func() {
					id := uint64(0)
					switch objType {
					case TableLocAgents:
						id = lAgent.ID
					case TableLocAccounts:
						id = lAccount.ID
					case TableRemAgents:
						id = rAgent.ID
					case TableRemAccounts:
						id = rAccount.ID
					}

					ra := &RuleAccess{
						RuleID:     r.ID,
						ObjectType: objType,
						ObjectID:   id,
					}

					Convey("When calling the `BeforeWrite` method", func() {
						err := ra.BeforeWrite(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			}
		})
	})
}
