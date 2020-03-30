package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
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
				ProtoConfig: []byte(`{}`),
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

						exp, err := json.Marshal(FromLocalAccount(expected))

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
				ProtoConfig: []byte(`{}`),
			}
			p2 := &model.LocalAgent{
				Name:        "parent2",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
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

			account1 := *FromLocalAccount(a1)
			account2 := *FromLocalAccount(a2)
			account3 := *FromLocalAccount(a3)
			account4 := *FromLocalAccount(a4)

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
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(parent), ShouldBeNil)

			Convey("Given a new account to insert in the database", func() {
				newAccount := &InAccount{
					Login:    "new_account",
					Password: []byte("new_account"),
				}
				body, err := json.Marshal(newAccount)
				So(err, ShouldBeNil)

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
							So(location, ShouldEqual, localAccountsURI(
								parent.Name, newAccount.Login))
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new account should be inserted in the "+
							"database", func() {
							clearPwd := newAccount.Password
							newAccount.Password = nil

							test := newAccount.ToLocal(parent)
							So(db.Get(test), ShouldBeNil)

							So(bcrypt.CompareHashAndPassword(test.Password, clearPwd),
								ShouldBeNil)
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
							So(w.Body.String(), ShouldEqual, "Record not found\n")
						})

						Convey("Then the new account should NOT exist", func() {
							check := newAccount.ToLocal(parent)
							So(db.Get(check), ShouldNotBeNil)
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
				ProtoConfig: []byte(`{}`),
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
						err := existing.BeforeInsert(nil)
						So(err, ShouldBeNil)

						exist, err := db.Exists(existing)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
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
						So(w.Body.String(), ShouldEqual, "Record not found\n")
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
						So(w.Body.String(), ShouldEqual, "Record not found\n")
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

		Convey("Given a database with 2 accounts", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(parent), ShouldBeNil)

			old := &model.LocalAccount{
				Login:        "old",
				Password:     []byte("old"),
				LocalAgentID: parent.ID,
			}
			other := &model.LocalAccount{
				Login:        "other",
				Password:     []byte("other"),
				LocalAgentID: parent.ID,
			}
			So(db.Create(old), ShouldBeNil)
			So(db.Create(other), ShouldBeNil)

			Convey("Given new values to update the account with", func() {
				update := InAccount{
					Login:    "update",
					Password: []byte("update"),
				}
				body, err := json.Marshal(update)
				So(err, ShouldBeNil)

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
						So(location, ShouldEqual, localAgentsURI+parent.Name+
							"/accounts/"+update.Login)
					})

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the account should have been updated", func() {
						result := &model.LocalAccount{ID: old.ID}
						So(db.Get(result), ShouldBeNil)

						So(result.Login, ShouldEqual, update.Login)
						So(result.LocalAgentID, ShouldEqual, parent.ID)
						So(bcrypt.CompareHashAndPassword(result.Password,
							update.Password), ShouldBeNil)
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
						So(w.Body.String(), ShouldEqual, "Record not found\n")
					})

					Convey("Then the old account should still exist", func() {
						exist, err := db.Exists(old)

						So(err, ShouldBeNil)
						So(exist, ShouldBeTrue)
					})
				})

				Convey("Given an invalid agent name", func() {
					r, err := http.NewRequest(http.MethodPatch, localAgentsURI+
						"toto/accounts/"+update.Login, bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": "toto",
						"local_account": update.Login})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the account was not found", func() {
						So(w.Body.String(), ShouldEqual, "Record not found\n")
					})

					Convey("Then the old account should still exist", func() {
						exist, err := db.Exists(old)

						So(err, ShouldBeNil)
						So(exist, ShouldBeTrue)
					})
				})
			})
		})
	})
}
