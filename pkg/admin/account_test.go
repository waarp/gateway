package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const accountsPath = RestURI + AccountsURI + "/"

func testGetAccount(db *database.Db, partnerID uint64) {
	testAccount := model.Account{
		Username:  "test_account",
		Password:  []byte("test_account_password"),
		PartnerID: partnerID,
	}

	Convey("Given a account get function", func() {
		err := db.Create(&testAccount)
		So(err, ShouldBeNil)

		id := strconv.FormatUint(testAccount.ID, 10)

		Convey("When called with an existing id", func() {
			r, err := http.NewRequest(http.MethodGet, accountsPath+id, nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"account": id})

			Convey("Then it should return the account and reply 'OK'", func() {
				getAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusOK)

				contentType := w.Header().Get("Content-Type")
				So(contentType, ShouldEqual, "application/json")

				expected, err := json.Marshal(&testAccount)
				So(err, ShouldBeNil)
				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When called with an unknown username", func() {
			r, err := http.NewRequest(http.MethodDelete, accountsPath+"/unknown", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"account": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				getAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func testListAccounts(db *database.Db, partnerID uint64) {
	testAccount1 := model.Account{
		Username:  "test_account1",
		Password:  []byte("test_account_password"),
		PartnerID: partnerID,
	}
	testAccount2 := model.Account{
		Username:  "test_account2",
		Password:  []byte("test_account_password"),
		PartnerID: partnerID,
	}
	testAccount3 := model.Account{
		Username:  "test_account3",
		Password:  []byte("test_account_password"),
		PartnerID: 1000,
	}
	testAccount4 := model.Account{
		Username:  "test_account4",
		Password:  []byte("test_account_password"),
		PartnerID: partnerID,
	}

	Convey("Given an account listing function", func() {
		err := db.Create(&testAccount1)
		So(err, ShouldBeNil)
		err = db.Create(&testAccount2)
		So(err, ShouldBeNil)
		err = db.Create(&testAccount3)
		So(err, ShouldBeNil)
		err = db.Create(&testAccount4)
		So(err, ShouldBeNil)

		Convey("When calling it with no filters", func() {
			r, err := http.NewRequest(http.MethodGet, accountsPath, nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.Account{testAccount1, testAccount2,
					testAccount3, testAccount4}
				expected, err := json.Marshal(map[string][]model.Account{"accounts": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a limit", func() {
			r, err := http.NewRequest(http.MethodGet, accountsPath+"?limit=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.Account{testAccount1}
				expected, err := json.Marshal(map[string][]model.Account{"accounts": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an offset", func() {
			r, err := http.NewRequest(http.MethodGet, accountsPath+"?offset=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.Account{testAccount2, testAccount3, testAccount4}
				expected, err := json.Marshal(map[string][]model.Account{"accounts": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an specific order", func() {
			r, err := http.NewRequest(http.MethodGet, accountsPath+"?sortby=username&order=desc", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusOK)

				contentType := w.Header().Get("Content-Type")
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.Account{testAccount4, testAccount3,
					testAccount2, testAccount1}
				object := map[string][]model.Account{"accounts": testResults}
				expected, err := json.Marshal(object)

				So(err, ShouldBeNil)
				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a partner id filter", func() {
			id := strconv.FormatUint(partnerID, 10)
			r, err := http.NewRequest(http.MethodGet, accountsPath+"?partner="+id, nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listAccounts(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusOK)

				contentType := w.Header().Get("Content-Type")
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.Account{testAccount1, testAccount2, testAccount4}
				object := map[string][]model.Account{"accounts": testResults}
				expected, err := json.Marshal(object)

				So(err, ShouldBeNil)
				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})
	})
}

func testCreateAccount(db *database.Db, partnerID uint64) {
	testAccount := model.Account{
		Username:  "test_account",
		Password:  []byte("test_account_password"),
		PartnerID: partnerID,
	}
	testAccountFail := model.Account{
		Username:  "test_account_fail",
		Password:  []byte("test_account_password"),
		PartnerID: partnerID,
	}

	Convey("Given a account creation function", func() {
		err := db.Create(&testAccountFail)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid JSON account", func() {
			body, err := json.Marshal(testAccount)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, accountsPath, reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should create the account and reply 'Created'", func() {
				createAccount(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusCreated)

				testAccount.Password = nil
				exist, err := db.Exists(&testAccount)
				So(err, ShouldBeNil)
				So(exist, ShouldBeTrue)

				err = db.Get(&testAccount)
				So(err, ShouldBeNil)
				id := strconv.FormatUint(testAccount.ID, 10)
				So(w.Header().Get("Location"), ShouldResemble, accountsPath+id)
			})
		})

		Convey("When calling it with an already existing id", func() {
			body, err := json.Marshal(testAccountFail)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, accountsPath, reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createAccount(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When calling it with an invalid JSON body", func() {
			body, err := json.Marshal(invalidObject{})
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, accountsPath, reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testDeleteAccount(db *database.Db, partnerID uint64) {
	testAccount := model.Account{
		Username:  "test_account",
		Password:  []byte("test_account_password"),
		PartnerID: partnerID,
	}

	Convey("Given a account deletion function", func() {
		err := db.Create(&testAccount)
		So(err, ShouldBeNil)

		id := strconv.FormatUint(testAccount.ID, 10)

		deleteTest(deleteAccount(testLogger, db), db, &testAccount, id, "account", accountsPath)
	})
}

func testUpdateAccount(db *database.Db, partnerID uint64) {
	testAccountBefore := model.Account{
		Username:  "test_account_before",
		Password:  []byte("test_account_password_before"),
		PartnerID: partnerID,
	}
	testAccountUpdate := &struct {
		Username  string
		PartnerID uint64
	}{
		Username:  "test_account_after",
		PartnerID: partnerID,
	}

	Convey("Given a account update function", func() {
		err := db.Create(&testAccountBefore)
		So(err, ShouldBeNil)
		testAccountAfter := model.Account{
			ID:        testAccountBefore.ID,
			Username:  testAccountUpdate.Username,
			PartnerID: testAccountUpdate.PartnerID,
			Password:  testAccountBefore.Password,
		}

		id := strconv.FormatUint(testAccountBefore.ID, 10)

		updateTest(updateAccount(testLogger, db), db, &testAccountBefore, testAccountUpdate,
			&testAccountAfter, accountsPath, "account", id, false)
	})
}

func testReplaceAccount(db *database.Db, partnerID uint64) {
	testAccountBefore := model.Account{
		Username:  "test_account_before",
		Password:  []byte("test_account_password"),
		PartnerID: partnerID,
	}
	testAccountUpdate := model.Account{
		Username:  "test_account_after",
		PartnerID: partnerID,
	}

	Convey("Given a account replace function", func() {
		err := db.Create(&testAccountBefore)
		So(err, ShouldBeNil)
		testAccountUpdate.ID = testAccountBefore.ID
		testAccountUpdate.Password = testAccountBefore.Password

		id := strconv.FormatUint(testAccountBefore.ID, 10)

		updateTest(updateAccount(testLogger, db), db, &testAccountBefore, testAccountUpdate,
			&testAccountUpdate, accountsPath, "account", id, true)
	})
}

func TestAccounts(t *testing.T) {
	testDb := database.GetTestDatabase()

	testPartner := model.Partner{
		Name:    "test_partner",
		Address: "test_partner_address",
		Port:    1,
	}
	if err := testDb.Create(&testPartner); err != nil {
		t.Fatal(err)
	}

	Convey("Testing the 'accounts' endpoint", t, func() {

		Reset(func() {
			err := testDb.Execute("DELETE FROM 'accounts'")
			So(err, ShouldBeNil)
		})

		testGetAccount(testDb, testPartner.ID)
		testListAccounts(testDb, testPartner.ID)
		testCreateAccount(testDb, testPartner.ID)
		testDeleteAccount(testDb, testPartner.ID)
		testUpdateAccount(testDb, testPartner.ID)
		testReplaceAccount(testDb, testPartner.ID)
	})

	_ = testDb.Execute("DELETE FROM 'partners'")

	_ = testDb.Stop(context.Background())
}
