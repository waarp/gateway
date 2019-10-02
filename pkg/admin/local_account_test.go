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
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const localAccountsURI = APIPath + LocalAccountsPath + "/"

func TestGetLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_get_test")

	Convey("Given the account get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			expected := model.LocalAccount{
				Login:        "existing",
				LocalAgentID: parent.ID,
				Password:     []byte("existing"),
			}
			err = db.Create(&expected)
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

						exp, err := json.Marshal(&expected)

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
	logger := log.NewLogger("rest_account_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]model.LocalAccount) {
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

			for i := range expected["localAccounts"] {
				err := expected["localAccounts"][i].BeforeInsert(nil)
				So(err, ShouldBeNil)
			}
			exp, err := json.Marshal(expected)
			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldResemble, string(exp)+"\n")
		})
	}

	Convey("Given the account listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listLocalAccounts(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]model.LocalAccount{}

		Convey("Given a database with 4 accounts", func() {
			parent1 := model.LocalAgent{
				Name:        "parent1",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			parent2 := model.LocalAgent{
				Name:        "parent2",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent1)
			So(err, ShouldBeNil)
			err = db.Create(&parent2)
			So(err, ShouldBeNil)

			account1 := model.LocalAccount{
				Login:        "account1",
				Password:     []byte("account1"),
				LocalAgentID: parent1.ID,
			}
			account2 := model.LocalAccount{
				Login:        "account2",
				Password:     []byte("account2"),
				LocalAgentID: parent2.ID,
			}
			account3 := model.LocalAccount{
				Login:        "account3",
				Password:     []byte("account3"),
				LocalAgentID: parent1.ID,
			}
			account4 := model.LocalAccount{
				Login:        "account4",
				Password:     []byte("account4"),
				LocalAgentID: parent2.ID,
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
				r, err := http.NewRequest(http.MethodGet, localAccountsURI, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAccounts"] = []model.LocalAccount{account1,
						account2, account3, account4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+
					"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAccounts"] = []model.LocalAccount{account1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+
					"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAccounts"] = []model.LocalAccount{account2,
						account3, account4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+
					"?sortby=login&order=desc", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAccounts"] = []model.LocalAccount{account4,
						account3, account2, account1}
					check(w, expected)
				})
			})

			Convey("Given a request with an agent parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAccountsURI+
					"?agent="+fmt.Sprint(parent1.ID), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAccounts"] = []model.LocalAccount{account1,
						account3}
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

		Convey("Given a database with 1 account", func() {
			parent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.LocalAccount{
				Login:        "existing",
				Password:     []byte("existing"),
				LocalAgentID: parent.ID,
			}
			err = db.Create(&existing)
			So(err, ShouldBeNil)

			Convey("Given a new account to insert in the database", func() {
				newAccount := model.LocalAccount{
					Login:        "new_account",
					Password:     []byte("new_account"),
					LocalAgentID: parent.ID,
				}

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

							err := db.Get(&newAccount)
							So(err, ShouldBeNil)

							err = bcrypt.CompareHashAndPassword(newAccount.Password, clearPwd)
							So(err, ShouldBeNil)
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
					r, err := http.NewRequest(http.MethodPost, localAccountsURI,
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
					r, err := http.NewRequest(http.MethodPost, localAccountsURI,
						bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message "+
							"stating that the login already exist", func() {

							So(w.Body.String(), ShouldEqual, "A local account "+
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

				Convey("Given that the new account's local agent ID type is not "+
					"a valid one", func() {
					newAccount.LocalAgentID = 1000

					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, localAccountsURI,
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
								"No local agent found with the ID '1000'\n")
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

func TestDeleteLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_delete_test")

	Convey("Given the account deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.LocalAccount{
				Login:        "existing",
				Password:     []byte("existing"),
				LocalAgentID: parent.ID,
			}

			err = db.Create(&existing)
			So(err, ShouldBeNil)

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

						exist, err := db.Exists(&existing)
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
			parent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.LocalAccount{
				Login:        "old",
				Password:     []byte("old"),
				LocalAgentID: parent.ID,
			}
			other := model.LocalAccount{
				Login:        "other",
				Password:     []byte("other"),
				LocalAgentID: parent.ID,
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

					expected := model.LocalAccount{
						Login:        update.Login,
						Password:     nil,
						LocalAgentID: old.ID,
					}

					checkValidUpdate(db, w, http.MethodPatch, localAccountsURI,
						id, "local_account", body, handler, &old, &expected)
				})

				Convey("Given an already existing login", func() {
					update := struct{ Login string }{Login: other.Login}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "A local account with the same login '" + update.Login +
						"' already exist\n"
					checkInvalidUpdate(db, handler, w, body, localAccountsURI,
						id, "local_account", &old, msg)
				})

				Convey("Given an invalid new partner ID", func() {
					update := struct{ LocalAgentID uint64 }{LocalAgentID: 1000}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "No local agent found with the ID '1000'\n"
					checkInvalidUpdate(db, handler, w, body, localAccountsURI,
						id, "local_account", &old, msg)
				})
			})
		})
	})
}

func TestReplaceLocalAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_replace_logger")

	Convey("Given the account replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateLocalAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 accounts", func() {
			parent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.LocalAccount{
				Login:        "old",
				Password:     []byte("old"),
				LocalAgentID: parent.ID,
			}
			other := model.LocalAccount{
				Login:        "other",
				Password:     []byte("other"),
				LocalAgentID: parent.ID,
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given a valid new account", func() {
				replace := struct {
					Login        string
					Password     []byte
					LocalAgentID uint64
				}{
					Login:        "replace",
					Password:     []byte("replace"),
					LocalAgentID: parent.ID,
				}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.LocalAccount{
					Login:        replace.Login,
					Password:     nil,
					LocalAgentID: replace.LocalAgentID,
				}

				checkValidUpdate(db, w, http.MethodPut, localAccountsURI,
					id, "local_account", body, handler, &old, &expected)
			})

			Convey("Given a non-existing account ID parameter", func() {
				replace := struct{}{}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPut, localAccountsURI+
						"1000", bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_account": "1000"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}
