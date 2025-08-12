package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	. "code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const ruleURI = "http://remotehost:8080/api/rules/"

func TestCreateRule(t *testing.T) {
	Convey("Given the rule creation handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_rule_create_logger")
		db := database.TestDatabase(c)
		handler := addRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule", func() {
			existing := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "test/existing/path",
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a new rule to insert in the database", func() {
				body := strings.NewReader(`{
					"name": "new_name",
					"isSend": false,
					"comment": "new comment",
					"path": "/test_path",
					"localDir": "/local/dir",
					"remoteDir": "/remote/dir",
					"tmpLocalRcvDir": "/local/tmp",
					"preTasks": [{
						"type": "DELETE"
					}]
				}`)

				Convey("Given that the new account is valid for insertion", func() {
					r, err := http.NewRequest(http.MethodPost, ruleURI, body)

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new rule", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, ruleURI+"new_name")
						})

						Convey("Then the new rule should be inserted "+
							"in the database", func() {
							var rules model.Rules

							So(db.Select(&rules).Run(), ShouldBeNil)
							So(len(rules), ShouldEqual, 2)
							So(rules[1], ShouldResemble, &model.Rule{
								ID:             2,
								Name:           "new_name",
								Comment:        "new comment",
								IsSend:         false,
								Path:           "test_path",
								LocalDir:       "/local/dir",
								RemoteDir:      "/remote/dir",
								TmpLocalRcvDir: "/local/tmp",
							})
						})

						Convey("Then the new tasks should be inserted "+
							"in the database", func() {
							var tasks model.Tasks

							So(db.Select(&tasks).Run(), ShouldBeNil)
							So(len(tasks), ShouldEqual, 1)
							So(tasks[0], ShouldResemble, &model.Task{
								RuleID: 2,
								Chain:  model.ChainPre,
								Rank:   0,
								Type:   "DELETE",
								Args:   map[string]string{},
							})
						})

						Convey("Then the existing rule should still be "+
							"present as well", func() {
							var rules model.Rules

							So(db.Select(&rules).Run(), ShouldBeNil)
							So(len(rules), ShouldEqual, 2)
							So(rules[0], ShouldResemble, existing)
						})
					})
				})
			})
		})
	})
}

func TestGetRule(t *testing.T) {
	Convey("Given the rule get handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_rule_get_test")
		db := database.TestDatabase(c)
		handler := getRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 rules with the same name", func() {
			recv := &model.Rule{
				Name:    "existing",
				Comment: "receive",
				IsSend:  false,
				Path:    "recv/existing/path",
			}
			So(db.Insert(recv).Run(), ShouldBeNil)

			send := &model.Rule{
				Name:    "existing",
				Comment: "send",
				IsSend:  true,
				Path:    "send/existing/path",
			}
			So(db.Insert(send).Run(), ShouldBeNil)

			Convey("Given a request with the valid rule name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"rule":      recv.Name,
					"direction": ruleDirection(recv),
				})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain "+
						"'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested rule "+
						"in JSON format", func() {
						r, err := DBRuleToREST(db, recv)
						So(err, ShouldBeNil)
						exp, err := json.Marshal(r)
						So(err, ShouldBeNil)

						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with the same rule name but different direction", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"rule":      send.Name,
					"direction": ruleDirection(send),
				})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain "+
						"'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested rule "+
						"in JSON format", func() {
						r, err := DBRuleToREST(db, send)
						So(err, ShouldBeNil)
						exp, err := json.Marshal(r)
						So(err, ShouldBeNil)

						So(send.Name, ShouldNotBeEmpty)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing rule name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"rule":      "toto",
					"direction": ruleDirection(recv),
				})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})

			Convey("Given some agents", func() {
				serv1 := &model.LocalAgent{
					Name:     "serv1",
					Address:  types.Addr("localhost", 1),
					Protocol: testProto1,
				}
				serv2 := &model.LocalAgent{
					Name:     "serv2",
					Address:  types.Addr("localhost", 2),
					Protocol: testProto2,
				}

				So(db.Insert(serv1).Run(), ShouldBeNil)
				So(db.Insert(serv2).Run(), ShouldBeNil)

				serv1acc1 := &model.LocalAccount{
					LocalAgentID: serv1.ID,
					Login:        "acc1",
				}
				serv1acc2 := &model.LocalAccount{
					LocalAgentID: serv1.ID,
					Login:        "acc2",
				}
				serv2acc1 := &model.LocalAccount{
					LocalAgentID: serv2.ID,
					Login:        "acc1",
				}

				So(db.Insert(serv1acc1).Run(), ShouldBeNil)
				So(db.Insert(serv1acc2).Run(), ShouldBeNil)
				So(db.Insert(serv2acc1).Run(), ShouldBeNil)

				part1 := &model.RemoteAgent{
					Name:     "part1",
					Address:  types.Addr("localhost", 10),
					Protocol: testProto1,
				}
				part2 := &model.RemoteAgent{
					Name:     "part2",
					Address:  types.Addr("localhost", 20),
					Protocol: testProto2,
				}

				So(db.Insert(part1).Run(), ShouldBeNil)
				So(db.Insert(part2).Run(), ShouldBeNil)

				part1acc1 := &model.RemoteAccount{
					RemoteAgentID: part1.ID,
					Login:         "acc1",
				}
				part2acc1 := &model.RemoteAccount{
					RemoteAgentID: part2.ID,
					Login:         "acc2",
				}
				part2acc2 := &model.RemoteAccount{
					RemoteAgentID: part2.ID,
					Login:         "acc1",
				}

				So(db.Insert(part1acc1).Run(), ShouldBeNil)
				So(db.Insert(part2acc1).Run(), ShouldBeNil)
				So(db.Insert(part2acc2).Run(), ShouldBeNil)

				Convey("Given some authorizations on those agents", func() {
					authServ1 := &model.RuleAccess{
						RuleID:       send.ID,
						LocalAgentID: utils.NewNullInt64(serv1.ID),
					}
					authServ1Acc1 := &model.RuleAccess{
						RuleID:         send.ID,
						LocalAccountID: utils.NewNullInt64(serv1acc1.ID),
					}
					authServ1Acc2 := &model.RuleAccess{
						RuleID:         send.ID,
						LocalAccountID: utils.NewNullInt64(serv1acc2.ID),
					}
					authServ2Acc1 := &model.RuleAccess{
						RuleID:         send.ID,
						LocalAccountID: utils.NewNullInt64(serv2acc1.ID),
					}
					authPart1 := &model.RuleAccess{
						RuleID:        send.ID,
						RemoteAgentID: utils.NewNullInt64(part1.ID),
					}
					authPart1Acc1 := &model.RuleAccess{
						RuleID:          send.ID,
						RemoteAccountID: utils.NewNullInt64(part1acc1.ID),
					}
					authPart2Acc1 := &model.RuleAccess{
						RuleID:          send.ID,
						RemoteAccountID: utils.NewNullInt64(part2acc1.ID),
					}

					So(db.Insert(authServ1).Run(), ShouldBeNil)
					So(db.Insert(authServ1Acc1).Run(), ShouldBeNil)
					So(db.Insert(authServ1Acc2).Run(), ShouldBeNil)
					So(db.Insert(authServ2Acc1).Run(), ShouldBeNil)
					So(db.Insert(authPart1).Run(), ShouldBeNil)
					So(db.Insert(authPart1Acc1).Run(), ShouldBeNil)
					So(db.Insert(authPart2Acc1).Run(), ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						//nolint:noctx // this is a test
						r, err := http.NewRequest(http.MethodGet, "", nil)
						So(err, ShouldBeNil)

						r = mux.SetURLVars(r, map[string]string{
							"rule":      send.Name,
							"direction": ruleDirection(send),
						})
						handler.ServeHTTP(w, r)

						Convey("Then it should have returned the correct authorizations", func() {
							var rule OutRule

							So(json.Unmarshal(w.Body.Bytes(), &rule), ShouldBeNil)
							So(rule.Authorized.LocalServers, ShouldResemble, []string{serv1.Name})
							So(rule.Authorized.RemotePartners, ShouldResemble, []string{part1.Name})
							So(rule.Authorized.LocalAccounts, ShouldResemble, map[string][]string{
								serv1.Name: {serv1acc1.Login, serv1acc2.Login},
								serv2.Name: {serv2acc1.Login},
							})
							So(rule.Authorized.RemoteAccounts, ShouldResemble, map[string][]string{
								part1.Name: {part1acc1.Login},
								part2.Name: {part2acc1.Login},
							})
						})
					})
				})
			})
		})
	})
}

func TestListRules(t *testing.T) {
	Convey("Testing the transfer list handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_rules_list_test")
		db := database.TestDatabase(c)
		handler := listRules(logger, db)
		w := httptest.NewRecorder()

		expected := map[string][]OutRule{}

		Convey("Given a database with 2 rules", func() {
			r1 := &model.Rule{
				Name:   "rule1",
				IsSend: false,
				Path:   "path1",
			}
			So(db.Insert(r1).Run(), ShouldBeNil)

			r2 := &model.Rule{
				Name:   "rule2",
				IsSend: true,
				Path:   "path2",
			}
			So(db.Insert(r2).Run(), ShouldBeNil)

			rule1, err := DBRuleToREST(db, r1)
			So(err, ShouldBeNil)
			rule2, err := DBRuleToREST(db, r2)
			So(err, ShouldBeNil)

			Convey("Given a valid request", func() {
				req, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return the 2 rules", func() {
						expected["rules"] = []OutRule{*rule1, *rule2}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})
		})
	})
}

func TestDeleteRule(t *testing.T) {
	Convey("Given the rules deletion handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_rule_delete_test")
		db := database.TestDatabase(c)
		handler := deleteRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule", func() {
			rule := &model.Rule{
				Name: "rule",
				Path: "path",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			Convey("Given a request with the valid rule name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"rule":      rule.Name,
					"direction": ruleDirection(rule),
				})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the rule should no longer be present "+
						"in the database", func() {
						var rules model.Rules

						So(db.Select(&rules).Run(), ShouldBeNil)
						So(rules, ShouldBeEmpty)
					})
				})
			})

			Convey("Given a request with a non-existing rule name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"rule":      "toto",
					"direction": ruleDirection(rule),
				})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}

func TestUpdateRule(t *testing.T) {
	Convey("Given the rule updating handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_rule_update_logger")
		db := database.TestDatabase(c)
		handler := updateRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 rules & a task", func() {
			old := &model.Rule{
				Name:      "old",
				IsSend:    true,
				Path:      "old/send",
				LocalDir:  "/send/local/dir",
				RemoteDir: "/send/remote/dir",
			}
			oldRecv := &model.Rule{
				Name:           "old",
				IsSend:         false,
				Path:           "old/pathRecv",
				LocalDir:       "/recv/local/dir",
				RemoteDir:      "/recv/remote/dir",
				TmpLocalRcvDir: "/recv/local/tmp",
			}
			other := &model.Rule{
				Name:   "other",
				Path:   "path/other",
				IsSend: false,
			}

			So(db.Insert(old).Run(), ShouldBeNil)
			So(db.Insert(oldRecv).Run(), ShouldBeNil)
			So(db.Insert(other).Run(), ShouldBeNil)

			pTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "DELETE",
			}
			So(db.Insert(pTask).Run(), ShouldBeNil)

			poTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "DELETE",
			}
			So(db.Insert(poTask).Run(), ShouldBeNil)

			eTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainError,
				Rank:   0,
				Type:   "DELETE",
			}
			So(db.Insert(eTask).Run(), ShouldBeNil)

			Convey("Given new values to update the rule with", func() {
				body := strings.NewReader(`{
					"name": "update_name",
					"localDir": "",
					"tmpLocalRcvDir": "/local/update/work",
					"postTasks": [{
						"type": "MOVE",
						"args": {"path": "/move/path"}
					}]
				}`)

				Convey("Given an existing rule name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, ruleURI+old.Name, body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"rule":      old.Name,
						"direction": ruleDirection(old),
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated rule", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, ruleURI+"update_name")
						})

						Convey("Then the rule should have been updated", func() {
							var results model.Rules

							So(db.Select(&results).OrderBy("id", true).Run(), ShouldBeNil)
							So(len(results), ShouldEqual, 3)
							So(results[0], ShouldResemble, &model.Rule{
								ID:             old.ID,
								Name:           "update_name",
								Path:           old.Path,
								LocalDir:       "",
								RemoteDir:      old.RemoteDir,
								TmpLocalRcvDir: "/local/update/work",
								IsSend:         true,
							})

							Convey("Then the tasks should have changed", func() {
								var tasks model.Tasks

								So(db.Select(&tasks).Run(), ShouldBeNil)
								So(len(tasks), ShouldEqual, 3)

								So(tasks, ShouldContain, pTask)
								So(tasks, ShouldContain, eTask)
								So(tasks, ShouldContain, &model.Task{
									RuleID: 1,
									Chain:  model.ChainPost,
									Rank:   0,
									Type:   "MOVE",
									Args:   map[string]string{"path": "/move/path"},
								})
							})
						})
					})
				})

				Convey("Given a non-existing rule name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, ruleURI+"toto", body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"rule":      "toto",
						"direction": ruleDirection(old),
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the rule was not found", func() {
							So(w.Body.String(), ShouldEqual, ruleDirection(old)+
								" rule \"toto\" not found\n")
						})

						Convey("Then the old rule should still exist", func() {
							var rules model.Rules

							So(db.Select(&rules).Run(), ShouldBeNil)
							So(rules, ShouldHaveLength, 3)
							So(rules[0], ShouldResemble, old)
						})
					})
				})
			})

			for _, rule := range []*model.Rule{old, oldRecv} {
				Convey(fmt.Sprintf("When updating a rule IsSend: %t", rule.IsSend), func() {
					testCases := []InRule{
						{
							Name: asNullable("update"),
						}, {
							Comment: asNullable("update comment"),
						}, {
							Path: asNullable("path/update"),
						}, {
							LocalDir: asNullable("/update/local"),
						}, {
							RemoteDir: asNullable("/update/remote"),
						}, {
							TmpLocalRcvDir: asNullable("/update/tmp"),
						}, {
							PreTasks: []*Task{
								{
									Type: "DELETE",
									Args: map[string]string{},
								},
							},
						}, {
							PostTasks: []*Task{
								{
									Type: "DELETE",
									Args: map[string]string{},
								},
							},
						}, {
							ErrorTasks: []*Task{
								{
									Type: "DELETE",
									Args: map[string]string{},
								},
							},
						},
					}

					for i, update := range testCases {
						Convey(fmt.Sprintf("TEST %d When updating %s", i, rule.Name), func() {
							resp, err := doUpdate(handler, rule, &update)
							So(err, ShouldBeNil)

							defer resp.Body.Close()

							Convey("Then only the property updated should be modified", func() {
								expected := getExpected(rule, &update)
								var dbRule model.Rule
								So(db.Get(&dbRule, "id=?", rule.ID).Run(), ShouldBeNil)
								So(&dbRule, ShouldResemble, expected)
							})
						})
					}
				})
			}
		})
	})
}

//nolint:wrapcheck // this is a test helper, err must be passed as is
func doUpdate(handler http.HandlerFunc, old *model.Rule, update *InRule) (*http.Response, error) {
	w := httptest.NewRecorder()

	body, err := json.Marshal(update)
	if err != nil {
		return nil, err
	}

	//nolint:noctx //this is a test
	r, err := http.NewRequest(http.MethodPatch, ruleURI+old.Name,
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	r = mux.SetURLVars(r, map[string]string{
		"rule":      old.Name,
		"direction": ruleDirection(old),
	})
	handler.ServeHTTP(w, r)

	return w.Result(), nil
}

func getExpected(src *model.Rule, upt *InRule) *model.Rule {
	res := &model.Rule{
		ID:             src.ID,
		Name:           src.Name,
		Comment:        src.Comment,
		IsSend:         src.IsSend,
		Path:           src.Path,
		LocalDir:       src.LocalDir,
		RemoteDir:      src.RemoteDir,
		TmpLocalRcvDir: src.TmpLocalRcvDir,
	}

	if upt.Name.Valid {
		res.Name = upt.Name.Value
	}

	if upt.Comment.Valid {
		res.Comment = upt.Comment.Value
	}

	if upt.Path.Valid {
		res.Path = upt.Path.Value
	}

	if upt.LocalDir.Valid {
		res.LocalDir = upt.LocalDir.Value
	}

	if upt.RemoteDir.Valid {
		res.RemoteDir = upt.RemoteDir.Value
	}

	if upt.TmpLocalRcvDir.Valid {
		res.TmpLocalRcvDir = upt.TmpLocalRcvDir.Value
	}

	// TODO Tasks
	return res
}

func TestReplaceRule(t *testing.T) {
	Convey("Given the rule updating handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_rule_replace")
		db := database.TestDatabase(c)
		handler := replaceRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with a rule & a task", func() {
			old := &model.Rule{
				Name:           "old",
				Path:           "old/path",
				LocalDir:       "/old/local",
				RemoteDir:      "/old/remote",
				TmpLocalRcvDir: "/old/tmp",
				IsSend:         true,
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			pTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "DELETE",
			}
			So(db.Insert(pTask).Run(), ShouldBeNil)

			Convey("Given new values to update the rule with", func() {
				body := strings.NewReader(`{
					"name": "update_name",
					"path": "update/path",
					"isSend": false,
					"postTasks": [{
						"type": "MOVE",
						"args": {"path": "/move/path"}
					}]
				}`)

				Convey("Given an existing rule name parameter", func() {
					r, err := http.NewRequest(http.MethodPut, ruleURI+old.Name, body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"rule":      old.Name,
						"direction": ruleDirection(old),
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated rule", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, ruleURI+"update_name")
						})

						Convey("Then the rule should have been updated", func() {
							var results model.Rules

							So(db.Select(&results).Run(), ShouldBeNil)
							So(len(results), ShouldEqual, 1)
							So(results[0], ShouldResemble, &model.Rule{
								ID:     old.ID,
								Name:   "update_name",
								Path:   "update/path",
								IsSend: false,
							})

							Convey("Then the tasks should have been changed", func() {
								var tasks model.Tasks

								So(db.Select(&tasks).Run(), ShouldBeNil)
								So(len(tasks), ShouldEqual, 1)
								So(tasks[0], ShouldResemble, &model.Task{
									RuleID: old.ID,
									Chain:  model.ChainPost,
									Rank:   0,
									Type:   "MOVE",
									Args:   map[string]string{"path": "/move/path"},
								})
							})
						})
					})
				})

				Convey("Given a non-existing rule name parameter", func() {
					r, err := http.NewRequest(http.MethodPut, ruleURI+"toto", body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"rule":      "toto",
						"direction": ruleDirection(old),
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the rule was not found", func() {
							So(w.Body.String(), ShouldEqual, ruleDirection(old)+
								" rule \"toto\" not found\n")
						})

						Convey("Then the old rule should still exist", func() {
							var rules model.Rules

							So(db.Select(&rules).Run(), ShouldBeNil)
							So(rules, ShouldNotBeEmpty)
							So(rules[0], ShouldResemble, old)
						})
					})
				})
			})
		})
	})
}
