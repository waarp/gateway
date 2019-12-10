package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const localAccountsURI = "http://localhost:8080" + APIPath + LocalAccountsPath + "/"

func TestGetLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_get_test", logConf)

	Convey("Given the account get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(parent)
			So(err, ShouldBeNil)

			expected := &model.LocalAccount{
				Login:        "existing",
				LocalAgentID: parent.ID,
				Password:     []byte("existing"),
			}
			err = db.Create(expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid account ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_account": id})

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

						exp, err := json.Marshal(fromLocalAccount(expected))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing account ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_account": "1000"})

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
	logger := log.NewLogger("rest_account_list_test", logConf)

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
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			p2 := &model.LocalAgent{
				Name:        "parent2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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
				LocalAgentID: p2.ID,
			}
			a3 := &model.LocalAccount{
				Login:        "account3",
				Password:     []byte("account3"),
				LocalAgentID: p1.ID,
			}
			a4 := &model.LocalAccount{
				Login:        "account4",
				Password:     []byte("account4"),
				LocalAgentID: p2.ID,
			}

			So(db.Create(a1), ShouldBeNil)
			So(db.Create(a2), ShouldBeNil)
			So(db.Create(a3), ShouldBeNil)
			So(db.Create(a4), ShouldBeNil)

			account1 := *fromLocalAccount(a1)
			account2 := *fromLocalAccount(a2)
			account3 := *fromLocalAccount(a3)
			account4 := *fromLocalAccount(a4)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account1, account2,
						account3, account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+
					"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account1}

					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+
					"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account2, account3, account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a sort parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+
					"?sort=login-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account4, account3,
						account2, account1}

					check(w, expected)
				})
			})

			Convey("Given a request with an agent parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+
					"?agent="+fmt.Sprint(p1.ID), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["localAccounts"] = []OutAccount{account1, account3}

					check(w, expected)
				})
			})
		})
	})
}

func TestCreateLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_create_logger", logConf)

	Convey("Given the account creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(parent), ShouldBeNil)

			Convey("Given a new account to insert in the database", func() {
				newAccount := &InAccount{
					Login:    "new_account",
					Password: []byte("new_account"),
					AgentID:  parent.ID,
				}
				So(newAccount.AgentID, ShouldNotBeZeroValue)

				Convey("Given that the new account is valid for insertion", func() {
					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, localAccountsURI,
						bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the "+
							"URI of the new account", func() {

							location := w.Header().Get("Location")
							So(location, ShouldStartWith, localAccountsURI)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new account should be inserted in the "+
							"database", func() {
							clearPwd := newAccount.Password
							newAccount.Password = nil

							test := newAccount.toLocal()
							err := db.Get(test)
							So(err, ShouldBeNil)

							err = bcrypt.CompareHashAndPassword(test.Password, clearPwd)
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

func TestDeleteLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_delete_test", logConf)

	Convey("Given the account deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(parent), ShouldBeNil)

			existing := &model.LocalAccount{
				Login:        "existing",
				Password:     []byte("existing"),
				LocalAgentID: parent.ID,
			}
			So(db.Create(existing), ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid account ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, localAccountsURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_account": id})

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

			Convey("Given a request with a non-existing account ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, localAccountsURI+
					"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_account": "1000"})

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
	logger := log.NewLogger("rest_account_update_logger", logConf)

	Convey("Given the account updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 accounts", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the account with", func() {

				Convey("Given a new login", func() {
					update := InAccount{
						Login:    "update",
						AgentID:  parent.ID,
						Password: []byte("update"),
					}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, localAccountsURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"local_account": id})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated account", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, localAccountsURI+id)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the account should have been updated", func() {
							result := &model.LocalAccount{ID: old.ID}
							err := db.Get(result)

							So(err, ShouldBeNil)
							So(result.Login, ShouldEqual, update.Login)
							So(result.LocalAgentID, ShouldEqual, update.AgentID)
							So(bcrypt.CompareHashAndPassword(result.Password, update.Password), ShouldBeNil)
						})
					})
				})

				Convey("Given an invalid account ID", func() {
					update := InAccount{
						Login:    "update",
						AgentID:  parent.ID,
						Password: []byte("update"),
					}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, localAccountsURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"local_account": "1000"})

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
	})
}
