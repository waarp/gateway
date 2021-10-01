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
	"golang.org/x/crypto/bcrypt"

	. "code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func localAccountsURI(agent, login string) string {
	return fmt.Sprintf("http://localhost:8080/api/servers/%s/accounts/%s", agent, login)
}

func TestGetLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_get_test")

	Convey("Given the account get handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := getLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			expected := &model.LocalAccount{
				Login:        "existing",
				LocalAgentID: parent.ID,
				PasswordHash: hash("existing"),
			}
			So(db.Insert(expected).Run(), ShouldBeNil)

			Convey("Given a request with a valid account login parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{
					"server":        parent.Name,
					"local_account": expected.Login,
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

					Convey("Then the body should contain the requested server "+
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
				r = mux.SetURLVars(r, map[string]string{
					"server":        parent.Name,
					"local_account": "toto",
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
					"server":        "toto",
					"local_account": expected.Login,
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

	Convey("Given the local account listing handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := listLocalAccounts(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutAccount{}

		Convey("Given a database with 4 local accounts", func() {
			p1 := &model.LocalAgent{
				Name:        "parent1",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			p2 := &model.LocalAgent{
				Name:        "parent2",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			So(db.Insert(p1).Run(), ShouldBeNil)
			So(db.Insert(p2).Run(), ShouldBeNil)

			a1 := &model.LocalAccount{
				Login:        "account1",
				PasswordHash: hash("account1"),
				LocalAgentID: p1.ID,
			}
			a2 := &model.LocalAccount{
				Login:        "account2",
				PasswordHash: hash("account2"),
				LocalAgentID: p1.ID,
			}
			a3 := &model.LocalAccount{
				Login:        "account3",
				PasswordHash: hash("account3"),
				LocalAgentID: p2.ID,
			}
			a4 := &model.LocalAccount{
				Login:        "account4",
				PasswordHash: hash("account4"),
				LocalAgentID: p1.ID,
			}

			So(db.Insert(a1).Run(), ShouldBeNil)
			So(db.Insert(a2).Run(), ShouldBeNil)
			So(db.Insert(a3).Run(), ShouldBeNil)
			So(db.Insert(a4).Run(), ShouldBeNil)

			account1 := *FromLocalAccount(a1, &AuthorizedRules{})
			account2 := *FromLocalAccount(a2, &AuthorizedRules{})
			account3 := *FromLocalAccount(a3, &AuthorizedRules{})
			account4 := *FromLocalAccount(a4, &AuthorizedRules{})

			Convey("Given a request with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{
						account1, account2,
						account4,
					}

					check(w, expected)
				})
			})

			Convey("Given a request with a different agent", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": p2.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account3}

					check(w, expected)
				})
			})

			Convey("Given a request with an invalid agent", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": "toto"})

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
				r = mux.SetURLVars(r, map[string]string{"server": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account1}

					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account2, account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a sort parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=login-", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": p1.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{
						account4, account2,
						account1,
					}

					check(w, expected)
				})
			})
		})
	})
}

func TestCreateLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_create_logger")

	Convey("Given the account creation handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := addLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			Convey("Given a new account to insert in the database", func() {
				body := strings.NewReader(`{
					"login": "new_account",
					"password": "new_password"
				}`)

				Convey("Given a valid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPost, localAccountsURI(
						parent.Name, ""), body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"server": parent.Name})

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
							var accs model.LocalAccounts
							So(db.Select(&accs).Run(), ShouldBeNil)
							So(len(accs), ShouldEqual, 1)

							So(bcrypt.CompareHashAndPassword(accs[0].PasswordHash,
								[]byte("new_password")), ShouldBeNil)
							So(accs[0], ShouldResemble, model.LocalAccount{
								ID:           1,
								LocalAgentID: parent.ID,
								Login:        "new_account",
								PasswordHash: accs[0].PasswordHash,
							})
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, localAccountsURI(
						"toto", ""), body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"server": "toto"})

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
							var accs model.LocalAccounts
							So(db.Select(&accs).Run(), ShouldBeNil)
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

	Convey("Given the account deletion handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := deleteLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			existing := &model.LocalAccount{
				Login:        "existing",
				PasswordHash: hash("existing"),
				LocalAgentID: parent.ID,
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a request with the valid account login parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{
					"server":        parent.Name,
					"local_account": existing.Login,
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
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldBeEmpty)
					})
				})
			})

			Convey("Given a request with a non-existing account login parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{
					"server":        parent.Name,
					"local_account": "toto",
				})

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
				r = mux.SetURLVars(r, map[string]string{
					"server":        "toto",
					"local_account": existing.Login,
				})

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

	Convey("Given the account updating handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := updateLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			old := &model.LocalAccount{
				Login:        "old",
				PasswordHash: hash("old"),
				LocalAgentID: parent.ID,
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				body := strings.NewReader(`{
					"password": "upd_password"
				}`)

				Convey("Given a valid account login", func() {
					//nolint:noctx // this is a test
					r, err := http.NewRequest(http.MethodPatch, testServersURI+
						parent.Name+"/accounts/"+old.Login, body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{
						"server":        parent.Name,
						"local_account": old.Login,
					})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the 'Location' header should contain "+
						"the URI of the updated account", func() {
						location := w.Header().Get("Location")
						So(location, ShouldEqual, localAccountsURI(parent.Name, old.Login))
					})

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the account should have been updated", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(len(accounts), ShouldEqual, 1)

						So(bcrypt.CompareHashAndPassword(accounts[0].PasswordHash,
							[]byte("upd_password")), ShouldBeNil)
						So(accounts[0], ShouldResemble, model.LocalAccount{
							ID:           old.ID,
							LocalAgentID: parent.ID,
							Login:        "old",
							PasswordHash: accounts[0].PasswordHash,
						})
					})
				})

				Convey("Given an invalid account login", func() {
					r, err := http.NewRequest(http.MethodPatch, localAccountsURI(
						parent.Name, "toto"), body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{
						"server":        parent.Name,
						"local_account": "toto",
					})

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
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(*old, ShouldBeIn, accounts)
					})
				})

				Convey("Given an invalid agent name", func() {
					r, err := http.NewRequest(http.MethodPatch, localAccountsURI(
						"toto", old.Login), body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{
						"server":        "toto",
						"local_account": old.Login,
					})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the account was not found", func() {
						So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
					})

					Convey("Then the old account should still exist", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(*old, ShouldBeIn, accounts)
					})
				})
			})
		})
	})
}

func TestReplaceLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_update_logger")

	Convey("Given the account updating handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := replaceLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(parent).Run(), ShouldBeNil)

			old := &model.LocalAccount{
				Login:        "old",
				PasswordHash: hash("old"),
				LocalAgentID: parent.ID,
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				body := strings.NewReader(`{
					"login": "upd_login",
					"password": "upd_password"
				}`)

				Convey("Given a valid account login", func() {
					//nolint:noctx // this is a test
					r, err := http.NewRequest(http.MethodPut, localAccountsURI(
						parent.Name, old.Login), body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{
						"server":        parent.Name,
						"local_account": old.Login,
					})

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
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(len(accounts), ShouldEqual, 1)

						So(bcrypt.CompareHashAndPassword(accounts[0].PasswordHash,
							[]byte("upd_password")), ShouldBeNil)
						So(accounts[0], ShouldResemble, model.LocalAccount{
							ID:           old.ID,
							LocalAgentID: parent.ID,
							Login:        "upd_login",
							PasswordHash: accounts[0].PasswordHash,
						})
					})
				})

				Convey("Given an invalid account login", func() {
					r, err := http.NewRequest(http.MethodPut, localAccountsURI(
						parent.Name, "toto"), body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{
						"server":        parent.Name,
						"local_account": "toto",
					})

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
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(*old, ShouldBeIn, accounts)
					})
				})

				Convey("Given an invalid agent name", func() {
					//nolint:noctx // this is a test
					r, err := http.NewRequest(http.MethodPut, localAccountsURI(
						"toto", old.Login), body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{
						"server":        "toto",
						"local_account": old.Login,
					})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the account was not found", func() {
						So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
					})

					Convey("Then the old account should still exist", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(*old, ShouldBeIn, accounts)
					})
				})
			})
		})
	})
}
