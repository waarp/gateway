package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
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

const remoteAccountsURI = "http://localhost:8080" + APIPath + RemoteAccountsPath + "/"

func TestGetRemoteAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_get_test")

	Convey("Given the account get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"remotehost","port":2022,"root":"toto"}`),
			}
			err := db.Create(parent)
			So(err, ShouldBeNil)

			expected := &model.RemoteAccount{
				Login:         "existing",
				RemoteAgentID: parent.ID,
				Password:      []byte("existing"),
			}
			err = db.Create(expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid account ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_account": id})

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

						exp, err := json.Marshal(FromRemoteAccount(expected))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing account ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_account": "1000"})

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
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"remotehost","port":2022,"root":"toto"}`),
			}
			p2 := &model.RemoteAgent{
				Name:        "parent2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"remotehost","port":2022,"root":"toto"}`),
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
				RemoteAgentID: p2.ID,
			}
			a3 := &model.RemoteAccount{
				Login:         "account3",
				Password:      []byte("account3"),
				RemoteAgentID: p1.ID,
			}
			a4 := &model.RemoteAccount{
				Login:         "account4",
				Password:      []byte("account4"),
				RemoteAgentID: p2.ID,
			}

			So(db.Create(a1), ShouldBeNil)
			So(db.Create(a2), ShouldBeNil)
			So(db.Create(a3), ShouldBeNil)
			So(db.Create(a4), ShouldBeNil)

			account1 := *FromRemoteAccount(a1)
			account2 := *FromRemoteAccount(a2)
			account3 := *FromRemoteAccount(a3)
			account4 := *FromRemoteAccount(a4)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account1, account2,
						account3, account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+
					"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account1}

					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+
					"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account2, account3, account4}

					check(w, expected)
				})
			})

			Convey("Given a request with a sort parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+
					"?sort=login-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account4, account3,
						account2, account1}

					check(w, expected)
				})
			})

			Convey("Given a request with an agent parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+
					"?agent="+fmt.Sprint(p1.ID), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["remoteAccounts"] = []OutAccount{account1, account3}

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
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"remotehost","port":2022,"root":"toto"}`),
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
					r, err := http.NewRequest(http.MethodPost, remoteAccountsURI,
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
							So(location, ShouldStartWith, remoteAccountsURI)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new account should be inserted in the "+
							"database", func() {
							clearPwd := newAccount.Password
							newAccount.Password = nil

							test := newAccount.ToRemote()
							err := db.Get(test)
							So(err, ShouldBeNil)

							pwd, err := model.DecryptPassword(test.Password)
							So(err, ShouldBeNil)
							So(string(pwd), ShouldEqual, string(clearPwd))
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
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"remotehost","port":2022,"root":"toto"}`),
			}
			So(db.Create(parent), ShouldBeNil)

			existing := &model.RemoteAccount{
				Login:         "existing",
				Password:      []byte("existing"),
				RemoteAgentID: parent.ID,
			}
			So(db.Create(existing), ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid account ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, remoteAccountsURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_account": id})

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
				r, err := http.NewRequest(http.MethodDelete, remoteAccountsURI+
					"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_account": "1000"})

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

		Convey("Given a database with 2 accounts", func() {
			parent := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"remotehost","port":2022,"root":"toto"}`),
			}
			So(db.Create(parent), ShouldBeNil)

			old := &model.RemoteAccount{
				Login:         "old",
				Password:      []byte("old"),
				RemoteAgentID: parent.ID,
			}
			other := &model.RemoteAccount{
				Login:         "other",
				Password:      []byte("other"),
				RemoteAgentID: parent.ID,
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
						r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"remote_account": id})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated account", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, remoteAccountsURI+id)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the account should have been updated", func() {
							result := &model.RemoteAccount{ID: old.ID}
							err := db.Get(result)

							So(err, ShouldBeNil)
							So(result.Login, ShouldEqual, update.Login)
							So(result.RemoteAgentID, ShouldEqual, update.AgentID)
							pwd, err := model.DecryptPassword(result.Password)
							So(err, ShouldBeNil)
							So(string(pwd), ShouldEqual, string(update.Password))
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
						r, err := http.NewRequest(http.MethodPatch, remoteAccountsURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"remote_account": "1000"})

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
