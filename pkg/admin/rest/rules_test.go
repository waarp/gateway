package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const ruleURI = "http://remotehost:8080/api/rules/"

func TestCreateRule(t *testing.T) {
	logger := log.NewLogger("rest_rule_create_logger")

	Convey("Given the rule creation handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := addRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule", func() {
			existing := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "/existing",
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
					"localTmpDir": "/local/tmp",
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

							exp := model.Rule{
								ID:          2,
								Name:        "new_name",
								Comment:     "new comment",
								IsSend:      false,
								Path:        "/test_path",
								LocalDir:    filepath.FromSlash("/local/dir"),
								RemoteDir:   "/remote/dir",
								LocalTmpDir: filepath.FromSlash("/local/tmp"),
							}
							So(rules[1], ShouldResemble, exp)
						})

						Convey("Then the new tasks should be inserted "+
							"in the database", func() {
							var tasks model.Tasks
							So(db.Select(&tasks).Run(), ShouldBeNil)
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
							var rules model.Rules
							So(db.Select(&rules).Run(), ShouldBeNil)
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

	Convey("Given the rule get handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := getRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 rules with the same name", func() {
			recv := &model.Rule{
				Name:    "existing",
				Comment: "receive",
				IsSend:  false,
				Path:    "/existing",
			}
			So(db.Insert(recv).Run(), ShouldBeNil)

			send := &model.Rule{
				Name:    "existing",
				Comment: "send",
				IsSend:  true,
				Path:    "/existing",
			}
			So(db.Insert(send).Run(), ShouldBeNil)

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

	Convey("Testing the transfer list handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
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

	Convey("Given the rules deletion handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
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
	logger := log.NewLogger("rest_rule_update_logger")

	Convey("Given the rule updating handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := updateRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 rules & a task", func() {
			old := &model.Rule{
				Name:      "old",
				IsSend:    true,
				Path:      "/old_send",
				LocalDir:  "/send/local/dir",
				RemoteDir: "/send/remote/dir",
			}
			oldRecv := &model.Rule{
				Name:        "old",
				IsSend:      false,
				Path:        "/old_recv",
				LocalDir:    "/recv/local/dir",
				RemoteDir:   "/recv/remote/dir",
				LocalTmpDir: "/recv/local/tmp",
			}
			other := &model.Rule{
				Name:   "other",
				Path:   "/path_other",
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
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(pTask).Run(), ShouldBeNil)

			poTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(poTask).Run(), ShouldBeNil)

			eTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainError,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(eTask).Run(), ShouldBeNil)

			Convey("Given new values to update the rule with", func() {
				body := strings.NewReader(`{
					"name": "update_name",
					"localDir": "",
					"localTmpDir": "/local/update/work",
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

							expected := model.Rule{
								ID:          old.ID,
								Name:        "update_name",
								Path:        old.Path,
								LocalDir:    "",
								RemoteDir:   old.RemoteDir,
								LocalTmpDir: filepath.FromSlash("/local/update/work"),
								IsSend:      true,
							}
							So(results[0], ShouldResemble, expected)

							Convey("Then the tasks should have changed", func() {
								var tasks model.Tasks
								So(db.Select(&tasks).Run(), ShouldBeNil)
								So(len(tasks), ShouldEqual, 3)

								So(tasks[0], ShouldResemble, *pTask)
								So(tasks[1], ShouldResemble, *eTask)
								newPoTask := model.Task{
									RuleID: 1,
									Chain:  model.ChainPost,
									Rank:   0,
									Type:   "MOVE",
									Args:   json.RawMessage(`{"path": "/move/path"}`),
								}
								So(tasks[2], ShouldResemble, newPoTask)
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
								" rule 'toto' not found\n")
						})

						Convey("Then the old rule should still exist", func() {
							var rules model.Rules
							So(db.Select(&rules).Run(), ShouldBeNil)
							So(rules, ShouldHaveLength, 3)
							So(rules[0], ShouldResemble, *old)
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
							Path: strPtr("/path_update"),
						}, {
							LocalDir: strPtr("/update/local"),
						}, {
							RemoteDir: strPtr("/update/remote"),
						}, {
							LocalTmpDir: strPtr("/update/tmp"),
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
		ID:          src.ID,
		Name:        src.Name,
		Comment:     src.Comment,
		IsSend:      src.IsSend,
		Path:        src.Path,
		LocalDir:    src.LocalDir,
		RemoteDir:   src.RemoteDir,
		LocalTmpDir: src.LocalTmpDir,
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
	if upt.LocalDir != nil {
		res.LocalDir = utils.ToOSPath(*upt.LocalDir)
	}
	if upt.RemoteDir != nil {
		res.RemoteDir = *upt.RemoteDir
	}
	if upt.LocalTmpDir != nil {
		res.LocalTmpDir = utils.ToOSPath(*upt.LocalTmpDir)
	}
	// TODO Tasks
	return res
}

func getFromDb(db *database.DB, name string, isSend bool) (*model.Rule, error) {
	var rule model.Rule
	if err := db.Get(&rule, "name=? AND send=?", name, isSend).Run(); err != nil {
		return nil, err
	}
	return &rule, nil
}

func TestReplaceRule(t *testing.T) {
	logger := log.NewLogger("rest_rule_replace")

	Convey("Given the rule updating handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := replaceRule(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with a rule & a task", func() {
			old := &model.Rule{
				Name:        "old",
				Path:        "/old",
				LocalDir:    "/old/local",
				RemoteDir:   "/old/remote",
				LocalTmpDir: "/old/tmp",
				IsSend:      true,
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			pTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage(`{}`),
			}
			So(db.Insert(pTask).Run(), ShouldBeNil)

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
							var results model.Rules
							So(db.Select(&results).Run(), ShouldBeNil)
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
								var tasks model.Tasks
								So(db.Select(&tasks).Run(), ShouldBeNil)
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
							So(w.Body.String(), ShouldEqual, ruleDirection(old)+
								" rule 'toto' not found\n")
						})

						Convey("Then the old rule should still exist", func() {
							var rules model.Rules
							So(db.Select(&rules).Run(), ShouldBeNil)
							So(rules, ShouldNotBeEmpty)
							So(rules[0], ShouldResemble, *old)
						})
					})
				})
			})
		})
	})
}
