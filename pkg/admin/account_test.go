package admin

import (
	"bytes"
	"encoding/json"
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

const accountsPath = RestURI + AccountsURI + "/"

func TestGetAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_get_test")

	Convey("Given the account get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := model.Partner{
				ID:          1,
				Name:        "parent",
				InterfaceID: 1,
				Address:     "address",
				Port:        1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			expected := model.Account{
				ID:        1,
				Username:  "existing",
				PartnerID: parent.ID,
				Password:  []byte("existing"),
			}
			err = db.Create(&expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid account ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, partnerPath+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"account": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested partner "+
						"in JSON format", func() {

						expected.Password = nil
						res := model.Account{}
						err := json.Unmarshal(w.Body.Bytes(), &res)

						So(err, ShouldBeNil)
						So(res, ShouldResemble, expected)
					})
				})
			})

			Convey("Given a request with a non-existing account ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, accountsPath+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"account": "1000"})

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

func TestListAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]model.Account) {
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

			response := map[string][]model.Account{}
			err := json.Unmarshal(w.Body.Bytes(), &response)

			for i := range expected["accounts"] {
				expected["accounts"][i].Password = nil
			}
			So(err, ShouldBeNil)
			So(response, ShouldResemble, expected)
		})
	}

	Convey("Given the account listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listAccounts(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]model.Account{}

		Convey("Given a database with 4 accounts", func() {
			parent := model.Partner{
				ID:          1,
				Name:        "parent",
				InterfaceID: 1,
				Address:     "address",
				Port:        1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			account1 := model.Account{
				ID:        1,
				Username:  "account1",
				Password:  []byte("account1"),
				PartnerID: parent.ID,
			}
			account2 := model.Account{
				ID:        2,
				Username:  "account2",
				Password:  []byte("account2"),
				PartnerID: parent.ID,
			}
			account3 := model.Account{
				ID:        3,
				Username:  "account3",
				Password:  []byte("account3"),
				PartnerID: 1000,
			}
			account4 := model.Account{
				ID:        4,
				Username:  "account4",
				Password:  []byte("account4"),
				PartnerID: parent.ID,
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
				r, err := http.NewRequest(http.MethodGet, accountsPath, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["accounts"] = []model.Account{account1, account2,
						account3, account4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, accountsPath+"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["accounts"] = []model.Account{account1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, accountsPath+"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["accounts"] = []model.Account{account2, account3, account4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, accountsPath+
					"?sortby=username&order=desc", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["accounts"] = []model.Account{account4, account3,
						account2, account1}
					check(w, expected)
				})
			})

			Convey("Given a request with a partner parameter", func() {
				r, err := http.NewRequest(http.MethodGet, accountsPath+
					"?partner=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["accounts"] = []model.Account{account1, account2, account4}
					check(w, expected)
				})
			})
		})
	})
}

func TestCreateAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_create_logger")

	Convey("Given the account creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := model.Partner{
				ID:          1,
				Name:        "parent",
				InterfaceID: 1,
				Address:     "address",
				Port:        1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.Account{
				ID:        1,
				Username:  "existing",
				Password:  []byte("existing"),
				PartnerID: parent.ID,
			}
			err = db.Create(&existing)
			So(err, ShouldBeNil)

			Convey("Given a new account to insert in the database", func() {
				newAccount := model.Account{
					ID:        2,
					Username:  "new_account",
					Password:  []byte("new_account"),
					PartnerID: parent.ID,
				}

				Convey("Given that the new account is valid for insertion", func() {
					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, accountsPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new account", func() {

							location := w.Header().Get("Location")
							expected := accountsPath + strconv.FormatUint(newAccount.ID, 10)
							So(location, ShouldEqual, expected)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new account should be inserted in the database", func() {
							newAccount.Password = nil
							exist, err := db.Exists(&newAccount)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing account should still be present as well", func() {
							existing.Password = nil
							exist, err := db.Exists(&existing)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the new account's ID already exist", func() {
					newAccount.ID = existing.ID

					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, accountsPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the ID already exist", func() {

							So(w.Body.String(), ShouldEqual,
								"An account with the same ID already exist\n")
						})

						Convey("Then the new account should NOT be inserted in the database", func() {
							newAccount.Password = nil
							exist, err := db.Exists(&newAccount)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new account's username already exist", func() {
					newAccount.Username = existing.Username

					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, accountsPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the username already exist", func() {

							So(w.Body.String(), ShouldEqual, "An account "+
								"with the same username already exist for this partner\n")
						})

						Convey("Then the new account should NOT be inserted in the database", func() {
							newAccount.Password = nil
							exist, err := db.Exists(&newAccount)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new account's partnerID type is not a valid one", func() {
					newAccount.PartnerID = 1000

					body, err := json.Marshal(newAccount)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, partnerPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the partnerID is not valid", func() {

							So(w.Body.String(), ShouldEqual,
								"No partner found with ID '1000'\n")
						})

						Convey("Then the new account should NOT be inserted in the database", func() {
							newAccount.Password = nil
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

func TestDeleteAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_delete_test")

	Convey("Given the account deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 account", func() {
			parent := model.Partner{
				ID:          1,
				Name:        "parent",
				InterfaceID: 1,
				Address:     "address",
				Port:        1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.Account{
				ID:        1,
				Username:  "existing",
				Password:  []byte("existing"),
				PartnerID: parent.ID,
			}

			err = db.Create(&existing)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid account ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, accountsPath+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"account": id})

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
						existing.Password = nil
						exist, err := db.Exists(&existing)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})

			Convey("Given a request with a non-existing account ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, accountsPath+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"account": "1000"})

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

func TestUpdateAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_update_logger")

	Convey("Given the account updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 accounts", func() {
			parent := model.Partner{
				ID:          1,
				Name:        "old",
				InterfaceID: 1,
				Address:     "address1",
				Port:        1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.Account{
				ID:        1,
				Username:  "old",
				Password:  []byte("old"),
				PartnerID: parent.ID,
			}
			other := model.Account{
				ID:        2,
				Username:  "other",
				Password:  []byte("other"),
				PartnerID: parent.ID,
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the account with", func() {

				Convey("Given a new username", func() {
					update := struct{ Username string }{Username: "update"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					expected := model.Account{
						ID:        old.ID,
						Username:  update.Username,
						Password:  nil,
						PartnerID: old.ID,
					}

					checkValidUpdate(db, w, http.MethodPatch, accountsPath,
						id, "account", body, handler, &old, &expected)
				})

				Convey("Given an already existing username", func() {
					update := struct{ Username string }{Username: other.Username}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "An account with the same username already exist for this partner\n"
					checkInvalidUpdate(db, handler, w, body, accountsPath, id,
						"account", &old, msg)
				})

				Convey("Given an invalid new partner ID", func() {
					update := struct{ PartnerID uint64 }{PartnerID: 1000}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "No partner found with ID '1000'\n"
					checkInvalidUpdate(db, handler, w, body, accountsPath, id,
						"account", &old, msg)
				})
			})
		})
	})
}

func TestReplaceAccount(t *testing.T) {
	logger := log.NewLogger("rest_account_replace_logger")

	Convey("Given the account replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateAccount(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 accounts", func() {
			parent := model.Partner{
				ID:          1,
				Name:        "old",
				InterfaceID: 1,
				Address:     "address1",
				Port:        1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.Account{
				ID:        1,
				Username:  "old",
				Password:  []byte("old"),
				PartnerID: parent.ID,
			}
			other := model.Account{
				ID:        2,
				Username:  "other",
				Password:  []byte("other"),
				PartnerID: parent.ID,
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given a valid new account", func() {
				replace := struct {
					Username  string
					Password  []byte
					PartnerID uint64
				}{
					Username:  "replace",
					Password:  []byte("replace"),
					PartnerID: parent.ID,
				}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.Account{
					ID:        old.ID,
					Username:  replace.Username,
					Password:  nil,
					PartnerID: replace.PartnerID,
				}

				checkValidUpdate(db, w, http.MethodPut, accountsPath,
					id, "account", body, handler, &old, &expected)
			})

			Convey("Given a non-existing account ID parameter", func() {
				replace := struct{}{}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPut, accountsPath+"1000",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"account": "1000"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}
