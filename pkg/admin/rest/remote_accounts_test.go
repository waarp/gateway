package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func remoteAccountsURI(agent, login string) string {
	return fmt.Sprintf("http://localhost:8080/api/partners/%s/accounts/%s", agent, login)
}

func TestGetRemoteAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_get_test")

	Convey("Given the account get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			expected := &model.RemoteAccount{
				Login:         "existing",
				RemoteAgentID: parent.ID,
				Password:      []byte("existing"),
			}
			So(db.Create(expected), ShouldBeNil)

			Convey("Given a request with the valid account login parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
					"remote_account": expected.Login})

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

						exp, err := json.Marshal(FromRemoteAccount(expected, &AuthorizedRules{}))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing account login parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
					"remote_account": "toto"})

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
				r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto",
					"remote_account": expected.Login})

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
	logger := log.NewLogger("rest_account_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]OutAccount) {
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

	Convey("Given the remote account listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listRemoteAccounts(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutAccount{}

		Convey("Given a database with 4 remote accounts", func() {
			p1 := &model.RemoteAgent{
				Name:        "parent1",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			p2 := &model.RemoteAgent{
				Name:        "parent2",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			So(db.Create(p1), ShouldBeNil)
			So(db.Create(p2), ShouldBeNil)

			a1 := &model.RemoteAccount{
				Login:         "account1",
				Password:      []byte("account1"),
				RemoteAgentID: p1.ID,
			}
			a2 := &model.RemoteAccount{
				Login:         "account2",
				Password:      []byte("account2"),
				RemoteAgentID: p1.ID,
			}
			a3 := &model.RemoteAccount{
				Login:         "account3",
				Password:      []byte("account3"),
				RemoteAgentID: p2.ID,
			}
			a4 := &model.RemoteAccount{
				Login:         "account4",
				Password:      []byte("account4"),
				RemoteAgentID: p1.ID,
			}

			So(db.Create(a1), ShouldBeNil)
			So(db.Create(a2), ShouldBeNil)
			So(db.Create(a3), ShouldBeNil)
			So(db.Create(a4), ShouldBeNil)

			account1 := *FromRemoteAccount(a1, &AuthorizedRules{})
			account2 := *FromRemoteAccount(a2, &AuthorizedRules{})
			account3 := *FromRemoteAccount(a3, &AuthorizedRules{})
			account4 := *FromRemoteAccount(a4, &AuthorizedRules{})

			Convey("Given a request with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account1, account2,
						account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a different agent", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": p2.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account3}

					check(w, expected)
				})
			})

			Convey("Given a request with an invalid agent", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto"})

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
				r = mux.SetURLVars(r, map[string]string{"remote_agent": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account1}

					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account2, account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a sort parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=login-", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account4, account2,
						account1}

					check(w, expected)
				})
			})
		})
	})
}

func TestCreateRemoteAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_create_logger")

	Convey("Given the account creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			parent := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			Convey("Given a new account to insert in the database", func() {
				body := []byte(`{
					"login": "new_account",
					"password": "new_password"
				}`)

				Convey("Given a valid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPost, remoteAccountsURI(
						parent.Name, ""), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name})

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

							var accs []model.RemoteAccount
							So(db.Select(&accs, nil), ShouldBeNil)
							So(len(accs), ShouldEqual, 1)

							clear, err := utils.DecryptPassword(accs[0].Password)
							So(err, ShouldBeNil)
							accs[0].Password = clear
							So(accs[0], ShouldResemble, model.RemoteAccount{
								ID:            1,
								RemoteAgentID: parent.ID,
								Login:         "new_account",
								Password:      []byte("new_password"),
							})
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						"toto", ""), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto"})

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
							var accs []model.RemoteAccount
							So(db.Select(&accs, nil), ShouldBeNil)
							So(accs, ShouldBeEmpty)
						})
					})
				})
			})
		})
	})
}

func TestDeleteRemoteAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_delete_test")

	Convey("Given the account deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			existing := &model.RemoteAccount{
				Login:         "existing",
				Password:      []byte("existing"),
				RemoteAgentID: parent.ID,
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a request with the valid account login parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
					"remote_account": existing.Login})

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
						var a []model.RemoteAccount
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})

			Convey("Given a request with a non-existing account login parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
					"remote_account": "toto"})

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
				r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto",
					"remote_account": existing.Login})

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
	logger := log.NewLogger("rest_account_update_logger")

	Convey("Given the account updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			old := &model.RemoteAccount{
				Login:         "old",
				Password:      []byte("old"),
				RemoteAgentID: parent.ID,
			}
			So(db.Create(old), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				body := []byte(`{
					"password": "upd_password"
				}`)

				Convey("Given a valid account login parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						parent.Name, old.Login), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
						"remote_account": old.Login})

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
							var res []model.RemoteAccount
							So(db.Select(&res, nil), ShouldBeNil)
							So(len(res), ShouldEqual, 1)

							pswd, err := utils.DecryptPassword(res[0].Password)
							So(err, ShouldBeNil)
							So(pswd, ShouldResemble, []byte("upd_password"))

							exp := model.RemoteAccount{
								ID:            old.ID,
								RemoteAgentID: parent.ID,
								Login:         "old",
								Password:      res[0].Password,
							}
							So(res[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given an invalid account login parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						parent.Name, "toto"), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
						"remote_account": "toto"})

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
							So(db.Get(old), ShouldBeNil)
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						"toto", old.Login), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto",
						"remote_account": old.Login})

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
							So(db.Get(old), ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

func TestReplaceRemoteAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_update_logger")

	Convey("Given the account updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := replaceRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			old := &model.RemoteAccount{
				Login:         "old",
				Password:      []byte("old"),
				RemoteAgentID: parent.ID,
			}
			So(db.Create(old), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				body := []byte(`{
					"login": "upd_login",
					"password": "upd_password"
				}`)

				Convey("Given a valid account login parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI(
						parent.Name, old.Login), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
						"remote_account": old.Login})

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
							var res []model.RemoteAccount
							So(db.Select(&res, nil), ShouldBeNil)
							So(len(res), ShouldEqual, 1)

							pswd, err := utils.DecryptPassword(res[0].Password)
							So(err, ShouldBeNil)
							So(pswd, ShouldResemble, []byte("upd_password"))

							exp := model.RemoteAccount{
								ID:            old.ID,
								RemoteAgentID: parent.ID,
								Login:         "upd_login",
								Password:      res[0].Password,
							}
							So(res[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given an invalid account login parameter", func() {
					r, err := http.NewRequest(http.MethodPut, remoteAccountsURI(
						parent.Name, "toto"), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
						"remote_account": "toto"})

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
							So(db.Get(old), ShouldBeNil)
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPut, remoteAccountsURI(
						"toto", old.Login), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto",
						"remote_account": old.Login})

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
							So(db.Get(old), ShouldBeNil)
						})
					})
				})
			})
		})
	})
}
