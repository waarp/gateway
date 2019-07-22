package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func testListAccounts(db *database.Db, testPartner *model.Partner) {
	testAccount1 := &model.Account{
		Username:  "testAccount1",
		Password:  []byte("test-password"),
		PartnerID: testPartner.ID,
	}
	testAccount2 := &model.Account{
		Username:  "testAccount2",
		Password:  []byte("test-password"),
		PartnerID: testPartner.ID,
	}
	testAccount3 := &model.Account{
		Username:  "testAccount3",
		Password:  []byte("test-password"),
		PartnerID: 1000,
	}
	testAccount4 := &model.Account{
		Username:  "testAccount4",
		Password:  []byte("test-password"),
		PartnerID: testPartner.ID,
	}

	Convey("Given an account listing function", func() {
		err := db.Create(testAccount1)
		So(err, ShouldBeNil)
		err = db.Create(testAccount2)
		So(err, ShouldBeNil)
		err = db.Create(testAccount3)
		So(err, ShouldBeNil)
		err = db.Create(testAccount4)
		So(err, ShouldBeNil)
		testAccount1.Password = nil
		testAccount2.Password = nil
		testAccount3.Password = nil
		testAccount4.Password = nil

		Convey("When calling it with no filters", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/testPartner/accounts", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Account{testAccount1, testAccount2, testAccount4}
				expected, err := json.Marshal(map[string]*[]*model.Account{"accounts": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a limit", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/testPartner/accounts?limit=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Account{testAccount1}
				expected, err := json.Marshal(map[string]*[]*model.Account{"accounts": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an offset", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners/testPartner/accounts?offset=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Account{testAccount2, testAccount4}
				expected, err := json.Marshal(map[string]*[]*model.Account{"accounts": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an specific order", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners/testPartner/accounts?sortby=username&order=desc", nil)
			So(err, ShouldBeNil)
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusOK)

				contentType := w.Header().Get("Content-Type")
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Account{testAccount4, testAccount2, testAccount1}
				object := map[string]*[]*model.Account{"accounts": testResults}
				expected, err := json.Marshal(object)

				So(err, ShouldBeNil)
				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an non-existing partner name", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/unknown/accounts", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func testCreateAccount(db *database.Db, testPartner *model.Partner) {
	testAccount := &model.Account{
		Username:  "testAccount",
		Password:  []byte("test_account_password"),
		PartnerID: testPartner.ID,
	}
	testAccountFail := &model.Account{
		Username:  "testAccountFail",
		Password:  []byte("test_account_password"),
		PartnerID: testPartner.ID,
	}

	Convey("Given a account creation function", func() {
		err := db.Create(testAccountFail)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid JSON account", func() {
			body, err := json.Marshal(testAccount)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, "/api/partners/testPartner/accounts", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})
			w := httptest.NewRecorder()

			Convey("Then it should create the account and reply 'Created'", func() {
				createAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)

				exist, err := db.Exists(&model.Account{Username: testAccount.Username})
				So(err, ShouldBeNil)
				So(exist, ShouldBeTrue)
			})
		})

		Convey("When calling it with an already existing username", func() {
			body, err := json.Marshal(testAccountFail)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, "/api/partners/testPartner/accounts", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When calling it with an non-existing partner name", func() {
			body, err := json.Marshal(testAccount)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, "/api/partners/unknown/accounts", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown"})
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Not Found'", func() {
				createAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When calling it with an invalid JSON body", func() {
			body, err := json.Marshal(invalidObject{})
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, "/api/partners/testPartner/accounts", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testDeleteAccount(db *database.Db, testPartner *model.Partner) {
	testAccount := &model.Account{
		Username:  "testAccount",
		Password:  []byte("test_account_password"),
		PartnerID: testPartner.ID,
	}

	Convey("Given a account deletion function", func() {
		err := db.Create(testAccount)
		So(err, ShouldBeNil)

		Convey("When called with an existing username", func() {
			r, err := http.NewRequest(http.MethodDelete,
				"/api/partners/testPartner/accounts/testAccount", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username})

			Convey("Then it should delete the account and reply 'No Content'", func() {
				deleteAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNoContent)

				exist, err := db.Exists(testAccount)
				So(err, ShouldBeNil)
				So(exist, ShouldBeFalse)
			})
		})

		Convey("When called with an unknown username", func() {
			r, err := http.NewRequest(http.MethodDelete,
				"/api/partners/testPartner/accounts/unknown", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name, "account": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				deleteAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown name", func() {
			r, err := http.NewRequest(http.MethodDelete,
				"/api/partners/unknown/accounts/testPartner", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown", "account": testAccount.Username})

			Convey("Then it should reply 'Not Found'", func() {
				deleteAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func checkAccountValidUpdate(w *httptest.ResponseRecorder, db *database.Db,
	before, after *model.Account) {

	So(w.Code, ShouldEqual, http.StatusCreated)

	existAfter, err := db.Exists(&model.Account{Username: after.Username})
	So(err, ShouldBeNil)
	So(existAfter, ShouldBeTrue)

	existBefore, err := db.Exists(&model.Account{Username: before.Username})
	So(err, ShouldBeNil)
	So(existBefore, ShouldBeFalse)
}

func testUpdateAccount(db *database.Db, testPartner *model.Partner) {
	testAccountBefore := &model.Account{
		Username:  "testAccountBefore",
		Password:  []byte("test_account_password"),
		PartnerID: testPartner.ID,
	}
	testAccountAfter := &model.Account{
		Username:  "testAccountAfter",
		PartnerID: testPartner.ID,
	}

	Convey("Given a account update function", func() {
		err := db.Create(testAccountBefore)
		So(err, ShouldBeNil)

		Convey("When called with an existing name", func() {
			body, err := json.Marshal(testAccountAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch,
				"/api/partners/testPartner/accounts/testAccountBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccountBefore.Username})

			Convey("Then it should update the account and reply 'Created'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				testAccountAfter.Password = testAccountBefore.Password
				checkAccountValidUpdate(w, db, testAccountBefore, testAccountAfter)
			})
		})

		Convey("When called with an unknown username", func() {
			body, err := json.Marshal(testAccountAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch,
				"/api/partners/unknown/accounts/testAccountBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown",
				"account": testAccountBefore.Username})

			Convey("Then it should reply 'Not Found'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown partner name", func() {
			body, err := json.Marshal(testAccountAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch,
				"/api/partners/testPartner/accounts/unknown", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name, "account": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an invalid JSON object", func() {
			body, err := json.Marshal(invalidObject{})
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch,
				"/api/partners/testPartner/accounts/testAccountBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccountBefore.Username})

			Convey("Then it should reply 'Bad Request'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testReplaceAccount(db *database.Db, testPartner *model.Partner) {
	testAccountBefore := &model.Account{
		Username:  "testAccountBefore",
		Password:  []byte("test_account_password"),
		PartnerID: testPartner.ID,
	}
	testAccountAfter := &model.Account{
		Username:  "testAccountAfter",
		PartnerID: testPartner.ID,
	}

	Convey("Given a account replace function", func() {
		err := db.Create(testAccountBefore)
		So(err, ShouldBeNil)

		Convey("When called with an existing name", func() {
			body, err := json.Marshal(testAccountAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut,
				"/api/partners/testPartner/accounts/testAccountBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccountBefore.Username})

			Convey("Then it should update the partner and reply 'Created'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				checkAccountValidUpdate(w, db, testAccountBefore, testAccountAfter)
			})
		})

		Convey("When called with an non-existing username", func() {
			body, err := json.Marshal(testAccountAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut,
				"/api/partners/testPartner/accounts/unknown", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccountAfter.Username})

			Convey("Then it should reply 'Not Found'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown partner name", func() {
			body, err := json.Marshal(testAccountAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut,
				"/api/partners/unknown/accounts/testAccountBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown",
				"account": testAccountBefore.Username})

			Convey("Then it should reply 'Not Found'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an invalid JSON object", func() {
			body, err := json.Marshal(invalidObject{})
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut,
				"/api/partners/testPartner/accounts/testAccountBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccountBefore.Username})

			Convey("Then it should reply 'Bad Request'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func TestAccounts(t *testing.T) {
	testDb := database.GetTestDatabase()

	testPartner := &model.Partner{
		Name:    "testPartner",
		Address: "test-address",
		Port:    1,
		Type:    "type",
	}
	if err := testDb.Create(testPartner); err != nil {
		t.Fatal(err)
	}

	Convey("Testing the 'accounts' endpoint", t, func() {

		Reset(func() {
			err := testDb.Execute("DELETE FROM 'accounts'")
			So(err, ShouldBeNil)
		})

		testListAccounts(testDb, testPartner)
		testCreateAccount(testDb, testPartner)
		testDeleteAccount(testDb, testPartner)
		testUpdateAccount(testDb, testPartner)
		testReplaceAccount(testDb, testPartner)
	})

	_ = testDb.Execute("DELETE FROM 'partners'")

	_ = testDb.Stop(context.Background())
}
