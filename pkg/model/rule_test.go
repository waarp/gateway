package model

import (
	"path"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestRuleTableName(t *testing.T) {
	Convey("Given a `rule` instance", t, func() {
		rule := &Rule{}

		Convey("When calling the 'TableName' method", func() {
			name := rule.TableName()

			Convey("Then it should return the name of the rule table", func() {
				So(name, ShouldEqual, TableRules)
			})
		})
	})
}

func TestRuleBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a rule entry", func() {
			old := Rule{
				Name:   "old",
				IsSend: true,
				Path:   "old_path/subdir",
			}
			So(db.Insert(&old).Run(), ShouldBeNil)

			rule := Rule{
				Name:   "new",
				IsSend: true,
				Path:   "new_path",
			}

			shouldSucceed := func() {
				Convey("When calling `BeforeWrite`", func() {
					err := db.Transaction(func(ses *database.Session) error {
						return rule.BeforeWrite(ses)
					})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			}

			shouldFailWith := func(errDesc string, expErr error) {
				Convey("When calling `BeforeWrite`", func() {
					err := db.Transaction(func(ses *database.Session) error {
						return rule.BeforeWrite(ses)
					})

					Convey("Then the error should say that "+errDesc, func() {
						So(err, ShouldBeError, expErr)
					})
				})
			}

			Convey("Given a rule with a different a name", func() {
				shouldSucceed()
			})

			Convey("Given a rule with the same name but with different send", func() {
				rule.Name = old.Name
				rule.IsSend = !old.IsSend

				shouldSucceed()
			})

			Convey("Given a rule with the same name and same direction", func() {
				rule.Name = old.Name
				rule.IsSend = old.IsSend

				shouldFailWith("the rule already exist", database.NewValidationError(
					"a %s rule named %q already exist", rule.Direction(), rule.Name))
			})

			Convey("Given a rule with a path ancestor to this rule's path", func() {
				rule.Path = path.Join(old.Path, rule.Path)

				shouldFailWith("the path cannot be a descendant", database.NewValidationError(
					"the rule's path cannot be the descendant of another rule's path "+
						"(the path %q is already used by rule %q)", old.Path, old.Name))
			})

			Convey("Given a rule with a path descendant to this rule's path", func() {
				rule.Path = path.Dir(old.Path)

				shouldFailWith("the path cannot be an ancestor", database.NewValidationError(
					"the rule's path cannot be the ancestor of another rule's path"))
			})

			Convey("Given a rule without a path", func() {
				rule.Path = ""

				Convey("When calling `BeforeWrite`", func() {
					err := db.Transaction(func(ses *database.Session) error {
						return rule.BeforeWrite(ses)
					})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)

						Convey("Then the path should have been filled", func() {
							So(rule.Path, ShouldEqual, rule.Name)
						})
					})
				})
			})
		})
	})
}

func TestRuleBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a rule with some tasks and permissions", func() {
			rule := Rule{
				Name:   "rule",
				IsSend: true,
				Path:   "path",
			}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			t1 := Task{RuleID: rule.ID, Chain: ChainPre, Rank: 0, Type: "TESTSUCCESS"}
			So(db.Insert(&t1).Run(), ShouldBeNil)

			t2 := Task{RuleID: rule.ID, Chain: ChainPost, Rank: 0, Type: "TESTSUCCESS"}
			So(db.Insert(&t2).Run(), ShouldBeNil)

			server := LocalAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 1111),
			}
			So(db.Insert(&server).Run(), ShouldBeNil)
			account := LocalAccount{LocalAgentID: server.ID, Login: "toto"}
			So(db.Insert(&account).Run(), ShouldBeNil)

			a1 := RuleAccess{RuleID: rule.ID, LocalAgentID: utils.NewNullInt64(server.ID)}
			So(db.Insert(&a1).Run(), ShouldBeNil)

			a2 := RuleAccess{RuleID: rule.ID, LocalAccountID: utils.NewNullInt64(account.ID)}
			So(db.Insert(&a2).Run(), ShouldBeNil)

			Convey("Given that the rule is unused", func() {
				Convey("When calling the `BeforeDelete` function", func() {
					err := db.Transaction(func(ses *database.Session) error {
						return rule.BeforeDelete(ses)
					})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the rule is used by a transfer", func() {
				trans := Transfer{
					RuleID:         rule.ID,
					LocalAccountID: utils.NewNullInt64(account.ID),
					SrcFilename:    "file",
				}
				So(db.Insert(&trans).Run(), ShouldBeNil)

				Convey("When calling the `BeforeDelete` function", func() {
					err := db.Transaction(func(ses *database.Session) error {
						return rule.BeforeDelete(ses)
					})

					Convey("Then the error should say that the rule cannot be deleted", func() {
						So(err, ShouldBeError, "this rule is currently being "+
							"used in a running transfer and cannot be deleted, "+
							"cancel the transfer or wait for it to finish")
					})
				})
			})
		})
	})
}
