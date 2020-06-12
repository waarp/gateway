package backup

import (
	"encoding/json"
	"testing"

	"github.com/go-xorm/builder"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func TestImportRules(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some rules", func() {
			insert := &model.Rule{
				Name:   "test",
				IsSend: true,
				Path:   "path/to/rule",
			}
			So(db.Create(insert), ShouldBeNil)

			pre1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Create(pre1), ShouldBeNil)

			pre2 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(pre2), ShouldBeNil)

			post1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Create(post1), ShouldBeNil)

			post2 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(post2), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(agent), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "foo",
				Password:     []byte("pwd"),
			}
			So(db.Create(account1), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "test",
				Password:     []byte("pwd"),
			}
			So(db.Create(account2), ShouldBeNil)

			Convey("Given a new rule to import", func() {
				rule1 := rule{
					Name:   "foo",
					IsSend: true,
					Path:   "test/path",
					Accesses: []string{
						"local::test",
						"local::test::foo",
						"local::test::test",
					},
					Pre: []ruleTask{
						{
							Type: "COPY",
							Args: []byte(`{"path":"copy/destination"}`),
						},
					},
					Post: []ruleTask{
						{
							Type: "DELETE",
							Args: []byte("{}"),
						},
					},
					Error: []ruleTask{
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
				rules := []rule{rule1}

				Convey("Given a new transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling importRules with the new rules", func() {
						err := importRules(ses, rules)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the database should contains the rule imported", func() {
							dbRule := &model.Rule{
								Name:   rule1.Name,
								IsSend: rule1.IsSend,
							}
							So(ses.Get(dbRule), ShouldBeNil)

							Convey("Then the record should correspond to the data imported", func() {
								So(dbRule.Path, ShouldEqual, rule1.Path)

								auths := []model.RuleAccess{}
								So(ses.Select(&auths, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID},
								}), ShouldBeNil)
								So(len(auths), ShouldEqual, 3)

								pres := []model.Task{}
								So(ses.Select(&pres, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "PRE"},
								}), ShouldBeNil)
								So(len(pres), ShouldEqual, 1)

								posts := []model.Task{}
								So(ses.Select(&posts, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "POST"},
								}), ShouldBeNil)
								So(len(posts), ShouldEqual, 1)

								errors := []model.Task{}
								So(ses.Select(&errors, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "ERROR"},
								}), ShouldBeNil)
								So(len(errors), ShouldEqual, 2)
							})
						})
					})

				})
			})

			Convey("Given a existing rule to fully updated", func() {
				rule1 := rule{
					Name:   insert.Name,
					IsSend: insert.IsSend,
					Path:   "testing",
					Accesses: []string{
						"local::test",
						"local::test::test",
					},
					Pre: []ruleTask{
						{
							Type: "COPY",
							Args: []byte(`{"path":"copy/destination"}`),
						},
					},
					Post: []ruleTask{
						{
							Type: "DELETE",
							Args: []byte("{}"),
						},
					},
					Error: []ruleTask{
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
				rules := []rule{rule1}

				Convey("Given a new transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling importRules with the new rules", func() {
						err := importRules(ses, rules)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the database should contains the rule imported", func() {
							dbRule := &model.Rule{
								Name:   insert.Name,
								IsSend: insert.IsSend,
							}
							So(ses.Get(dbRule), ShouldBeNil)

							Convey("Then the record should correspond to the data imported", func() {
								So(dbRule.Path, ShouldEqual, rule1.Path)

								auths := []model.RuleAccess{}
								So(ses.Select(&auths, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID},
								}), ShouldBeNil)
								So(len(auths), ShouldEqual, 2)

								pres := []model.Task{}
								So(ses.Select(&pres, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "PRE"},
								}), ShouldBeNil)
								So(len(pres), ShouldEqual, 1)

								posts := []model.Task{}
								So(ses.Select(&posts, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "POST"},
								}), ShouldBeNil)
								So(len(posts), ShouldEqual, 1)

								errors := []model.Task{}
								So(ses.Select(&errors, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "ERROR"},
								}), ShouldBeNil)
								So(len(errors), ShouldEqual, 2)
							})
						})
					})

				})
			})

			Convey("Given a existing rule to partially updated", func() {
				rule1 := rule{
					Name:   insert.Name,
					IsSend: insert.IsSend,
					Path:   "testing",
					Accesses: []string{
						"local::test",
						"local::test::test",
					},
				}
				rules := []rule{rule1}

				Convey("Given a new transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling importRules with the new rules", func() {
						err := importRules(ses, rules)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the database should contains the rule imported", func() {
							dbRule := &model.Rule{
								Name:   insert.Name,
								IsSend: insert.IsSend,
							}
							So(ses.Get(dbRule), ShouldBeNil)

							Convey("Then the record should correspond to the data imported", func() {
								So(dbRule.Path, ShouldEqual, rule1.Path)

								auths := []model.RuleAccess{}
								So(ses.Select(&auths, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID},
								}), ShouldBeNil)
								So(len(auths), ShouldEqual, 2)

								pres := []model.Task{}
								So(ses.Select(&pres, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "PRE"},
								}), ShouldBeNil)
								So(len(pres), ShouldEqual, 2)

								posts := []model.Task{}
								So(ses.Select(&posts, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "POST"},
								}), ShouldBeNil)
								So(len(posts), ShouldEqual, 2)

								errors := []model.Task{}
								So(ses.Select(&errors, &database.Filters{
									Conditions: builder.Eq{"rule_id": dbRule.ID, "chain": "ERROR"},
								}), ShouldBeNil)
								So(len(errors), ShouldEqual, 0)
							})
						})
					})

				})
			})
		})

	})
}

func TestImportRuleAccess(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some rules", func() {
			insert := &model.Rule{
				Name:   "test",
				IsSend: true,
				Path:   "path/to/rule",
			}
			So(db.Create(insert), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(agent), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "foo",
				Password:     []byte("pwd"),
			}
			So(db.Create(account1), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "test",
				Password:     []byte("pwd"),
			}
			So(db.Create(account2), ShouldBeNil)

			Convey("Given a new access to import", func() {
				accesses := []string{
					"local::test",
					"local::test::foo",
					"local::test::test",
				}

				Convey("Given a new transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling importRuleAccesses with new", func() {
						err := importRuleAccesses(ses, accesses, insert.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the database should contains 3 accesses", func() {
							dbAccesses := []model.RuleAccess{}
							So(ses.Select(&dbAccesses, &database.Filters{
								Conditions: builder.Eq{"rule_id": insert.ID},
							}), ShouldBeNil)
							So(len(dbAccesses), ShouldEqual, 3)

							Convey("Then the data should correspond to the ones imported", func() {
								for i := 0; i < len(dbAccesses); i++ {
									acc := dbAccesses[i]
									if acc.ObjectType == "local_agents" && acc.ObjectID == agent.ID {
										Convey("Then access for agent is found", func() {
										})
									} else if acc.ObjectType == "local_accounts" && acc.ObjectID == account1.ID {
										Convey("Then access for accunt1 is found", func() {
										})
									} else if acc.ObjectType == "local_accounts" && acc.ObjectID == account2.ID {
										Convey("Then access for accunt2 is found", func() {
										})
									} else {
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

			Convey("Given a rule with 1 access", func() {
				acc1 := &model.RuleAccess{
					RuleID:     insert.ID,
					ObjectType: "local_agents",
					ObjectID:   agent.ID,
				}
				So(db.Create(acc1), ShouldBeNil)

				Convey("Given a new access to import", func() {
					accesses := []string{
						"local::test::foo",
						"local::test::test",
					}

					Convey("Given a new transaction", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						defer ses.Rollback()

						Convey("When calling importRuleAccesses with new", func() {
							err := importRuleAccesses(ses, accesses, insert.ID)

							Convey("Then it should return no error", func() {
								So(err, ShouldBeNil)
							})

							Convey("Then the database should contains 3 accesses", func() {
								dbAccesses := []model.RuleAccess{}
								So(ses.Select(&dbAccesses, &database.Filters{
									Conditions: builder.Eq{"rule_id": insert.ID},
								}), ShouldBeNil)
								So(len(dbAccesses), ShouldEqual, 3)

								Convey("Then the data should correspond to the ones imported", func() {
									for i := 0; i < len(dbAccesses); i++ {
										acc := dbAccesses[i]
										if acc.ObjectType == "local_agents" && acc.ObjectID == agent.ID {
											Convey("Then access for agent is found", func() {
											})
										} else if acc.ObjectType == "local_accounts" && acc.ObjectID == account1.ID {
											Convey("Then access for accunt1 is found", func() {
											})
										} else if acc.ObjectType == "local_accounts" && acc.ObjectID == account2.ID {
											Convey("Then access for accunt2 is found", func() {
											})
										} else {
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
	})
}

func TestImportTasks(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some rules", func() {
			insert := &model.Rule{
				Name:   "test",
				IsSend: true,
				Path:   "path/to/rule",
			}
			So(db.Create(insert), ShouldBeNil)

			pre1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Create(pre1), ShouldBeNil)

			pre2 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(pre2), ShouldBeNil)

			post1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "COPY",
				Args:   json.RawMessage(`{"path":"pre1"}`),
			}
			So(db.Create(post1), ShouldBeNil)

			post2 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(post2), ShouldBeNil)

			error1 := &model.Task{
				RuleID: insert.ID,
				Chain:  model.ChainError,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(error1), ShouldBeNil)

			Convey("Given some tasks to import", func() {
				tasks := []ruleTask{
					{
						Type: "COPY",
						Args: []byte(`{"path":"copy/destination"}`),
					},
					{
						Type: "DELETE",
						Args: []byte("{}"),
					},
				}

				Convey("Given a new transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling importTasks on pre tasks", func() {
						err := importRuleTasks(ses, tasks, insert.ID, model.ChainPre)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the database should contains 2 tasks", func() {
							dbTasks := []model.Task{}
							So(ses.Select(&dbTasks, &database.Filters{
								Conditions: builder.Eq{"rule_id": insert.ID, "chain": "PRE"},
							}), ShouldBeNil)
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
						err := importRuleTasks(ses, tasks, insert.ID, model.ChainPost)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the database should contains 2 tasks", func() {
							dbTasks := []model.Task{}
							So(ses.Select(&dbTasks, &database.Filters{
								Conditions: builder.Eq{"rule_id": insert.ID, "chain": "POST"},
							}), ShouldBeNil)
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
						err := importRuleTasks(ses, tasks, insert.ID, model.ChainError)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the database should contains 2 tasks", func() {
							dbTasks := []model.Task{}
							So(ses.Select(&dbTasks, &database.Filters{
								Conditions: builder.Eq{"rule_id": insert.ID, "chain": "ERROR"},
							}), ShouldBeNil)
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
	})
}
