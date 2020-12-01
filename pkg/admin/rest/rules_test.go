package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const ruleURI = "http://remotehost:8080/api/rules/"

func TestCreateRule(t *testing.T) {
	logger := log.NewLogger("rest_rule_create_logger")

	Convey("Given the rule creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := addRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule", func() {
			existing := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "test/existing/path",
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a new rule to insert in the database", func() {
				body := strings.NewReader(`{
					"name": "new_name",
					"comment": "new comment",
					"path": "/test_path",
					"inPath": "/test_in",
					"outPath": "/test_out",
					"workPath": "/test_work",
					"isSend": false,
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
							var rules []model.Rule
							So(db.Select(&rules, nil), ShouldBeNil)
							So(len(rules), ShouldEqual, 2)

							exp := model.Rule{
								ID:       2,
								Name:     "new_name",
								Comment:  "new comment",
								IsSend:   false,
								Path:     "/test_path",
								InPath:   "/test_in",
								OutPath:  "/test_out",
								WorkPath: "/test_work",
							}
							So(rules[1], ShouldResemble, exp)
						})

						Convey("Then the new tasks should be inserted "+
							"in the database", func() {
							var tasks []model.Task
							So(db.Select(&tasks, nil), ShouldBeNil)
							So(len(tasks), ShouldEqual, 1)

							exp := model.Task{
								RuleID: 2,
								Chain:  model.ChainPre,
								Rank:   0,
								Type:   "DELETE",
								Args:   json.RawMessage(`{}`),
							}
							So(tasks[0], ShouldResemble, exp)
						})

						Convey("Then the existing rule should still be "+
							"present as well", func() {
							var rules []model.Rule
							So(db.Select(&rules, nil), ShouldBeNil)
							So(len(rules), ShouldEqual, 2)

							So(rules[0], ShouldResemble, *existing)
						})
					})
				})
			})
		})
	})
}

func TestGetRule(t *testing.T) {
	logger := log.NewLogger("rest_rule_get_test")

	Convey("Given the rule get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 rules with the same name", func() {
			recv := &model.Rule{
				Name:    "existing",
				Comment: "receive",
				IsSend:  false,
				Path:    "recv/existing/path",
			}
			So(db.Create(recv), ShouldBeNil)

			send := &model.Rule{
				Name:    recv.Name,
				Comment: "send",
				IsSend:  true,
				Path:    "send/existing/path",
			}
			So(db.Create(send), ShouldBeNil)

			SkipConvey("Given a request with the valid rule name parameter", func() {
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
						r, err := FromRule(db, recv)
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
						r, err := FromRule(db, send)
						So(err, ShouldBeNil)
						exp, err := json.Marshal(r)
						So(err, ShouldBeNil)

						So(reflect.ValueOf(send).Elem().Type().Name(), ShouldEqual, "Rule")
						So(reflect.ValueOf(send).Elem().FieldByName("Name").IsZero(), ShouldBeFalse)
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
		})
	})
}

func TestListRules(t *testing.T) {
	logger := log.NewLogger("rest_rules_list_test")

	Convey("Testing the transfer list handler", t, func() {
		db := database.GetTestDatabase()
		handler := listRules(logger, db)
		w := httptest.NewRecorder()

		expected := map[string][]OutRule{}

		Convey("Given a database with 2 rules", func() {
			r1 := &model.Rule{
				Name:   "rule1",
				IsSend: false,
				Path:   "path1",
			}
			So(db.Create(r1), ShouldBeNil)

			r2 := &model.Rule{
				Name:   "rule2",
				IsSend: true,
				Path:   "path2",
			}
			So(db.Create(r2), ShouldBeNil)

			rule1, err := FromRule(db, r1)
			So(err, ShouldBeNil)
			rule2, err := FromRule(db, r2)
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
	logger := log.NewLogger("rest_rule_delete_test")

	Convey("Given the rules deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule", func() {
			rule := &model.Rule{
				Name: "rule",
				Path: "path",
			}
			So(db.Create(rule), ShouldBeNil)

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
						var rules []model.Rule
						So(db.Select(&rules, nil), ShouldBeNil)
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
	logger := log.NewLogger("rest_rule_update_logger")

	Convey("Given the rule updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 rules & a task", func() {
			old := &model.Rule{
				Name:    "old",
				Path:    "/old_path",
				InPath:  "/old_in",
				OutPath: "/old_out",
				IsSend:  true,
			}
			oldRecv := &model.Rule{
				Name:    "old",
				Path:    "/old/pathRecv",
				InPath:  "/old/in",
				OutPath: "/old/out",
				IsSend:  false,
			}
			other := &model.Rule{
				Name:   "other",
				Path:   "/path_other",
				IsSend: false,
			}
			So(db.Create(old), ShouldBeNil)
			So(db.Create(oldRecv), ShouldBeNil)
			So(db.Create(other), ShouldBeNil)

			pTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(pTask), ShouldBeNil)

			poTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(poTask), ShouldBeNil)

			eTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainError,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(eTask), ShouldBeNil)

			Convey("Given new values to update the rule with", func() {
				body := strings.NewReader(`{
					"name": "update_name",
					"inPath": "",
					"workPath": "/update_work",
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
							var results []model.Rule
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 3)

							expected := model.Rule{
								ID:       old.ID,
								Name:     "update_name",
								Path:     "/old_path",
								InPath:   "",
								OutPath:  "/old_out",
								WorkPath: "/update_work",
								IsSend:   true,
							}
							So(results[0], ShouldResemble, expected)

							Convey("Then the tasks should have changed", func() {
								var p []model.Task
								So(db.Select(&p, nil), ShouldBeNil)
								So(len(p), ShouldEqual, 3)

								So(p[0], ShouldResemble, *pTask)
								So(p[1], ShouldResemble, *eTask)
								newPoTask := model.Task{
									RuleID: 1,
									Chain:  model.ChainPost,
									Rank:   0,
									Type:   "MOVE",
									Args:   json.RawMessage(`{"path": "/move/path"}`),
								}
								So(p[2], ShouldResemble, newPoTask)
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
							So(w.Body.String(), ShouldEqual, "rule 'toto' not found\n")
						})

						Convey("Then the old rule should still exist", func() {
							So(db.Get(old), ShouldBeNil)
						})
					})
				})
			})

			for _, rule := range []*model.Rule{old, oldRecv} {
				Convey(fmt.Sprintf("When updating a rule IsSend: %t", rule.IsSend), func() {
					testCases := []UptRule{
						{
							Name: strPtr("update"),
						}, {
							Comment: strPtr("update comment"),
						}, {
							Path: strPtr("/path/update"),
						}, {
							InPath: strPtr("/update/in"),
						}, {
							OutPath: strPtr("/update/out"),
						}, {
							WorkPath: strPtr("/update/work"),
						}, {
							PreTasks: []Task{
								{
									Type: "DELETE",
									Args: []byte("{}"),
								},
							},
						}, {
							PostTasks: []Task{
								{
									Type: "DELETE",
									Args: []byte("{}"),
								},
							},
						}, {
							ErrorTasks: []Task{
								{
									Type: "DELETE",
									Args: []byte("{}"),
								},
							},
						},
					}

					for i, update := range testCases {
						Convey(fmt.Sprintf("TEST %d When updating %s", i, rule.Name), func() {
							_, err := doUpdate(handler, rule, &update)
							So(err, ShouldBeNil)

							Convey("Then only the property updated should be modified", func() {
								expected := getExpected(rule, update)
								dbRule, err := getFromDb(db, expected.Name, rule.IsSend)
								So(err, ShouldBeNil)
								So(dbRule, ShouldResemble, expected)
							})
						})
					}
				})
			}
		})
	})
}

func doUpdate(handler http.HandlerFunc, old *model.Rule, update *UptRule) (*http.Response, error) {
	w := httptest.NewRecorder()
	body, err := json.Marshal(update)
	if err != nil {
		return nil, err
	}
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

func getExpected(src *model.Rule, upt UptRule) *model.Rule {
	res := &model.Rule{
		ID:       src.ID,
		Name:     src.Name,
		Comment:  src.Comment,
		IsSend:   src.IsSend,
		Path:     src.Path,
		InPath:   src.InPath,
		OutPath:  src.OutPath,
		WorkPath: src.WorkPath,
	}
	if upt.Name != nil {
		res.Name = *upt.Name
	}
	if upt.Comment != nil {
		res.Comment = *upt.Comment
	}
	if upt.Path != nil {
		res.Path = *upt.Path
	}
	if upt.InPath != nil {
		res.InPath = *upt.InPath
	}
	if upt.OutPath != nil {
		res.OutPath = *upt.OutPath
	}
	if upt.WorkPath != nil {
		res.WorkPath = *upt.WorkPath
	}
	// TODO Tasks
	return res
}

func getFromDb(db *database.DB, name string, isSend bool) (*model.Rule, error) {
	res := &model.Rule{
		Name:   name,
		IsSend: isSend,
	}
	err := db.Get(res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func TestReplaceRule(t *testing.T) {
	logger := log.NewLogger("rest_rule_replace")

	Convey("Given the rule updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := replaceRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with a rule & a task", func() {
			old := &model.Rule{
				Name:     "old",
				Path:     "/old/path",
				InPath:   "/old/in",
				OutPath:  "/old/out",
				WorkPath: "/old/work",
				IsSend:   true,
			}
			So(db.Create(old), ShouldBeNil)

			pTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Create(pTask), ShouldBeNil)

			Convey("Given new values to update the rule with", func() {
				body := strings.NewReader(`{
					"name": "update_name",
					"path": "/update_path",
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
							var results []model.Rule
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 1)

							expected := model.Rule{
								ID:     old.ID,
								Name:   "update_name",
								Path:   "/update_path",
								IsSend: old.IsSend,
							}
							So(results[0], ShouldResemble, expected)

							Convey("Then the tasks should have been changed", func() {
								exp := model.Task{
									RuleID: old.ID,
									Chain:  model.ChainPost,
									Rank:   0,
									Type:   "MOVE",
									Args:   json.RawMessage(`{"path": "/move/path"}`),
								}
								var tasks []model.Task
								So(db.Select(&tasks, nil), ShouldBeNil)
								So(len(tasks), ShouldEqual, 1)
								So(tasks[0], ShouldResemble, exp)
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
							So(w.Body.String(), ShouldEqual, "rule 'toto' not found\n")
						})

						Convey("Then the old rule should still exist", func() {
							So(db.Get(old), ShouldBeNil)
						})
					})
				})
			})
		})
	})
}
