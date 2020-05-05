package model

import (
	"fmt"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	. "github.com/smartystreets/goconvey/convey"
)

type dummyProtocol struct{}

func (d *dummyProtocol) ValidServer() error  { return nil }
func (d *dummyProtocol) ValidPartner() error { return nil }

func init() {
	config.ProtoConfigs["dummy"] = func() config.ProtoConfig { return &dummyProtocol{} }
}

func TestRuleTableName(t *testing.T) {
	Convey("Given a `rule` instance", t, func() {
		rule := &Rule{}

		Convey("When calling the 'TableName' method", func() {
			name := rule.TableName()

			Convey("Then it should return the name of the rule table", func() {
				So(name, ShouldEqual, "rules")
			})
		})

	})
}

func TestRuleBeforeInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a rule entry", func() {
			old := &Rule{
				Name:   "old",
				IsSend: true,
				Path:   "/old_path",
			}
			So(db.Create(old), ShouldBeNil)

			Convey("Given a rule with a different a name", func() {
				rule := &Rule{
					Name:   "rule",
					IsSend: true,
					Path:   "/path",
				}

				Convey("When calling `BeforeUpdate`", func() {
					err := rule.BeforeInsert(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given a rule with the same name but with different send", func() {
				rule := &Rule{
					Name:   old.Name,
					IsSend: !old.IsSend,
					Path:   "/path",
				}

				Convey("When calling `BeforeUpdate`", func() {
					err := rule.BeforeInsert(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given a rule with the same name and same send", func() {
				rule := &Rule{
					Name:   old.Name,
					IsSend: old.IsSend,
					Path:   "/path",
				}

				Convey("When calling `BeforeUpdate`", func() {
					err := rule.BeforeInsert(db)

					Convey("Then the error should say that rule already exist", func() {
						So(err, ShouldBeError, fmt.Sprintf("a rule named '%s' "+
							"with send = %t already exist", old.Name, old.IsSend))
					})
				})
			})

			Convey("Given a rule without a path", func() {
				rule := &Rule{
					Name:   "rule",
					IsSend: false,
				}

				Convey("When calling `BeforeUpdate`", func() {
					err := rule.BeforeInsert(db)

					Convey("Then it should return an error saying that the path cannot be empty", func() {
						So(err, ShouldBeError, "the rule's path cannot be empty")
					})
				})
			})
		})
	})
}

func TestRuleBeforeUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given two rule entry", func() {
			rule1 := &Rule{
				Name:   "rule1",
				IsSend: true,
				Path:   "/path",
			}
			So(db.Create(rule1), ShouldBeNil)

			rule2 := &Rule{
				Name:   "rule2",
				IsSend: true,
				Path:   "/path2",
			}
			So(db.Create(rule2), ShouldBeNil)

			Convey("When updating with invalid data", func() {
				update := &Rule{Name: rule2.Name, IsSend: rule2.IsSend}

				Convey("When calling the `BeforeUpdate` function", func() {
					err := update.BeforeUpdate(db, rule1.ID)

					Convey("Then it should return an error say the name is taken", func() {
						So(err, ShouldBeError, fmt.Sprintf("a rule named '%s' "+
							"with send = %t already exist", update.Name, update.IsSend))
					})

				})
			})

			Convey("When updating with valid data", func() {
				update := &Rule{Name: "toto", IsSend: true}

				Convey("When calling the `BeforeUpdate` function", func() {
					err := update.BeforeUpdate(db, rule1.ID)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestRuleBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a rule with some tasks and permissions", func() {
			rule := &Rule{
				Name:   "rule",
				IsSend: true,
				Path:   "/path",
			}
			So(db.Create(rule), ShouldBeNil)

			t1 := &Task{RuleID: rule.ID, Chain: ChainPre, Rank: 0, Type: "TESTSUCCESS", Args: []byte(`{}`)}
			So(db.Create(t1), ShouldBeNil)
			t2 := &Task{RuleID: rule.ID, Chain: ChainPost, Rank: 0, Type: "TESTSUCCESS", Args: []byte(`{}`)}
			So(db.Create(t2), ShouldBeNil)

			server := &LocalAgent{Name: "server", Protocol: "dummy", ProtoConfig: []byte(`{}`)}
			So(db.Create(server), ShouldBeNil)
			account := &LocalAccount{LocalAgentID: server.ID, Login: "toto", Password: []byte("password")}
			So(db.Create(account), ShouldBeNil)

			a1 := &RuleAccess{RuleID: rule.ID, ObjectID: server.ID, ObjectType: server.TableName()}
			So(db.Create(a1), ShouldBeNil)
			a2 := &RuleAccess{RuleID: rule.ID, ObjectID: account.ID, ObjectType: account.TableName()}
			So(db.Create(a2), ShouldBeNil)

			Convey("Given that the rule is unused", func() {

				Convey("When calling the `BeforeDelete` function", func() {
					err := rule.BeforeDelete(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the rule is used by a transfer", func() {
				trans := &Transfer{
					RuleID:     rule.ID,
					IsServer:   true,
					AgentID:    server.ID,
					AccountID:  account.ID,
					SourceFile: "file.src",
					DestFile:   "file.dst",
				}
				So(db.Create(trans), ShouldBeNil)

				Convey("When calling the `BeforeDelete` function", func() {
					err := rule.BeforeDelete(db)

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
