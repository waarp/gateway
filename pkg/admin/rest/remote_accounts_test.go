package rest

import (
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
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func remoteAccountsURI(agent, login string) string {
	return fmt.Sprintf("http://localhost:8080/api/partners/%s/accounts/%s", agent, login)
}

func TestGetRemoteAccount(t *testing.T) {
	Convey("Given the account get handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_account_get_test")
		db := database.TestDatabase(c)
		handler := getRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			existing := &model.RemoteAccount{
				Login:         "existing",
				RemoteAgentID: parent.ID,
				Password:      "existing",
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a request with the valid account login parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"partner":        parent.Name,
					"remote_account": existing.Login,
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

					Convey("Then the body should contain the requested partner "+
						"in JSON format", func() {
						expected, err := DBRemoteAccountToREST(db, existing)
						So(err, ShouldBeNil)

						exp, err := json.Marshal(expected)
						So(err, ShouldBeNil)

						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing account login parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"partner":        parent.Name,
					"remote_account": "toto",
				})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})

			Convey("Given a request with a non-existing agent name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"partner":        "toto",
					"remote_account": existing.Login,
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

func TestListRemoteAccounts(t *testing.T) {
	check := func(w *httptest.ResponseRecorder, expected map[string][]*OutAccount) {
		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})

		Convey("Then the response body should contain an array "+
			"of the requested accounts in JSON format", func() {
			exp, err := json.Marshal(expected)
			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldResemble, string(exp)+"\n")
		})
	}

	Convey("Given the remote account listing handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_account_list_test")
		db := database.TestDatabase(c)
		handler := listRemoteAccounts(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]*OutAccount{}

		Convey("Given a database with 4 remote accounts", func() {
			p1 := &model.RemoteAgent{
				Name:     "parent1",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			p2 := &model.RemoteAgent{
				Name:     "parent2",
				Protocol: testProto1,
				Address:  "localhost:2",
			}

			So(db.Insert(p1).Run(), ShouldBeNil)
			So(db.Insert(p2).Run(), ShouldBeNil)

			a1 := &model.RemoteAccount{
				Login:         "account1",
				Password:      "account1",
				RemoteAgentID: p1.ID,
			}
			a2 := &model.RemoteAccount{
				Login:         "account2",
				Password:      "account2",
				RemoteAgentID: p1.ID,
			}
			a3 := &model.RemoteAccount{
				Login:         "account3",
				Password:      "account3",
				RemoteAgentID: p2.ID,
			}
			a4 := &model.RemoteAccount{
				Login:         "account4",
				Password:      "account4",
				RemoteAgentID: p1.ID,
			}

			So(db.Insert(a1).Run(), ShouldBeNil)
			So(db.Insert(a2).Run(), ShouldBeNil)
			So(db.Insert(a3).Run(), ShouldBeNil)
			So(db.Insert(a4).Run(), ShouldBeNil)

			account1, err := DBRemoteAccountToREST(db, a1)
			So(err, ShouldBeNil)
			account2, err := DBRemoteAccountToREST(db, a2)
			So(err, ShouldBeNil)
			account3, err := DBRemoteAccountToREST(db, a3)
			So(err, ShouldBeNil)
			account4, err := DBRemoteAccountToREST(db, a4)
			So(err, ShouldBeNil)

			Convey("Given a request with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []*OutAccount{
						account1, account2, account4,
					}
					check(w, expected)
				})
			})

			Convey("Given a request with a different agent", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": p2.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []*OutAccount{account3}
					check(w, expected)
				})
			})

			Convey("Given a request with an invalid agent", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": "toto"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []*OutAccount{account1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []*OutAccount{account2, account4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=login-", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []*OutAccount{
						account4, account2, account1,
					}
					check(w, expected)
				})
			})
		})
	})
}

func TestCreateRemoteAccount(t *testing.T) {
	Convey("Given the account creation handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_account_create_logger")
		db := database.TestDatabase(c)
		handler := addRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			parent := &model.RemoteAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			Convey("Given a new account to insert in the database", func() {
				body := strings.NewReader(`{
					"login": "new_account",
					"password": "new_password"
				}`)

				Convey("Given a valid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPost, remoteAccountsURI(
						parent.Name, ""), body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"partner": parent.Name})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the "+
							"URI of the new account", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, remoteAccountsURI(parent.Name,
								"new_account"))
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new account should be inserted in the "+
							"database", func() {
							var accs model.RemoteAccounts
							So(db.Select(&accs).Run(), ShouldBeNil)
							So(len(accs), ShouldEqual, 1)

							So(accs[0], ShouldResemble, &model.RemoteAccount{
								ID:            1,
								RemoteAgentID: parent.ID,
								Login:         "new_account",
								Password:      "new_password",
							})
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						"toto", ""), body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"partner": "toto"})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the account was not found", func() {
							So(w.Body.String(), ShouldEqual, "partner 'toto' not found\n")
						})

						Convey("Then the new account should NOT exist", func() {
							var accs model.RemoteAccounts
							So(db.Select(&accs).Run(), ShouldBeNil)
							So(accs, ShouldBeEmpty)
						})
					})
				})
			})
		})
	})
}

func TestDeleteRemoteAccount(t *testing.T) {
	Convey("Given the account deletion handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_account_delete_test")
		db := database.TestDatabase(c)
		handler := deleteRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			existing := &model.RemoteAccount{
				Login:         "existing",
				Password:      "existing",
				RemoteAgentID: parent.ID,
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a request with the valid account login parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"partner":        parent.Name,
					"remote_account": existing.Login,
				})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the account should no longer be present "+
						"in the database", func() {
						var accs model.RemoteAccounts
						So(db.Select(&accs).Run(), ShouldBeNil)
						So(accs, ShouldBeEmpty)
					})
				})
			})

			Convey("Given a request with a non-existing account login parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"partner":        parent.Name,
					"remote_account": "toto",
				})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})

			Convey("Given a request with a non-existing agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{
					"partner":        "toto",
					"remote_account": existing.Login,
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

func TestUpdateRemoteAccount(t *testing.T) {
	Convey("Given the account updating handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_account_update_logger")
		db := database.TestDatabase(c)
		handler := updateRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			old := &model.RemoteAccount{
				Login:         "old",
				Password:      "old",
				RemoteAgentID: parent.ID,
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				body := strings.NewReader(`{
					"password": "upd_password"
				}`)

				Convey("Given a valid account login parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						parent.Name, old.Login), body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"partner":        parent.Name,
						"remote_account": old.Login,
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated account", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, remoteAccountsURI(parent.Name,
								old.Login))
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the account should have been updated", func() {
							var res model.RemoteAccounts
							So(db.Select(&res).Run(), ShouldBeNil)
							So(len(res), ShouldEqual, 1)

							So(res[0], ShouldResemble, &model.RemoteAccount{
								ID:            old.ID,
								RemoteAgentID: parent.ID,
								Login:         "old",
								Password:      res[0].Password,
							})
						})
					})
				})

				Convey("Given an invalid account login parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						parent.Name, "toto"), body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"partner":        parent.Name,
						"remote_account": "toto",
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the account was not found", func() {
							So(w.Body.String(), ShouldEqual, "no account 'toto' "+
								"found for partner parent\n")
						})

						Convey("Then the old account should still exist", func() {
							var accs model.RemoteAccounts
							So(db.Select(&accs).Run(), ShouldBeNil)
							So(accs, ShouldNotBeEmpty)
							So(accs[0], ShouldResemble, old)
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						"toto", old.Login), body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"partner":        "toto",
						"remote_account": old.Login,
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the account was not found", func() {
							So(w.Body.String(), ShouldEqual, "partner 'toto' not found\n")
						})

						Convey("Then the old account should still exist", func() {
							var accs model.RemoteAccounts
							So(db.Select(&accs).Run(), ShouldBeNil)
							So(accs, ShouldNotBeEmpty)
							So(accs[0], ShouldResemble, old)
						})
					})
				})
			})
		})
	})
}

func TestReplaceRemoteAccount(t *testing.T) {
	Convey("Given the account updating handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_account_update_logger")
		db := database.TestDatabase(c)
		handler := replaceRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			old := &model.RemoteAccount{
				Login:         "old",
				Password:      "old",
				RemoteAgentID: parent.ID,
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				body := strings.NewReader(`{
					"login": "upd_login",
					"password": "upd_password"
				}`)

				Convey("Given a valid account login parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						parent.Name, old.Login), body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"partner":        parent.Name,
						"remote_account": old.Login,
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
							"the URI of the updated account", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, remoteAccountsURI("parent", "upd_login"))
						})

						Convey("Then the account should have been updated", func() {
							var res model.RemoteAccounts
							So(db.Select(&res).Run(), ShouldBeNil)
							So(len(res), ShouldEqual, 1)

							So(res[0], ShouldResemble, &model.RemoteAccount{
								ID:            old.ID,
								RemoteAgentID: parent.ID,
								Login:         "upd_login",
								Password:      res[0].Password,
							})
						})
					})
				})

				Convey("Given an invalid account login parameter", func() {
					r, err := http.NewRequest(http.MethodPut, remoteAccountsURI(
						parent.Name, "toto"), body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"partner":        parent.Name,
						"remote_account": "toto",
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the account was not found", func() {
							So(w.Body.String(), ShouldEqual, "no account 'toto' "+
								"found for partner parent\n")
						})

						Convey("Then the old account should still exist", func() {
							var accs model.RemoteAccounts
							So(db.Select(&accs).Run(), ShouldBeNil)
							So(accs, ShouldNotBeEmpty)
							So(accs[0], ShouldResemble, old)
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPut, remoteAccountsURI(
						"toto", old.Login), body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{
						"partner":        "toto",
						"remote_account": old.Login,
					})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the account was not found", func() {
							So(w.Body.String(), ShouldEqual, "partner 'toto' not found\n")
						})

						Convey("Then the old account should still exist", func() {
							var accs model.RemoteAccounts
							So(db.Select(&accs).Run(), ShouldBeNil)
							So(accs, ShouldNotBeEmpty)
							So(accs[0], ShouldResemble, old)
						})
					})
				})
			})
		})
	})
}
