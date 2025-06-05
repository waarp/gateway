package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
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
		db := database.TestDatabase(c)

		Convey("Given a rule entry", func() {
			r := Rule{
				Name:   "rulename",
				Path:   "/test_path",
				IsSend: true,
			}
			So(db.Insert(&r).Run(), ShouldBeNil)

			rAgent := RemoteAgent{
				Name: "partner", Protocol: testProtocol,
				Address: types.Addr("localhost", 1111),
			}
			So(db.Insert(&rAgent).Run(), ShouldBeNil)

			rAccount := RemoteAccount{
				RemoteAgentID: rAgent.ID,
				Login:         "toto",
			}
			So(db.Insert(&rAccount).Run(), ShouldBeNil)

			lAgent := LocalAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 2222),
			}
			So(db.Insert(&lAgent).Run(), ShouldBeNil)

			lAccount := LocalAccount{
				LocalAgentID: lAgent.ID,
				Login:        "titi",
			}
			So(db.Insert(&lAccount).Run(), ShouldBeNil)

			Convey("Given a local_agent authorized for the rule", func() {
				lAccess := RuleAccess{
					RuleID:       r.ID,
					LocalAgentID: utils.NewNullInt64(lAgent.ID),
				}
				So(db.Insert(&lAccess).Run(), ShouldBeNil)

				Convey("Given a non authorized transfer", func() {
					t := &Transfer{
						RuleID:          r.ID,
						RemoteAccountID: utils.NewNullInt64(rAccount.ID),
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
		db := database.TestDatabase(c)

		Convey("Given a rule entry", func() {
			r := &Rule{
				Name:   "rulename",
				IsSend: true,
				Path:   "path",
			}
			So(db.Insert(r).Run(), ShouldBeNil)

			rAgent := RemoteAgent{
				Name: "partner", Protocol: testProtocol,
				Address: types.Addr("localhost", 1111),
			}
			So(db.Insert(&rAgent).Run(), ShouldBeNil)

			rAccount := RemoteAccount{
				RemoteAgentID: rAgent.ID,
				Login:         "toto",
			}
			So(db.Insert(&rAccount).Run(), ShouldBeNil)

			lAgent := LocalAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 2222),
			}
			So(db.Insert(&lAgent).Run(), ShouldBeNil)

			lAccount := LocalAccount{
				LocalAgentID: lAgent.ID,
				Login:        "titi",
			}
			So(db.Insert(&lAccount).Run(), ShouldBeNil)

			Convey("Given a valid rule and target", func() {
				ra := &RuleAccess{
					RuleID:       r.ID,
					LocalAgentID: utils.NewNullInt64(lAgent.ID),
				}

				Convey("When calling the `BeforeWrite` method", func() {
					err := ra.BeforeWrite(db)

					Convey("Then it should return NO error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given a RuleAccess with an invalid RuleID", func() {
				ra := &RuleAccess{
					RuleID:       1000,
					LocalAgentID: utils.NewNullInt64(lAgent.ID),
				}

				Convey("When calling the `BeforeWrite` method", func() {
					err := ra.BeforeWrite(db)

					Convey("Then the error should say 'No rule found'", func() {
						So(err, ShouldBeError, database.NewValidationErrorf(
							"no rule found with ID %d", ra.RuleID))
					})
				})
			})

			Convey("Given a RuleAccess with no target", func() {
				ra := &RuleAccess{
					RuleID: r.ID,
				}

				Convey("When calling the `BeforeWrite` method", func() {
					err := ra.BeforeWrite(db)

					Convey("Then the error should say that the access has no target", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"the rule access is missing a target"))
					})
				})
			})

			Convey("Given a RuleAccess with multiple targets", func() {
				ra := &RuleAccess{
					RuleID:        r.ID,
					LocalAgentID:  utils.NewNullInt64(lAgent.ID),
					RemoteAgentID: utils.NewNullInt64(rAgent.ID),
				}

				Convey("When calling the `BeforeWrite` method", func() {
					err := ra.BeforeWrite(db)

					Convey("Then the error should say that the access cannot have multiple targets", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"the rule access cannot have multiple targets"))
					})
				})
			})
		})
	})
}
