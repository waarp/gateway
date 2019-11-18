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

func TestCreateAccess(t *testing.T) {
	logger := log.NewLogger("rest_access_create_logger", logConf)

	Convey("Given the rule creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createAccess(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule access", func() {
			object := &model.LocalAgent{
				Name:        "object1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(object), ShouldBeNil)

			rule := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "/test/existing/path",
			}
			So(db.Create(rule), ShouldBeNil)
			ruleID := strconv.FormatUint(rule.ID, 10)

			Convey("Given a new access to insert in the database", func() {
				acc := &model.RuleAccess{
					ObjectID:   object.ID,
					ObjectType: object.TableName(),
				}

				Convey("Given that the new access is valid for insertion", func() {
					body, err := json.Marshal(acc)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new access", func() {

							accessURI := APIPath + RulesPath + "/" + ruleID + RulePermissionPath
							location := w.Header().Get("Location")
							So(location, ShouldStartWith, accessURI)
						})

						Convey("Then the response body should state that access "+
							"to the rule is now restricted", func() {
							So(w.Body.String(), ShouldEqual, "Access to rule 1 "+
								"is now restricted\n")
						})

						Convey("Then the new access should be inserted "+
							"in the database", func() {
							exist, err := db.Exists(acc)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the new access already exist", func() {
					ex := &model.RuleAccess{
						RuleID:     rule.ID,
						ObjectID:   acc.ObjectID,
						ObjectType: acc.ObjectType,
					}
					So(db.Create(ex), ShouldBeNil)

					body, err := json.Marshal(acc)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should state that access "+
							"to the rule is now restricted", func() {
							So(w.Body.String(), ShouldEqual, "The agent has "+
								"already been granted access to this rule\n")
						})
					})
				})

				Convey("Given a request with a non-existing rule ID parameter", func() {
					body, err := json.Marshal(acc)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
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

			Convey("Given that the JSON body in invalid", func() {
				body := []byte("invalid JSON body")
				r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should state that access "+
						"to the rule is now restricted", func() {
						So(w.Body.String(), ShouldEqual, "invalid character 'i' "+
							"looking for beginning of value\n")
					})
				})
			})
		})
	})
}

func TestListAccess(t *testing.T) {
	logger := log.NewLogger("rest_access_list_logger", logConf)

	Convey("Given the rule creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := listAccess(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule access", func() {
			object1 := &model.LocalAgent{
				Name:        "object1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(object1), ShouldBeNil)

			object2 := &model.LocalAgent{
				Name:        "object2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(object2), ShouldBeNil)

			rule := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "/test/existing/path",
			}
			So(db.Create(rule), ShouldBeNil)
			ruleID := strconv.FormatUint(rule.ID, 10)

			acc1 := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   object1.ID,
				ObjectType: object1.TableName(),
			}
			So(db.Create(acc1), ShouldBeNil)

			acc2 := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   object2.ID,
				ObjectType: object2.TableName(),
			}
			So(db.Create(acc2), ShouldBeNil)

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

						expected := map[string][]*model.RuleAccess{}
						expected["permissions"] = []*model.RuleAccess{acc1, acc2}
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

func TestDeleteAccess(t *testing.T) {
	logger := log.NewLogger("rest_access_list_logger", logConf)

	Convey("Given the rule creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteAccess(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 rule access", func() {
			object := &model.LocalAgent{
				Name:        "object1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
			}
			So(db.Create(object), ShouldBeNil)

			rule := &model.Rule{
				Name:    "existing",
				Comment: "",
				IsSend:  false,
				Path:    "/test/existing/path",
			}
			So(db.Create(rule), ShouldBeNil)
			ruleID := strconv.FormatUint(rule.ID, 10)

			acc := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   object.ID,
				ObjectType: object.TableName(),
			}
			So(db.Create(acc), ShouldBeNil)

			Convey("Given that the access can be deleted", func() {
				body, err := json.Marshal(acc)
				So(err, ShouldBeNil)
				r, err := http.NewRequest(http.MethodDelete, "", bytes.NewReader(body))
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the response body should state that access "+
						"to the rule is now unrestricted", func() {
						So(w.Body.String(), ShouldEqual, "Access to rule 1 is "+
							"now unrestricted\n")
					})

					Convey("Then the access should have been deleted from the "+
						"database", func() {
						exist, err := db.Exists(acc)

						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})

			Convey("Given a request with a non-existing rule ID parameter", func() {
				body, err := json.Marshal(acc)
				So(err, ShouldBeNil)
				r, err := http.NewRequest(http.MethodDelete, "", bytes.NewReader(body))
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

			Convey("Given that the access does not exist", func() {
				other := &model.RuleAccess{
					ObjectID:   1000,
					ObjectType: object.TableName(),
				}

				body, err := json.Marshal(other)
				So(err, ShouldBeNil)
				r, err := http.NewRequest(http.MethodDelete, "", bytes.NewReader(body))
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"rule": ruleID})

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
