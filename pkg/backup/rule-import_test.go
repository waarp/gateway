package backup

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestImportRules(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some Rules", func() {
			insert := &model.Rule{
				Name:   "rule_insert",
				IsSend: true,
				Path:   "path/to/Rule",
			}
			So(db.Insert(insert).Run(), ShouldBeNil)

			pre1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Insert(pre1).Run(), ShouldBeNil)

			pre2 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(pre2).Run(), ShouldBeNil)

			post1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Insert(post1).Run(), ShouldBeNil)

			post2 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(post2).Run(), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "server",
				Protocol:    testProtocol,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "account1",
				PasswordHash: hash("pwd"),
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "account2",
				PasswordHash: hash("pwd"),
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			Convey("Given a new Rule to import", func() {
				Rule1 := file.Rule{
					Name:   "foo",
					IsSend: true,
					Path:   "test/path",
					Accesses: []string{
						"local::server",
						"local::server::account1",
						"local::server::account2",
					},
					Pre: []file.Task{
						{
							Type: "COPY",
							Args: []byte(`{"path":"copy/destination"}`),
						},
					},
					Post: []file.Task{
						{
							Type: "DELETE",
							Args: []byte("{}"),
						},
					},
					Error: []file.Task{
						{
							Type: "COPY",
							Args: []byte(`{"path":"copy/destination"}`),
						},
						{
							Type: "DELETE",
							Args: []byte("{}"),
						},
					},
				}
				Rules := []file.Rule{Rule1}

				Convey("When calling importRules with the new Rules", func() {
					err := importRules(discard, db, Rules)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contains the Rule "+
						"imported", func() {
						var dbRule model.Rule
						So(db.Get(&dbRule, "name=? AND send=?", Rule1.Name,
							Rule1.IsSend).Run(), ShouldBeNil)

						Convey("Then the record should correspond to "+
							"the data imported", func() {
							So(dbRule.Path, ShouldEqual, Rule1.Path)

							var auths model.RuleAccesses
							So(db.Select(&auths).Where("rule_id=?", dbRule.ID).
								Run(), ShouldBeNil)
							So(len(auths), ShouldEqual, 3)

							var pres model.Tasks
							So(db.Select(&pres).Where("rule_id=? AND chain='PRE'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(pres), ShouldEqual, 1)

							var posts model.Tasks
							So(db.Select(&posts).Where("rule_id=? AND chain='POST'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(posts), ShouldEqual, 1)

							var errors model.Tasks
							So(db.Select(&errors).Where("rule_id= ? AND chain='ERROR'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(errors), ShouldEqual, 2)
						})
					})
				})
			})

			Convey("Given a existing Rule to fully updated", func() {
				Rule1 := file.Rule{
					Name:   insert.Name,
					IsSend: insert.IsSend,
					Path:   "testing",
					Accesses: []string{
						"local::server",
						"local::server::account2",
					},
					Pre: []file.Task{
						{
							Type: "COPY",
							Args: []byte(`{"path":"copy/destination"}`),
						},
					},
					Post: []file.Task{
						{
							Type: "DELETE",
							Args: []byte("{}"),
						},
					},
					Error: []file.Task{
						{
							Type: "COPY",
							Args: []byte(`{"path":"copy/destination"}`),
						},
						{
							Type: "DELETE",
							Args: []byte("{}"),
						},
					},
				}
				Rules := []file.Rule{Rule1}

				Convey("When calling importRules with the new Rules", func() {
					err := importRules(discard, db, Rules)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contains the Rule imported", func() {
						var dbRule model.Rule
						So(db.Get(&dbRule, "name=? AND send=?", insert.Name,
							insert.IsSend).Run(), ShouldBeNil)

						Convey("Then the record should correspond to "+
							"the data imported", func() {
							So(dbRule.Path, ShouldEqual, Rule1.Path)

							var auths model.RuleAccesses
							So(db.Select(&auths).Where("rule_id=?", dbRule.ID).
								Run(), ShouldBeNil)
							So(len(auths), ShouldEqual, 2)

							var pres model.Tasks
							So(db.Select(&pres).Where("rule_id=? AND chain='PRE'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(pres), ShouldEqual, 1)

							var posts model.Tasks
							So(db.Select(&posts).Where("rule_id=? AND chain='POST'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(posts), ShouldEqual, 1)

							var errors model.Tasks
							So(db.Select(&errors).Where("rule_id=? AND chain='ERROR'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(errors), ShouldEqual, 2)
						})
					})
				})
			})

			Convey("Given a existing Rule to partially updated", func() {
				Rule1 := file.Rule{
					Name:   insert.Name,
					IsSend: insert.IsSend,
					Path:   "testing",
					Accesses: []string{
						"local::server",
						"local::server::account2",
					},
				}
				Rules := []file.Rule{Rule1}

				Convey("When calling importRules with the new Rules", func() {
					err := importRules(discard, db, Rules)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contains the "+
						"Rule imported", func() {
						var dbRule model.Rule
						So(db.Get(&dbRule, "name=? AND send=?", insert.Name,
							insert.IsSend).Run(), ShouldBeNil)

						Convey("Then the record should correspond to "+
							"the data imported", func() {
							So(dbRule.Path, ShouldEqual, Rule1.Path)

							var auths model.RuleAccesses
							So(db.Select(&auths).Where("rule_id=?", dbRule.ID).
								Run(), ShouldBeNil)
							So(len(auths), ShouldEqual, 2)

							var pres model.Tasks
							So(db.Select(&pres).Where("rule_id=? AND chain='PRE'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(pres), ShouldEqual, 2)

							var posts model.Tasks
							So(db.Select(&posts).Where("rule_id=? AND chain='POST'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(posts), ShouldEqual, 2)

							var errors model.Tasks
							So(db.Select(&errors).Where("rule_id=? AND chain='ERROR'",
								dbRule.ID).Run(), ShouldBeNil)
							So(len(errors), ShouldEqual, 0)
						})
					})
				})
			})
		})
	})
}

func TestImportRuleAccess(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some Rules", func() {
			insert := &model.Rule{
				Name:   "rule_insert",
				IsSend: true,
				Path:   "path/to/Rule",
			}
			So(db.Insert(insert).Run(), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "server",
				Protocol:    testProtocol,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "account1",
				PasswordHash: hash("pwd"),
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "account2",
				PasswordHash: hash("pwd"),
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			Convey("Given a new access to import", func() {
				accesses := []string{
					"local::server",
					"local::server::account1",
					"local::server::account2",
				}

				Convey("When calling importRuleAccesses with new", func() {
					err := importRuleAccesses(db, accesses, insert.ID)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contains 3 accesses", func() {
						var dbAccesses model.RuleAccesses
						So(db.Select(&dbAccesses).Where("rule_id=?", insert.ID).
							Run(), ShouldBeNil)
						So(len(dbAccesses), ShouldEqual, 3)

						Convey("Then the data should correspond to "+
							"the ones imported", func() {
							for i := 0; i < len(dbAccesses); i++ {
								acc := dbAccesses[i]
								switch {
								case acc.ObjectType == model.TableLocAgents &&
									acc.ObjectID == agent.ID:
									Convey("Then access for agent is found", func() {
									})
								case acc.ObjectType == model.TableLocAccounts &&
									acc.ObjectID == account1.ID:
									Convey("Then access for accunt1 is found", func() {
									})
								case acc.ObjectType == model.TableLocAccounts &&
									acc.ObjectID == account2.ID:
									Convey("Then access for accunt2 is found", func() {
									})
								default:
									Convey("Then they should be no "+
										"other records", func() {
										So(1, ShouldBeNil)
									})
								}
							}
						})
					})
				})
			})

			Convey("Given a Rule with 1 access", func() {
				acc1 := &model.RuleAccess{
					RuleID:     insert.ID,
					ObjectType: model.TableLocAgents,
					ObjectID:   agent.ID,
				}
				So(db.Insert(acc1).Run(), ShouldBeNil)

				Convey("Given a new access to import", func() {
					accesses := []string{
						"local::server::account1",
						"local::server::account2",
					}

					Convey("When calling importRuleAccesses with new", func() {
						err := importRuleAccesses(db, accesses, insert.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the database should contains 3 accesses", func() {
							var dbAccesses model.RuleAccesses
							So(db.Select(&dbAccesses).Where("rule_id=?", insert.ID).
								Run(), ShouldBeNil)
							So(len(dbAccesses), ShouldEqual, 3)

							Convey("Then the data should correspond to "+
								"the ones imported", func() {
								for i := 0; i < len(dbAccesses); i++ {
									acc := dbAccesses[i]
									switch {
									case acc.ObjectType == model.TableLocAgents &&
										acc.ObjectID == agent.ID:
										Convey("Then access for agent is found", func() {
										})
									case acc.ObjectType == model.TableLocAccounts &&
										acc.ObjectID == account1.ID:
										Convey("Then access for account1 is found", func() {
										})
									case acc.ObjectType == model.TableLocAccounts &&
										acc.ObjectID == account2.ID:
										Convey("Then access for account2 is found", func() {
										})
									default:
										Convey("Then they should be no other records", func() {
											So(1, ShouldBeNil)
										})
									}
								}
							})
						})
					})
				})
			})
		})
	})
}

func TestImportTasks(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some Rules", func() {
			insert := &model.Rule{
				Name:   "rule_insert",
				IsSend: true,
				Path:   "path/to/Rule",
			}
			So(db.Insert(insert).Run(), ShouldBeNil)

			pre1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Insert(pre1).Run(), ShouldBeNil)

			pre2 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(pre2).Run(), ShouldBeNil)

			post1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Insert(post1).Run(), ShouldBeNil)

			post2 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(post2).Run(), ShouldBeNil)

			error1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainError,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(error1).Run(), ShouldBeNil)

			Convey("Given some tasks to import", func() {
				tasks := []file.Task{
					{
						Type: "COPY",
						Args: []byte(`{"path":"copy/destination"}`),
					},
					{
						Type: "DELETE",
						Args: []byte("{}"),
					},
				}

				Convey("When calling importTasks on pre tasks", func() {
					err := importRuleTasks(discard, db, tasks, insert.ID, model.ChainPre)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contains 2 tasks", func() {
						var dbTasks model.Tasks
						So(db.Select(&dbTasks).Where("rule_id=? AND chain='PRE'",
							insert.ID).Run(), ShouldBeNil)
						So(len(dbTasks), ShouldEqual, 2)

						Convey("Then the data should correspond to the ones imported", func() {
							for i := 0; i < len(dbTasks); i++ {
								So(dbTasks[i].Type, ShouldEqual, tasks[i].Type)
								So(dbTasks[i].Args, ShouldResemble, tasks[i].Args)
							}
						})
					})
				})

				Convey("When calling importTasks on post tasks", func() {
					err := importRuleTasks(discard, db, tasks, insert.ID, model.ChainPost)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contains 2 tasks", func() {
						var dbTasks model.Tasks
						So(db.Select(&dbTasks).Where("rule_id=? AND chain='POST'",
							insert.ID).Run(), ShouldBeNil)
						So(len(dbTasks), ShouldEqual, 2)

						Convey("Then the data should correspond to the ones imported", func() {
							for i := 0; i < len(dbTasks); i++ {
								So(dbTasks[i].Type, ShouldEqual, tasks[i].Type)
								So(dbTasks[i].Args, ShouldResemble, tasks[i].Args)
							}
						})
					})
				})

				Convey("When calling importTasks on error tasks", func() {
					err := importRuleTasks(discard, db, tasks, insert.ID, model.ChainError)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contains 2 tasks", func() {
						var dbTasks model.Tasks
						So(db.Select(&dbTasks).Where("rule_id=? AND chain='ERROR'",
							insert.ID).Run(), ShouldBeNil)
						So(len(dbTasks), ShouldEqual, 2)

						Convey("Then the data should correspond to the ones imported", func() {
							for i := 0; i < len(dbTasks); i++ {
								So(dbTasks[i].Type, ShouldEqual, tasks[i].Type)
								So(dbTasks[i].Args, ShouldResemble, tasks[i].Args)
							}
						})
					})
				})
			})
		})
	})
}
