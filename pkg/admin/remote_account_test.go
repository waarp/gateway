package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const remoteAccountsURI = APIPath + RemoteAccountsPath + "/"

func TestGetRemoteAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_get_test")

	Convey("Given the account get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			expected := model.RemoteAccount{
				Login:         "existing",
				RemoteAgentID: parent.ID,
				Password:      []byte("existing"),
			}
			err = db.Create(&expected)
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

						exp, err := json.Marshal(&expected)

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

	check := func(w *httptest.ResponseRecorder, expected map[string][]model.RemoteAccount) {
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

			for i := range expected["remoteAccounts"] {
				err := expected["remoteAccounts"][i].BeforeInsert(nil)
				So(err, ShouldBeNil)
			}
			exp, err := json.Marshal(expected)
			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldResemble, string(exp)+"\n")
		})
	}

	Convey("Given the account listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listRemoteAccounts(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]model.RemoteAccount{}

		Convey("Given a database with 4 accounts", func() {
			parent1 := model.RemoteAgent{
				Name:        "parent1",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			parent2 := model.RemoteAgent{
				Name:        "parent2",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent1)
			So(err, ShouldBeNil)
			err = db.Create(&parent2)
			So(err, ShouldBeNil)

			account1 := model.RemoteAccount{
				Login:         "account1",
				Password:      []byte("account1"),
				RemoteAgentID: parent1.ID,
			}
			account2 := model.RemoteAccount{
				Login:         "account2",
				Password:      []byte("account2"),
				RemoteAgentID: parent2.ID,
			}
			account3 := model.RemoteAccount{
				Login:         "account3",
				Password:      []byte("account3"),
				RemoteAgentID: parent1.ID,
			}
			account4 := model.RemoteAccount{
				Login:         "account4",
				Password:      []byte("account4"),
				RemoteAgentID: parent2.ID,
			}

			err = db.Create(&account1)
			So(err, ShouldBeNil)
			err = db.Create(&account2)
			So(err, ShouldBeNil)
			err = db.Create(&account3)
			So(err, ShouldBeNil)
			err = db.Create(&account4)
			So(err, ShouldBeNil)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []model.RemoteAccount{account1,
						account2, account3, account4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+
					"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []model.RemoteAccount{account1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+
					"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []model.RemoteAccount{account2,
						account3, account4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+
					"?sortby=login&order=desc", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []model.RemoteAccount{account4,
						account3, account2, account1}
					check(w, expected)
				})
			})

			Convey("Given a request with an agent parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAccountsURI+
					"?agent="+fmt.Sprint(parent1.ID), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["remoteAccounts"] = []model.RemoteAccount{account1,
						account3}
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

		Convey("Given a database with 1 account", func() {
			parent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.RemoteAccount{
				Login:         "existing",
				Password:      []byte("existing"),
				RemoteAgentID: parent.ID,
			}
			err = db.Create(&existing)
			So(err, ShouldBeNil)

			Convey("Given a new account to insert in the database", func() {
				newAccount := model.RemoteAccount{
					Login:         "new_account",
					Password:      []byte("new_account"),
					RemoteAgentID: parent.ID,
				}

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

							err := db.Get(&newAccount)
							So(err, ShouldBeNil)

							storedPwd, err := model.DecryptPassword(newAccount.Password)
							So(err, ShouldBeNil)
							So(string(storedPwd), ShouldEqual, string(clearPwd))
						})

						Convey("Then the existing account should still be "+
							"present as well", func() {
							err := newAccount.BeforeInsert(nil)
							So(err, ShouldBeNil)

							exist, err := db.Exists(&existing)
							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the new account has an ID", func() {
					newAccount.ID = existing.ID

					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, remoteAccountsURI,
						bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error",
							func() {
								So(w.Code, ShouldEqual, http.StatusBadRequest)
							})

						Convey("Then the response body should contain a message "+
							"stating that the ID cannot be entered manually", func() {

							So(w.Body.String(), ShouldEqual,
								"The account's ID cannot be entered manually\n")
						})

						Convey("Then the new account should NOT be inserted in "+
							"the database", func() {
							err := newAccount.BeforeInsert(nil)
							So(err, ShouldBeNil)

							exist, err := db.Exists(&newAccount)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new account's login already exist", func() {
					newAccount.Login = existing.Login

					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, remoteAccountsURI,
						bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message "+
							"stating that the login already exist", func() {

							So(w.Body.String(), ShouldEqual, "A remote account "+
								"with the same login '"+newAccount.Login+
								"' already exist\n")
						})

						Convey("Then the new account should NOT be inserted in "+
							"the database", func() {
							err := newAccount.BeforeInsert(nil)
							So(err, ShouldBeNil)

							exist, err := db.Exists(&newAccount)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new account's remote agent ID type is not "+
					"a valid one", func() {
					newAccount.RemoteAgentID = 1000

					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, remoteAccountsURI,
						bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error",
							func() {
								So(w.Code, ShouldEqual, http.StatusBadRequest)
							})

						Convey("Then the response body should contain a message "+
							"stating that the partnerID is not valid", func() {

							So(w.Body.String(), ShouldEqual,
								"No remote agent found with the ID '1000'\n")
						})

						Convey("Then the new account should NOT be inserted in "+
							"the database", func() {
							err := newAccount.BeforeInsert(nil)
							So(err, ShouldBeNil)

							exist, err := db.Exists(&newAccount)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
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
			parent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.RemoteAccount{
				Login:         "existing",
				Password:      []byte("existing"),
				RemoteAgentID: parent.ID,
			}

			err = db.Create(&existing)
			So(err, ShouldBeNil)

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

						exist, err := db.Exists(&existing)
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
			parent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.RemoteAccount{
				Login:         "old",
				Password:      []byte("old"),
				RemoteAgentID: parent.ID,
			}
			other := model.RemoteAccount{
				Login:         "other",
				Password:      []byte("other"),
				RemoteAgentID: parent.ID,
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the account with", func() {

				Convey("Given a new login", func() {
					update := struct{ Login string }{Login: "update"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					expected := model.RemoteAccount{
						Login:         update.Login,
						Password:      nil,
						RemoteAgentID: old.ID,
					}

					checkValidUpdate(db, w, http.MethodPatch, remoteAccountsURI,
						id, "remote_account", body, handler, &old, &expected)
				})

				Convey("Given an already existing login", func() {
					update := struct{ Login string }{Login: other.Login}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "A remote account with the same login '" + update.Login +
						"' already exist\n"
					checkInvalidUpdate(db, handler, w, body, remoteAccountsURI,
						id, "remote_account", &old, msg)
				})

				Convey("Given an invalid new partner ID", func() {
					update := struct{ RemoteAgentID uint64 }{RemoteAgentID: 1000}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "No remote agent found with the ID '1000'\n"
					checkInvalidUpdate(db, handler, w, body, remoteAccountsURI,
						id, "remote_account", &old, msg)
				})
			})
		})
	})
}

func TestReplaceRemoteAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_replace_logger")

	Convey("Given the account replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateRemoteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 accounts", func() {
			parent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.RemoteAccount{
				Login:         "old",
				Password:      []byte("old"),
				RemoteAgentID: parent.ID,
			}
			other := model.RemoteAccount{
				Login:         "other",
				Password:      []byte("other"),
				RemoteAgentID: parent.ID,
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given a valid new account", func() {
				replace := struct {
					Login         string
					Password      []byte
					RemoteAgentID uint64
				}{
					Login:         "replace",
					Password:      []byte("replace"),
					RemoteAgentID: parent.ID,
				}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.RemoteAccount{
					Login:         replace.Login,
					Password:      nil,
					RemoteAgentID: replace.RemoteAgentID,
				}

				checkValidUpdate(db, w, http.MethodPut, remoteAccountsURI,
					id, "remote_account", body, handler, &old, &expected)
			})

			Convey("Given a non-existing account ID parameter", func() {
				replace := struct{}{}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPut, remoteAccountsURI+
						"1000", bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_account": "1000"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}
