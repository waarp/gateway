package model

import (
	"encoding/json"
	"path"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
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
		db := database.TestDatabase(c, "ERROR")

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
					err := db.Transaction(func(ses *database.Session) database.Error {
						return rule.BeforeWrite(ses)
					})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			}

			shouldFailWith := func(errDesc string, expErr error) {
				Convey("When calling `BeforeWrite`", func() {
					err := db.Transaction(func(ses *database.Session) database.Error {
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
					"a %s rule named '%s' already exist", rule.Direction(), rule.Name))

			})

			Convey("Given a rule with a path ancestor to this rule's path", func() {
				rule.Path = path.Join(old.Path, rule.Path)
				shouldFailWith("the path cannot be a descendant", database.NewValidationError(
					"the rule's path cannot be the descendant of another rule's path "+
						"(the path '%s' is already used by rule '%s')", old.Path, old.Name))
			})

			Convey("Given a rule with a path descendant to this rule's path", func() {
				rule.Path = path.Dir(old.Path)
				shouldFailWith("the path cannot be an ancestor", database.NewValidationError(
					"the rule's path cannot be the ancestor of another rule's path"))
			})

			Convey("Given a rule without a path", func() {
				rule.Path = ""

				Convey("When calling `BeforeWrite`", func() {
					err := db.Transaction(func(ses *database.Session) database.Error {
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
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a rule with some tasks and permissions", func() {
			rule := Rule{
				Name:   "rule",
				IsSend: true,
				Path:   "path",
			}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			t1 := Task{RuleID: rule.ID, Chain: ChainPre, Rank: 0, Type: "TESTSUCCESS", Args: []byte(`{}`)}
			So(db.Insert(&t1).Run(), ShouldBeNil)
			t2 := Task{RuleID: rule.ID, Chain: ChainPost, Rank: 0, Type: "TESTSUCCESS", Args: []byte(`{}`)}
			So(db.Insert(&t2).Run(), ShouldBeNil)

			server := LocalAgent{
				Name:        "server",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1111",
			}
			So(db.Insert(&server).Run(), ShouldBeNil)
			account := LocalAccount{LocalAgentID: server.ID, Login: "toto", PasswordHash: hash("sesame")}
			So(db.Insert(&account).Run(), ShouldBeNil)

			a1 := RuleAccess{RuleID: rule.ID, ObjectID: server.ID, ObjectType: server.TableName()}
			So(db.Insert(&a1).Run(), ShouldBeNil)
			a2 := RuleAccess{RuleID: rule.ID, ObjectID: account.ID, ObjectType: account.TableName()}
			So(db.Insert(&a2).Run(), ShouldBeNil)

			Convey("Given that the rule is unused", func() {

				Convey("When calling the `BeforeDelete` function", func() {
					err := db.Transaction(func(ses *database.Session) database.Error {
						return rule.BeforeDelete(ses)
					})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the rule is used by a transfer", func() {
				trans := Transfer{
					RuleID:     rule.ID,
					IsServer:   true,
					AgentID:    server.ID,
					AccountID:  account.ID,
					LocalPath:  "file.loc",
					RemotePath: "file.rem",
				}
				So(db.Insert(&trans).Run(), ShouldBeNil)

				Convey("When calling the `BeforeDelete` function", func() {
					err := db.Transaction(func(ses *database.Session) database.Error {
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
