package backup

import (
	"encoding/json"
	"fmt"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExportRules(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains 3 rules", func() {
			rule1 := &model.Rule{
				Name:   "test1",
				IsSend: true,
				Path:   "test1/send",
			}
			So(db.Insert(rule1).Run(), ShouldBeNil)

			rule2 := &model.Rule{
				Name:   "test2",
				IsSend: false,
				Path:   "test2",
			}
			So(db.Insert(rule2).Run(), ShouldBeNil)

			rule1b := &model.Rule{
				Name:   "test1",
				IsSend: false,
				Path:   "test1/recv",
			}
			So(db.Insert(rule1b).Run(), ShouldBeNil)

			Convey("When calling the exportRule function", func() {
				res, err := exportRules(discard, db)

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 3 records", func() {
					So(len(res), ShouldEqual, 3)
				})
			})
		})
	})
}

func TestExportRuleAccesses(t *testing.T) {

	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a rules with accesses", func() {
			agent := &model.RemoteAgent{
				Name:        "agent",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account1 := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "test",
				Password:      "sesame",
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			account2 := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "test2",
				Password:      "sesame",
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			rule1 := &model.Rule{
				Name:   "test1",
				IsSend: true,
				Path:   "test1/send",
			}
			So(db.Insert(rule1).Run(), ShouldBeNil)

			access1 := &model.RuleAccess{
				RuleID:     rule1.ID,
				ObjectType: model.TableRemAgents,
				ObjectID:   agent.ID,
			}
			So(db.Insert(access1).Run(), ShouldBeNil)

			rule2 := &model.Rule{
				Name:   "test2",
				IsSend: false,
				Path:   "test2/path",
			}
			So(db.Insert(rule2).Run(), ShouldBeNil)

			access2 := &model.RuleAccess{
				RuleID:     rule2.ID,
				ObjectType: model.TableRemAccounts,
				ObjectID:   account1.ID,
			}
			So(db.Insert(access2).Run(), ShouldBeNil)

			access3 := &model.RuleAccess{
				RuleID:     rule2.ID,
				ObjectType: model.TableRemAccounts,
				ObjectID:   account2.ID,
			}
			So(db.Insert(access3).Run(), ShouldBeNil)

			Convey("When calling the exportRuleAccesses function for rule1", func() {
				res, err := exportRuleAccesses(db, rule1.ID)

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 1 records", func() {
					So(len(res), ShouldEqual, 1)

					Convey("Then the result should correspond to the access of rule1", func() {
						value := fmt.Sprintf("remote::%s", agent.Name)
						So(res[0], ShouldEqual, value)
					})
				})
			})

			Convey("When calling the exportRuleAccesses function for ruler2", func() {
				res, err := exportRuleAccesses(db, rule2.ID)

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 2 records", func() {
					So(len(res), ShouldEqual, 2)
				})
			})
		})
	})
}

func TestExportRuleTasks(t *testing.T) {

	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given rules with tasks", func() {
			rule1 := &model.Rule{
				Name:   "test1",
				IsSend: true,
				Path:   "test1/send",
			}
			So(db.Insert(rule1).Run(), ShouldBeNil)

			rule2 := &model.Rule{
				Name:   "test2",
				IsSend: true,
				Path:   "test2/send",
			}
			So(db.Insert(rule2).Run(), ShouldBeNil)

			pre1 := &model.Task{
				RuleID: rule1.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Insert(pre1).Run(), ShouldBeNil)

			post1 := &model.Task{
				RuleID: rule1.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(post1).Run(), ShouldBeNil)

			post2 := &model.Task{
				RuleID: rule1.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"post2"}`),
			}
			So(db.Insert(post2).Run(), ShouldBeNil)

			error1 := &model.Task{
				RuleID: rule2.ID,
				Chain:  model.ChainError,
				Rank:   0,
				Type:   "MOVE",
				Args:   json.RawMessage(`{"path":"error1"}`),
			}
			So(db.Insert(error1).Run(), ShouldBeNil)

			Convey("When calling the exportRuleTasks function for chain PRE of rule1", func() {
				res, err := exportRuleTasks(db, rule1.ID, "PRE")

				Convey("Then it should return NO error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 1 result", func() {
					So(len(res), ShouldEqual, 1)

					Convey("Then this result should correspond to the pre1 Task", func() {
						So(res[0].Type, ShouldEqual, pre1.Type)
						So(res[0].Args, ShouldResemble, pre1.Args)
					})
				})
			})

			Convey("When calling the exportRuleTasks function for chain POST of rule1", func() {
				res, err := exportRuleTasks(db, rule1.ID, "POST")

				Convey("Then it should return NO error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 2 result", func() {
					So(len(res), ShouldEqual, 2)

					Convey("Then this result should correspond to the pre1 Task", func() {
						So(res[0].Type, ShouldEqual, post1.Type)
						So(res[0].Args, ShouldResemble, post1.Args)
						So(res[1].Type, ShouldEqual, post2.Type)
						So(res[1].Args, ShouldResemble, post2.Args)
					})
				})
			})

			Convey("When calling the exportRuleTasks function for chain ERROR of rule1", func() {
				res, err := exportRuleTasks(db, rule1.ID, "ERROR")

				Convey("Then it should return NO error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 0 result", func() {
					So(len(res), ShouldEqual, 0)
				})
			})

			Convey("When calling the exportRuleTasks function for chain ERROR of rule2", func() {
				res, err := exportRuleTasks(db, rule2.ID, "ERROR")

				Convey("Then it should return NO error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 1 result", func() {
					So(len(res), ShouldEqual, 1)

					Convey("Then this result should correspond to the pre1 Task", func() {
						So(res[0].Type, ShouldEqual, error1.Type)
						So(res[0].Args, ShouldResemble, error1.Args)
					})
				})
			})
		})
	})
}
