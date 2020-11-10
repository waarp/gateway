package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func localAccountsURI(agent, login string) string {
	return fmt.Sprintf("http://localhost:8080/api/servers/%s/accounts/%s", agent, login)
}

func TestGetLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_get_test")

	Convey("Given the account get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			expected := &model.LocalAccount{
				Login:        "existing",
				LocalAgentID: parent.ID,
				Password:     []byte("existing"),
			}
			So(db.Create(expected), ShouldBeNil)

			Convey("Given a request with a valid account login parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name,
					"local_account": expected.Login})

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

						exp, err := json.Marshal(FromLocalAccount(expected, &AuthorizedRules{}))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing account login parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name,
					"local_account": "toto"})

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
				r = mux.SetURLVars(r, map[string]string{"local_agent": "toto",
					"local_account": expected.Login})

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

func TestListLocalAccounts(t *testing.T) {
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

	Convey("Given the local account listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listLocalAccounts(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutAccount{}

		Convey("Given a database with 4 local accounts", func() {
			p1 := &model.LocalAgent{
				Name:        "parent1",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			p2 := &model.LocalAgent{
				Name:        "parent2",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			So(db.Create(p1), ShouldBeNil)
			So(db.Create(p2), ShouldBeNil)

			a1 := &model.LocalAccount{
				Login:        "account1",
				Password:     []byte("account1"),
				LocalAgentID: p1.ID,
			}
			a2 := &model.LocalAccount{
				Login:        "account2",
				Password:     []byte("account2"),
				LocalAgentID: p1.ID,
			}
			a3 := &model.LocalAccount{
				Login:        "account3",
				Password:     []byte("account3"),
				LocalAgentID: p2.ID,
			}
			a4 := &model.LocalAccount{
				Login:        "account4",
				Password:     []byte("account4"),
				LocalAgentID: p1.ID,
			}

			So(db.Create(a1), ShouldBeNil)
			So(db.Create(a2), ShouldBeNil)
			So(db.Create(a3), ShouldBeNil)
			So(db.Create(a4), ShouldBeNil)

			account1 := *FromLocalAccount(a1, &AuthorizedRules{})
			account2 := *FromLocalAccount(a2, &AuthorizedRules{})
			account3 := *FromLocalAccount(a3, &AuthorizedRules{})
			account4 := *FromLocalAccount(a4, &AuthorizedRules{})

			Convey("Given a request with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account1, account2,
						account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a different agent", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": p2.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account3}

					check(w, expected)
				})
			})

			Convey("Given a request with an invalid agent", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": "toto"})

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
				r = mux.SetURLVars(r, map[string]string{"local_agent": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account1}

					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account2, account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a sort parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=login-", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account4, account2,
						account1}

					check(w, expected)
				})
			})
		})
	})
}

func TestCreateLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_create_logger")

	Convey("Given the account creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			parent := &model.LocalAgent{
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
					r, err := http.NewRequest(http.MethodPost, localAccountsURI(
						parent.Name, ""), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the "+
							"URI of the new account", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, localAccountsURI(parent.Name,
								"new_account"))
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new account should be inserted in the "+
							"database", func() {

							var accs []model.LocalAccount
							So(db.Select(&accs, nil), ShouldBeNil)
							So(len(accs), ShouldEqual, 1)

							So(bcrypt.CompareHashAndPassword(accs[0].Password,
								[]byte("new_password")), ShouldBeNil)
							So(accs[0], ShouldResemble, model.LocalAccount{
								ID:           1,
								LocalAgentID: parent.ID,
								Login:        "new_account",
								Password:     accs[0].Password,
							})
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, localAccountsURI(
						"toto", ""), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": "toto"})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the account was not found", func() {
							So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
						})

						Convey("Then the new account should NOT exist", func() {
							var accs []model.LocalAccount
							So(db.Select(&accs, nil), ShouldBeNil)
							So(accs, ShouldBeEmpty)
						})
					})
				})
			})
		})
	})
}

func TestDeleteLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_delete_test")

	Convey("Given the account deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			existing := &model.LocalAccount{
				Login:        "existing",
				Password:     []byte("existing"),
				LocalAgentID: parent.ID,
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a request with the valid account login parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name,
					"local_account": existing.Login})

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
						var a []model.LocalAccount
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})

			Convey("Given a request with a non-existing account login parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name,
					"local_account": "toto"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that the account "+
						"was not found", func() {
						So(w.Body.String(), ShouldEqual, "no account 'toto' found "+
							"for server parent\n")
					})
				})
			})

			Convey("Given a request with a non-existing agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": "toto",
					"local_account": existing.Login})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that the account "+
						"was not found", func() {
						So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
					})
				})
			})
		})
	})
}

func TestUpdateLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_update_logger")

	Convey("Given the account updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			old := &model.LocalAccount{
				Login:        "old",
				Password:     []byte("old"),
				LocalAgentID: parent.ID,
			}
			So(db.Create(old), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				body := []byte(`{
					"password": "upd_password"
				}`)

				Convey("Given a valid account login", func() {
					r, err := http.NewRequest(http.MethodPatch, localAgentsURI+
						parent.Name+"/accounts/"+old.Login, bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name,
						"local_account": old.Login})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the 'Location' header should contain "+
						"the URI of the updated account", func() {

						location := w.Header().Get("Location")
						u, _ := url.QueryUnescape(localAccountsURI(parent.Name,
							old.Login))
						So(location, ShouldEqual, u)
					})

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the account should have been updated", func() {
						var res []model.LocalAccount
						So(db.Select(&res, nil), ShouldBeNil)
						So(len(res), ShouldEqual, 1)

						So(bcrypt.CompareHashAndPassword(res[0].Password,
							[]byte("upd_password")), ShouldBeNil)
						So(res[0], ShouldResemble, model.LocalAccount{
							ID:           old.ID,
							LocalAgentID: parent.ID,
							Login:        "old",
							Password:     res[0].Password,
						})
					})
				})

				Convey("Given an invalid account login", func() {
					r, err := http.NewRequest(http.MethodPatch, localAccountsURI(
						parent.Name, "toto"), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name,
						"local_account": "toto"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the account was not found", func() {
						So(w.Body.String(), ShouldEqual, "no account 'toto' found "+
							"for server parent\n")
					})

					Convey("Then the old account should still exist", func() {
						So(db.Get(old), ShouldBeNil)
					})
				})

				Convey("Given an invalid agent name", func() {
					r, err := http.NewRequest(http.MethodPatch, localAccountsURI(
						"toto", old.Login), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": "toto",
						"local_account": old.Login})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the account was not found", func() {
						So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
					})

					Convey("Then the old account should still exist", func() {
						So(db.Get(old), ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestReplaceLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_update_logger")

	Convey("Given the account updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := replaceLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(parent), ShouldBeNil)

			old := &model.LocalAccount{
				Login:        "old",
				Password:     []byte("old"),
				LocalAgentID: parent.ID,
			}
			So(db.Create(old), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				body := []byte(`{
					"login": "upd_login",
					"password": "upd_password"
				}`)

				Convey("Given a valid account login", func() {
					r, err := http.NewRequest(http.MethodPut, localAccountsURI(
						parent.Name, old.Login), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name,
						"local_account": old.Login})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the 'Location' header should contain "+
						"the URI of the updated account", func() {

						location := w.Header().Get("Location")
						So(location, ShouldEqual, localAccountsURI("parent", "upd_login"))
					})

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the account should have been updated", func() {
						var res []model.LocalAccount
						So(db.Select(&res, nil), ShouldBeNil)
						So(len(res), ShouldEqual, 1)

						So(bcrypt.CompareHashAndPassword(res[0].Password,
							[]byte("upd_password")), ShouldBeNil)
						So(res[0], ShouldResemble, model.LocalAccount{
							ID:           old.ID,
							LocalAgentID: parent.ID,
							Login:        "upd_login",
							Password:     res[0].Password,
						})
					})
				})

				Convey("Given an invalid account login", func() {
					r, err := http.NewRequest(http.MethodPut, localAccountsURI(
						parent.Name, "toto"), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": parent.Name,
						"local_account": "toto"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the account was not found", func() {
						So(w.Body.String(), ShouldEqual, "no account 'toto' found "+
							"for server parent\n")
					})

					Convey("Then the old account should still exist", func() {
						So(db.Get(old), ShouldBeNil)
					})
				})

				Convey("Given an invalid agent name", func() {
					r, err := http.NewRequest(http.MethodPut, localAccountsURI(
						"toto", old.Login), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": "toto",
						"local_account": old.Login})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the account was not found", func() {
						So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
					})

					Convey("Then the old account should still exist", func() {
						So(db.Get(old), ShouldBeNil)
					})
				})
			})
		})
	})
}
