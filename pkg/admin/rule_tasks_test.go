package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func TestListTasks(t *testing.T) {
	logger := log.NewLogger("rest_access_list_logger", logConf)

	Convey("Given the rule creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := listTasks(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule access", func() {
			rule := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "/test/existing/path",
			}
			So(db.Create(rule), ShouldBeNil)
			ruleID := strconv.FormatUint(rule.ID, 10)

			pre := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "COPY",
				Args:   "{}",
			}
			So(db.Create(pre), ShouldBeNil)

			post := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   "MOVE",
				Args:   "{}",
			}
			So(db.Create(post), ShouldBeNil)

			er := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainError,
				Rank:   1,
				Type:   "DELETE",
				Args:   "{}",
			}
			So(db.Create(er), ShouldBeNil)

			Convey("Given a request with the valid rule ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested accesses "+
						"in JSON format", func() {

						expected := map[string][]*model.Task{}
						expected["preTasks"] = []*model.Task{pre}
						expected["postTasks"] = []*model.Task{post}
						expected["errorTasks"] = []*model.Task{er}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing rule ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"rule": "1000"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the body should contain the error message", func() {
						So(w.Body.String(), ShouldEqual, "Record not found\n")
					})
				})
			})
		})
	})
}

func TestUpdateTasks(t *testing.T) {
	logger := log.NewLogger("rest_access_list_logger", logConf)

	Convey("Given the rule creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateTasks(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule access", func() {
			rule := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "/test/existing/path",
			}
			So(db.Create(rule), ShouldBeNil)
			ruleID := strconv.FormatUint(rule.ID, 10)

			Convey("Given all valid new tasks", func() {
				pre := []model.Task{{
					Type: "COPY",
					Args: "{}",
				}}

				post := []model.Task{{
					Type: "MOVE",
					Args: "{}",
				}}

				er := []model.Task{{
					Type: "DELETE",
					Args: "{}",
				}}

				obj := map[string][]model.Task{
					"preTasks":   pre,
					"postTasks":  post,
					"errorTasks": er,
				}

				Convey("Given a request with the valid rule ID parameter", func() {
					body, err := json.Marshal(obj)
					So(err, ShouldBeNil)

					r, err := http.NewRequest(http.MethodGet, "", bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new access", func() {

							accessURI := APIPath + RulesPath + "/" + ruleID +
								RuleTasksPath
							location := w.Header().Get("Location")
							So(location, ShouldStartWith, accessURI)
						})

						Convey("Then the new tasks should be inserted "+
							"in the database", func() {
							exist, err := db.Exists(&pre[0])
							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)

							exist, err = db.Exists(&post[0])
							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)

							exist, err = db.Exists(&er[0])
							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})

				Convey("Given a request with a non-existing rule ID parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "", nil)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": "1000"})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Not Found' error", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the body should contain the error message", func() {
							So(w.Body.String(), ShouldEqual, "Record not found\n")
						})
					})
				})
			})

			Convey("Given partial valid new tasks", func() {
				pre := []model.Task{{
					Type: "COPY",
					Args: "{}",
				}}

				obj := map[string][]model.Task{
					"preTasks": pre,
				}

				Convey("Given a request with the valid rule ID parameter", func() {
					body, err := json.Marshal(obj)
					So(err, ShouldBeNil)

					r, err := http.NewRequest(http.MethodGet, "", bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new access", func() {

							accessURI := APIPath + RulesPath + "/" + ruleID +
								RuleTasksPath
							location := w.Header().Get("Location")
							So(location, ShouldStartWith, accessURI)
						})

						Convey("Then the new tasks should be inserted "+
							"in the database", func() {
							exist, err := db.Exists(&pre[0])
							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)

							exist, err = db.Exists(&model.Task{Chain: model.ChainPost})
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)

							exist, err = db.Exists(&model.Task{Chain: model.ChainError})
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})
			})

			Convey("Given invalid new tasks", func() {
				body := []byte("invalid JSON body")

				Convey("Given a request with the valid rule ID parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "", bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Bad Request'", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain the error "+
							"message", func() {
							So(w.Body.String(), ShouldEqual, "invalid character "+
								"'i' looking for beginning of value\n")
						})

						Convey("Then the new tasks should NOT have been inserted "+
							"in the database", func() {
							exist, err := db.Exists(&model.Task{})
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})
			})
		})
	})
}
