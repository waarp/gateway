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

const certPath = RestURI + CertsURI + "/"

func testListCerts(db *database.Db, accountID uint64) {

	testCert1 := model.CertChain{
		Name:       "test_cert1",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}
	testCert2 := model.CertChain{
		Name:       "test_cert2",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}
	testCert3 := model.CertChain{
		Name:       "test_cert3",
		OwnerType:  "ACCOUNT",
		OwnerID:    100,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}
	testCert4 := model.CertChain{
		Name:       "test_cert4",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}

	Convey("Given an certificates listing function", func() {
		err := db.Create(&testCert1)
		So(err, ShouldBeNil)
		err = db.Create(&testCert2)
		So(err, ShouldBeNil)
		err = db.Create(&testCert3)
		So(err, ShouldBeNil)
		err = db.Create(&testCert4)
		So(err, ShouldBeNil)

		Convey("When calling it with no filters", func() {
			r, err := http.NewRequest(http.MethodGet, certPath, nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.CertChain{testCert1, testCert2,
					testCert3, testCert4}
				expected, err := json.Marshal(map[string][]model.CertChain{"certificates": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a limit", func() {
			r, err := http.NewRequest(http.MethodGet, certPath+"?limit=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.CertChain{testCert1}
				expected, err := json.Marshal(map[string][]model.CertChain{"certificates": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an offset", func() {
			r, err := http.NewRequest(http.MethodGet, certPath+"?offset=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.CertChain{testCert2, testCert3, testCert4}
				expected, err := json.Marshal(map[string][]model.CertChain{"certificates": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an specific order", func() {
			r, err := http.NewRequest(http.MethodGet, certPath+"?sortby=name&order=desc", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listCertificates(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusOK)

				contentType := w.Header().Get("Content-Type")
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := []model.CertChain{testCert4, testCert3, testCert2, testCert1}
				object := map[string][]model.CertChain{"certificates": testResults}
				expected, err := json.Marshal(object)

				So(err, ShouldBeNil)
				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})
	})
}

func testCreateCert(db *database.Db, accountID uint64) {

	testCert := model.CertChain{
		Name:       "test_cert",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}
	testCertFail := model.CertChain{
		Name:       "test_cert_fail",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}

	Convey("Given a certificate creation function", func() {
		err := db.Create(&testCertFail)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid JSON account", func() {
			body, err := json.Marshal(testCert)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, certPath, reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should create the certificate and reply 'Created'", func() {
				createCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)

				exist, err := db.Exists(&model.CertChain{Name: testCert.Name, OwnerID: accountID})
				So(err, ShouldBeNil)
				So(exist, ShouldBeTrue)

				err = db.Get(&testCert)
				So(err, ShouldBeNil)
				id := strconv.FormatUint(testCert.ID, 10)
				So(w.Header().Get("Location"), ShouldResemble, certPath+id)
			})
		})

		Convey("When calling it with an already existing name", func() {
			body, err := json.Marshal(testCertFail)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, certPath, reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When calling it with an invalid JSON body", func() {
			body, err := json.Marshal(invalidObject{})
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, certPath, reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createCertificate(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testGetCert(db *database.Db, accountID uint64) {
	testCert := model.CertChain{
		Name:       "test_cert",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}

	Convey("Given a certificate get function", func() {
		err := db.Create(&testCert)
		So(err, ShouldBeNil)

		id := strconv.FormatUint(testCert.ID, 10)

		Convey("When calling it with a valid name", func() {
			r, err := http.NewRequest(http.MethodGet, certPath+id, nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"certificate": id})

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
			r, err := http.NewRequest(http.MethodGet, certPath+"unknown", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"certificate": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				getCertificate(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func testDeleteCert(db *database.Db, accountID uint64) {
	testCert := model.CertChain{
		Name:       "test_cert",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}

	Convey("Given a certificate deletion function", func() {
		err := db.Create(&testCert)
		So(err, ShouldBeNil)

		id := strconv.FormatUint(testCert.ID, 10)

		deleteTest(deleteCertificate(testLogger, db), db, &testCert, id, "certificate", certPath)
	})
}

func testUpdateCert(db *database.Db, accountID uint64) {
	testCertBefore := model.CertChain{
		Name:       "test_cert_before",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}
	testCertUpdate := &struct {
		Name       string
		OwnerID    uint64
		PrivateKey []byte
	}{
		Name:       "test_cert_after",
		OwnerID:    accountID,
		PrivateKey: []byte("new_private_key"),
	}

	Convey("Given a certificate update function", func() {
		err := db.Create(&testCertBefore)
		So(err, ShouldBeNil)
		testCertAfter := model.CertChain{
			ID:         testCertBefore.ID,
			OwnerType:  "ACCOUNT",
			OwnerID:    testCertUpdate.OwnerID,
			Name:       testCertUpdate.Name,
			PrivateKey: testCertUpdate.PrivateKey,
			PublicKey:  testCertBefore.PublicKey,
			Cert:       testCertBefore.Cert,
		}

		id := strconv.FormatUint(testCertBefore.ID, 10)

		updateTest(updateCertificate(testLogger, db), db, &testCertBefore, testCertUpdate,
			&testCertAfter, certPath, "certificate", id, false)
	})
}

func testReplaceCert(db *database.Db, accountID uint64) {
	testCertBefore := model.CertChain{
		Name:       "test_cert_before",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("private_key"),
		PublicKey:  []byte("public_key"),
		Cert:       []byte("cert"),
	}
	testCertAfter := model.CertChain{
		Name:       "test_cert_after",
		OwnerType:  "ACCOUNT",
		OwnerID:    accountID,
		PrivateKey: []byte("new_private_key"),
		PublicKey:  []byte("new_public_key"),
		Cert:       []byte("new_cert"),
	}

	Convey("Given a certificate replacing function", func() {
		err := db.Create(&testCertBefore)
		So(err, ShouldBeNil)
		testCertAfter.ID = testCertBefore.ID

		id := strconv.FormatUint(testCertBefore.ID, 10)

		updateTest(updateCertificate(testLogger, db), db, &testCertBefore, testCertAfter,
			&testCertAfter, certPath, "certificate", id, true)
	})
}

func TestCerts(t *testing.T) {
	testDb := database.GetTestDatabase()

	testPartner := model.Partner{
		ID:      1,
		Name:    "test_partner",
		Address: "test_partner_address",
		Port:    1,
	}
	testAccount := model.Account{
		ID:        1,
		PartnerID: testPartner.ID,
		Username:  "test_account",
		Password:  []byte("test_account_password"),
	}
	if err := testDb.Create(&testPartner); err != nil {
		t.Fatal(err)
	}
	if err := testDb.Create(&testAccount); err != nil {
		t.Fatal(err)
	}

	Convey("Testing the 'certificates' endpoint", t, func() {

		Reset(func() {
			err := testDb.Execute("DELETE FROM 'certificates'")
			So(err, ShouldBeNil)
		})

		testListCerts(testDb, testAccount.ID)
		testCreateCert(testDb, testAccount.ID)
		testGetCert(testDb, testAccount.ID)
		testDeleteCert(testDb, testAccount.ID)
		testUpdateCert(testDb, testAccount.ID)
		testReplaceCert(testDb, testAccount.ID)
	})

	_ = testDb.Execute("DELETE FROM 'partners'")
	_ = testDb.Execute("DELETE FROM 'accounts'")

	_ = testDb.Stop(context.Background())
}
