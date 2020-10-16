package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
		handler := createRule(logger, db)
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
				newRule := &InRule{
					UptRule: &UptRule{
						Name:     strPtr("new rule"),
						Comment:  strPtr(""),
						Path:     strPtr("/test/rule/path"),
						InPath:   strPtr("/test/rule/in"),
						OutPath:  strPtr("/test/rule/out"),
						WorkPath: strPtr("/test/rule/work"),
					},
					IsSend: boolPtr(false),
				}

				Convey("Given that the new account is valid for insertion", func() {
					body, err := json.Marshal(newRule)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, ruleURI, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new rule", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, ruleURI+url.PathEscape(
								str(newRule.Name)))
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new rule should be inserted "+
							"in the database", func() {
							var rules []model.Rule
							So(db.Select(&rules, nil), ShouldBeNil)
							So(len(rules), ShouldEqual, 2)

							exp, err := newRule.ToModel(2)
							So(err, ShouldBeNil)
							So(rules[1], ShouldResemble, *exp)
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

		Convey("Given a database with 1 rule", func() {
			rule := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "test/existing/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a request with the valid rule name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"rule": rule.Name,
					"direction": ruleDirection(rule)})

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
						r, err := FromRule(db, rule)
						So(err, ShouldBeNil)
						exp, err := json.Marshal(r)
						So(err, ShouldBeNil)

						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing rule name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"rule": "toto",
					"direction": ruleDirection(rule)})

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
				r = mux.SetURLVars(r, map[string]string{"rule": rule.Name,
					"direction": ruleDirection(rule)})

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
				r = mux.SetURLVars(r, map[string]string{"rule": "toto",
					"direction": ruleDirection(rule)})

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
				Path:    "/old/path",
				InPath:  "/old/in",
				OutPath: "/old/out",
			}
			other := &model.Rule{
				Name: "other",
				Path: "/path/other",
			}
			So(db.Create(old), ShouldBeNil)
			So(db.Create(other), ShouldBeNil)

			pTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage("{}"),
			}
			So(db.Create(pTask), ShouldBeNil)

			Convey("Given new values to update the rule with", func() {
				update := UptRule{
					Name:     strPtr("update"),
					InPath:   strPtr(""),
					WorkPath: strPtr("/update/work"),
				}
				body, err := json.Marshal(update)
				So(err, ShouldBeNil)

				Convey("Given an existing rule name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, ruleURI+old.Name,
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": old.Name,
						"direction": ruleDirection(old)})

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
							So(location, ShouldEqual, ruleURI+str(update.Name))
						})

						Convey("Then the rule should have been updated", func() {
							var results []model.Rule
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.Rule{
								ID:       old.ID,
								Name:     "update",
								Path:     "/old/path",
								InPath:   "",
								OutPath:  "/old/out",
								WorkPath: "/update/work",
							}
							So(results[0], ShouldResemble, expected)

							Convey("Then the tasks should be unchanged", func() {
								//So(db.Get(pTask), ShouldBeNil)
								var p []model.Task
								So(db.Select(&p, nil), ShouldBeNil)
								So(len(p), ShouldEqual, 1)
								So(p[0], ShouldResemble, *pTask)
							})
						})
					})
				})

				Convey("Given a non-existing rule name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, ruleURI+"toto",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": "toto",
						"direction": ruleDirection(old)})

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
			}
			So(db.Create(old), ShouldBeNil)

			pTask := &model.Task{
				RuleID: old.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "DELETE",
				Args:   json.RawMessage("{}"),
			}
			So(db.Create(pTask), ShouldBeNil)

			Convey("Given new values to update the rule with", func() {
				update := UptRule{
					Name: strPtr("update"),
					Path: strPtr("/update/path"),
					PostTasks: []RuleTask{{
						Type: "MOVE",
						Args: json.RawMessage(`{"path":"/move/path"}`),
					}},
				}
				body, err := json.Marshal(update)
				So(err, ShouldBeNil)

				Convey("Given an existing rule name parameter", func() {
					r, err := http.NewRequest(http.MethodPut, ruleURI+old.Name,
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": old.Name,
						"direction": ruleDirection(old)})

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
							So(location, ShouldEqual, ruleURI+str(update.Name))
						})

						Convey("Then the rule should have been updated", func() {
							var results []model.Rule
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 1)

							expected := model.Rule{
								ID:   old.ID,
								Name: str(update.Name),
								Path: str(update.Path),
							}
							So(results[0], ShouldResemble, expected)

							Convey("Then the tasks should have been changed", func() {
								exp := model.Task{
									RuleID: old.ID,
									Chain:  model.ChainPost,
									Rank:   0,
									Type:   update.PostTasks[0].Type,
									Args:   update.PostTasks[0].Args,
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
					r, err := http.NewRequest(http.MethodPut, ruleURI+"toto",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": "toto",
						"direction": ruleDirection(old)})

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
