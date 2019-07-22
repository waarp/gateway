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

func testListCerts(db *database.Db, testPartner *model.Partner, testAccount *model.Account) {

	testCert1 := &model.CertChain{
		Name:      "testCert1",
		AccountID: testAccount.ID,
	}
	testCert2 := &model.CertChain{
		Name:      "testCert2",
		AccountID: testAccount.ID,
	}
	testCert3 := &model.CertChain{
		Name:      "testCert3",
		AccountID: 100,
	}
	testCert4 := &model.CertChain{
		Name:      "testCert4",
		AccountID: testAccount.ID,
	}

	Convey("Given an certificates listing function", func() {
		err := db.Create(testCert1)
		So(err, ShouldBeNil)
		err = db.Create(testCert2)
		So(err, ShouldBeNil)
		err = db.Create(testCert3)
		So(err, ShouldBeNil)
		err = db.Create(testCert4)
		So(err, ShouldBeNil)

		Convey("When calling it with no filters", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners/testPartner/accounts/testAccount/certificates", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username})

			Convey("Then it should reply OK with a JSON body", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.CertChain{testCert1, testCert2, testCert4}
				expected, err := json.Marshal(map[string]*[]*model.CertChain{"certificates": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a limit", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners/testPartner/accounts/testAccount/certificates?limit=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username})

			Convey("Then it should reply OK with a JSON body", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.CertChain{testCert1}
				expected, err := json.Marshal(map[string]*[]*model.CertChain{"certificates": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an offset", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners/testPartner/accounts/testAccount/certificates?offset=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username})

			Convey("Then it should reply OK with a JSON body", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.CertChain{testCert2, testCert4}
				expected, err := json.Marshal(map[string]*[]*model.CertChain{"certificates": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an specific order", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/testPartner/accounts"+
				"/testAccount/certificates?sortby=name&order=desc", nil)
			So(err, ShouldBeNil)
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username})
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusOK)

				contentType := w.Header().Get("Content-Type")
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.CertChain{testCert4, testCert2, testCert1}
				object := map[string]*[]*model.CertChain{"certificates": testResults}
				expected, err := json.Marshal(object)

				So(err, ShouldBeNil)
				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an non-existing partner name", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners/unknown/accounts/testAccount/certificates", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown",
				"account": testAccount.Username})

			Convey("Then it should reply 'Not Found'", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When calling it with an non-existing account name", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners/testPartner/accounts/unknown/certificates", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func testCreateCert(db *database.Db, testPartner *model.Partner, testAccount *model.Account) {

	testCert := &model.CertChain{
		Name:      "testCert",
		AccountID: testAccount.ID,
	}
	testCertFail := &model.CertChain{
		Name:      "testCertFail",
		AccountID: testAccount.ID,
	}

	Convey("Given a certificate creation function", func() {
		err := db.Create(testCertFail)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid JSON account", func() {
			body, err := json.Marshal(testCert)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost,
				"/api/partners/testPartner/accounts/testAccount/certificates", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username})
			w := httptest.NewRecorder()

			Convey("Then it should create the certificate and reply 'Created'", func() {
				createCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)

				exist, err := db.Exists(&model.CertChain{Name: testCert.Name, AccountID: testAccount.ID})
				So(err, ShouldBeNil)
				So(exist, ShouldBeTrue)
			})
		})

		Convey("When calling it with an already existing name", func() {
			body, err := json.Marshal(testCertFail)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost,
				"/api/partners/testPartner/accounts/testAccount/certificates", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username})
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When calling it with an non-existing partner name", func() {
			body, err := json.Marshal(testCert)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost,
				"/api/partners/unknown/accounts/testAccount/certificates", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown",
				"account": testAccount.Username})
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Not Found'", func() {
				createCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When calling it with an non-existing account name", func() {
			body, err := json.Marshal(testCert)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost,
				"/api/partners/testPartner/accounts/unknown/certificates", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": "unknown"})
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Not Found'", func() {
				createCertificate(testLogger, db).ServeHTTP(w, r)
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
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username})
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testGetCert(db *database.Db, testPartner *model.Partner, testAccount *model.Account) {

	testCert := &model.CertChain{
		Name:      "testCert",
		AccountID: testAccount.ID,
	}

	Convey("Given a certificate get function", func() {
		err := db.Create(testCert)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid name", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/testCert", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username, "certificate": testCert.Name})

			Convey("Then it should reply OK with a JSON body", func() {
				getCertificate(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				expected, err := json.Marshal(testCert)
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an invalid name", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/unknown", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username, "certificate": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				getCertificate(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When calling it with an invalid account name", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/testPartner/"+
				"accounts/unknown/certificates/testCert", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": "unknown", "certificate": testCert.Name})

			Convey("Then it should reply 'Not Found'", func() {
				getCertificate(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When calling it with an invalid partner name", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/unknown/"+
				"accounts/testAccount/certificates/testCert", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown",
				"account": testAccount.Username, "certificate": testCert.Name})

			Convey("Then it should reply 'Not Found'", func() {
				getCertificate(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func testDeleteCert(db *database.Db, testPartner *model.Partner, testAccount *model.Account) {
	testCert := &model.CertChain{
		Name:      "testCert",
		AccountID: testAccount.ID,
	}

	Convey("Given a certificate deletion function", func() {
		err := db.Create(testCert)
		So(err, ShouldBeNil)

		Convey("When called with an existing certificate name", func() {
			r, err := http.NewRequest(http.MethodDelete, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/testCert", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username, "certificate": testCert.Name})

			Convey("Then it should delete the account and reply 'No Content'", func() {
				deleteCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNoContent)

				exist, err := db.Exists(testCert)
				So(err, ShouldBeNil)
				So(exist, ShouldBeFalse)
			})
		})

		Convey("When called with an unknown certificate name", func() {
			r, err := http.NewRequest(http.MethodDelete, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/unknown", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username, "certificate": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				deleteCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown account name", func() {
			r, err := http.NewRequest(http.MethodDelete, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/unknown", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": "unknown", "certificate": testCert.Name})

			Convey("Then it should reply 'Not Found'", func() {
				deleteCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown partner name", func() {
			r, err := http.NewRequest(http.MethodDelete, "/api/partners/unknown/"+
				"accounts/testAccount/certificates/testCert", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown",
				"account": testAccount.Username, "certificate": testCert.Name})

			Convey("Then it should reply 'Not Found'", func() {
				deleteCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func checkCertValidUpdate(w *httptest.ResponseRecorder, db *database.Db,
	before, after *model.CertChain) {

	So(w.Code, ShouldEqual, http.StatusCreated)

	existAfter, err := db.Exists(after)
	So(err, ShouldBeNil)
	So(existAfter, ShouldBeTrue)

	existBefore, err := db.Exists(before)
	So(err, ShouldBeNil)
	So(existBefore, ShouldBeFalse)
}

func testUpdateCert(db *database.Db, testPartner *model.Partner, testAccount *model.Account) {
	testCertBefore := &model.CertChain{
		Name:      "testCertBefore",
		AccountID: testAccount.ID,
		PublicKey: []byte("test_public_key"),
	}
	testCertAfter := &model.CertChain{
		Name:      "testCertAfter",
		AccountID: testAccount.ID,
	}

	Convey("Given a certificate update function", func() {
		err := db.Create(testCertBefore)
		So(err, ShouldBeNil)

		Convey("When called with an existing name", func() {
			body, err := json.Marshal(testCertAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/testCert", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username, "certificate": testCertBefore.Name})

			Convey("Then it should update the account and reply 'Created'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
				testCertAfter.PublicKey = testCertBefore.PublicKey
				checkCertValidUpdate(w, db, testCertBefore, testCertAfter)
			})
		})

		Convey("When called with an unknown certificate name", func() {
			body, err := json.Marshal(testCertAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/unknown", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username, "certificate": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown account name", func() {
			body, err := json.Marshal(testCertAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch, "/api/partners/testPartner/"+
				"accounts/unknown/certificates/testCert", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": "unknown", "certificate": testCertBefore.Name})

			Convey("Then it should reply 'Not Found'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown partner name", func() {
			body, err := json.Marshal(testCertAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch, "/api/partners/unknown/"+
				"accounts/testAccount/certificates/testCert", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown",
				"account": testAccount.Username, "certificate": testCertBefore.Name})

			Convey("Then it should reply 'Not Found'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
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
				"account": testAccount.Username, "certificate": testCertBefore.Name})

			Convey("Then it should reply 'Bad Request'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testReplaceCert(db *database.Db, testPartner *model.Partner, testAccount *model.Account) {
	testCertBefore := &model.CertChain{
		Name:      "testCertBefore",
		AccountID: testAccount.ID,
	}
	testCertAfter := &model.CertChain{
		Name:      "testCertAfter",
		AccountID: testAccount.ID,
	}

	Convey("Given a certificate replacing function", func() {
		err := db.Create(testCertBefore)
		So(err, ShouldBeNil)

		Convey("When called with an existing name", func() {
			body, err := json.Marshal(testCertAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/testCert", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username, "certificate": testCertBefore.Name})

			Convey("Then it should replace the account and reply 'Created'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
				checkCertValidUpdate(w, db, testCertBefore, testCertAfter)
			})
		})

		Convey("When called with an unknown certificate name", func() {
			body, err := json.Marshal(testCertAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, "/api/partners/testPartner/"+
				"accounts/testAccount/certificates/unknown", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": testAccount.Username, "certificate": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown account name", func() {
			body, err := json.Marshal(testCertAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, "/api/partners/testPartner/"+
				"accounts/unknown/certificates/testCert", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name,
				"account": "unknown", "certificate": testCertBefore.Name})

			Convey("Then it should reply 'Not Found'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an unknown partner name", func() {
			body, err := json.Marshal(testCertAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, "/api/partners/unknown/"+
				"accounts/testAccount/certificates/testCert", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown",
				"account": testAccount.Username, "certificate": testCertBefore.Name})

			Convey("Then it should reply 'Not Found'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
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
				"account": testAccount.Username, "certificate": testCertBefore.Name})

			Convey("Then it should reply 'Bad Request'", func() {
				updateCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func TestCerts(t *testing.T) {
	testDb := database.GetTestDatabase()

	testPartner := &model.Partner{
		Name:    "testPartner",
		Address: "test-address",
		Port:    1,
		Type:    "type",
	}
	testAccount := &model.Account{
		Username: "testAccount",
		Password: []byte("test_account_password"),
	}
	if err := testDb.Create(testPartner); err != nil {
		t.Fatal(err)
	}
	testAccount.PartnerID = testPartner.ID
	if err := testDb.Create(testAccount); err != nil {
		t.Fatal(err)
	}

	Convey("Testing the 'certificates' endpoint", t, func() {

		Reset(func() {
			err := testDb.Execute("DELETE FROM 'certificates'")
			So(err, ShouldBeNil)
		})

		testListCerts(testDb, testPartner, testAccount)
		testCreateCert(testDb, testPartner, testAccount)
		testGetCert(testDb, testPartner, testAccount)
		testDeleteCert(testDb, testPartner, testAccount)
		testUpdateCert(testDb, testPartner, testAccount)
		testReplaceCert(testDb, testPartner, testAccount)
	})

	_ = testDb.Execute("DELETE FROM 'partners'")
	_ = testDb.Execute("DELETE FROM 'accounts'")

	_ = testDb.Stop(context.Background())
}
