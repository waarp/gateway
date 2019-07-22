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

type invalidObject struct {
	InvalidField1 int
	InvalidField2 string
}

func testListPartners(db *database.Db) {
	testPartner1 := &model.Partner{
		Name:    "testPartner1",
		Address: "test-address1",
		Port:    1,
		Type:    "type1",
	}
	testPartner2 := &model.Partner{
		Name:    "testPartner2",
		Address: "test-address3",
		Port:    1,
		Type:    "type2",
	}
	testPartner3 := &model.Partner{
		Name:    "testPartner3",
		Address: "test-address2",
		Port:    1,
		Type:    "type2",
	}
	testPartner4 := &model.Partner{
		Name:    "testPartner4",
		Address: "test-address4",
		Port:    1,
		Type:    "type3",
	}

	Convey("Given a partners listing function", func() {
		err := db.Create(testPartner1)
		So(err, ShouldBeNil)
		err = db.Create(testPartner2)
		So(err, ShouldBeNil)
		err = db.Create(testPartner3)
		So(err, ShouldBeNil)
		err = db.Create(testPartner4)
		So(err, ShouldBeNil)

		Convey("When calling it with no filters", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listPartners(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Partner{testPartner1, testPartner2, testPartner3, testPartner4}
				expected, err := json.Marshal(map[string]*[]*model.Partner{"partners": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a limit", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners?limit=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listPartners(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Partner{testPartner1}
				expected, err := json.Marshal(map[string]*[]*model.Partner{"partners": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an offset", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners?offset=1", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listPartners(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Partner{testPartner2, testPartner3, testPartner4}
				expected, err := json.Marshal(map[string]*[]*model.Partner{"partners": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an specific order", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners?sortby=address&order=desc", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listPartners(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Partner{testPartner4, testPartner2, testPartner3, testPartner1}
				expected, err := json.Marshal(map[string]*[]*model.Partner{"partners": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a filter by type", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners?type=type1&type=type2", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listPartners(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Partner{testPartner1, testPartner2, testPartner3}
				expected, err := json.Marshal(map[string]*[]*model.Partner{"partners": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a filter by address", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners?address=test-address2&address=test-address3", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listPartners(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Partner{testPartner2, testPartner3}
				expected, err := json.Marshal(map[string]*[]*model.Partner{"partners": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with a filter by type and address", func() {
			r, err := http.NewRequest(http.MethodGet,
				"/api/partners?address=test-address2&type=type2", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then it should reply OK with a JSON body", func() {
				listPartners(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				testResults := &[]*model.Partner{testPartner3}
				expected, err := json.Marshal(map[string]*[]*model.Partner{"partners": testResults})
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})
	})
}

func testGetPartner(db *database.Db) {
	testPartner := &model.Partner{
		Name:    "testPartner",
		Address: "test-address",
		Port:    1,
		Type:    "type",
	}

	Convey("Given a partner get function", func() {
		err := db.Create(testPartner)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid name", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/testPartner", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})

			Convey("Then it should reply OK with a JSON body", func() {
				getPartner(testLogger, db).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				expected, err := json.Marshal(testPartner)
				So(err, ShouldBeNil)

				So(w.Body.String(), ShouldResemble, string(expected)+"\n")
			})
		})

		Convey("When calling it with an invalid name", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/partners/unknown", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				getPartner(testLogger, db).ServeHTTP(w, r)

				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func testCreatePartner(db *database.Db) {
	testPartner := &model.Partner{
		Name:    "testPartner",
		Address: "test-address",
		Port:    1,
		Type:    "type1",
	}
	testPartnerFail := &model.Partner{
		Name:    "testPartnerFail",
		Address: "test-address-fail",
		Port:    1,
		Type:    "type2",
	}

	Convey("Given a partner creation function", func() {
		err := db.Create(testPartnerFail)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid JSON partner", func() {
			body, err := json.Marshal(testPartner)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, "/api/partners", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should create the partner and reply 'Created'", func() {
				createPartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)

				exist, err := db.Exists(testPartner)
				So(err, ShouldBeNil)
				So(exist, ShouldBeTrue)
			})
		})

		Convey("When calling it with an already existing name", func() {
			body, err := json.Marshal(testPartnerFail)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, "/api/partners", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createPartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When calling it with an invalid JSON body", func() {
			body, err := json.Marshal(invalidObject{})
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, "/api/partners", reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should reply 'Bad Request'", func() {
				createPartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testDeletePartner(db *database.Db) {
	testPartner := &model.Partner{
		Name:    "testPartner",
		Address: "test-address",
		Port:    1,
		Type:    "type",
	}

	Convey("Given a partner deletion function", func() {
		err := db.Create(testPartner)
		So(err, ShouldBeNil)

		Convey("When called with an existing name", func() {
			r, err := http.NewRequest(http.MethodDelete, "/api/partners/testPartner", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartner.Name})

			Convey("Then it should delete the partner and reply 'No Content'", func() {
				deletePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNoContent)

				exist, err := db.Exists(testPartner)
				So(err, ShouldBeNil)
				So(exist, ShouldBeFalse)
			})
		})

		Convey("When called with an unknown name", func() {
			r, err := http.NewRequest(http.MethodDelete, "/api/partners/unknown", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				deletePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func testUpdatePartner(db *database.Db) {
	testPartnerBefore := &model.Partner{
		Name:    "testPartnerBefore",
		Address: "test-address-before",
		Port:    1,
		Type:    "type1",
	}
	testPartnerAfter := &model.Partner{
		Name:    "testPartnerAfter",
		Address: "test-address-after",
		Type:    "type2",
	}

	Convey("Given a partner update function", func() {
		err := db.Create(testPartnerBefore)
		So(err, ShouldBeNil)

		Convey("When called with an existing name", func() {
			body, err := json.Marshal(testPartnerAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch, "/api/partners/testPartnerBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartnerBefore.Name})

			Convey("Then it should update the partner and reply 'Created'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)

				testPartnerAfter.Port = testPartnerBefore.Port
				existAfter, err := db.Exists(testPartnerAfter)
				So(err, ShouldBeNil)
				So(existAfter, ShouldBeTrue)

				existBefore, err := db.Exists(testPartnerBefore)
				So(err, ShouldBeNil)
				So(existBefore, ShouldBeFalse)
			})
		})

		Convey("When called with an unknown name", func() {
			body, err := json.Marshal(testPartnerAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch, "/api/partners/unknown", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an invalid JSON object", func() {
			body, err := json.Marshal(invalidObject{})
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch, "/api/partners/testPartnerBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartnerBefore.Name})

			Convey("Then it should reply 'Bad Request'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testReplacePartner(db *database.Db) {
	testPartnerBefore := &model.Partner{
		Name:    "testPartnerBefore",
		Address: "test-address-before",
		Port:    1,
		Type:    "type1",
	}
	testPartnerAfter := &model.Partner{
		Name:    "testPartnerAfter",
		Address: "test-address-after",
		Type:    "type2",
	}

	Convey("Given a partner replacing function", func() {
		err := db.Create(testPartnerBefore)
		So(err, ShouldBeNil)

		Convey("When called with an existing name", func() {

			body, err := json.Marshal(testPartnerAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, "/api/partners/testPartnerBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartnerBefore.Name})

			Convey("Then it should update the partner and reply 'Created'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)

				existAfter, err := db.Exists(testPartnerAfter)
				So(err, ShouldBeNil)
				So(existAfter, ShouldBeTrue)

				existBefore, err := db.Exists(testPartnerBefore)
				So(err, ShouldBeNil)
				So(existBefore, ShouldBeFalse)
			})
		})

		Convey("When called with an non-existing name", func() {
			body, err := json.Marshal(testPartnerAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, "/api/partners/unknown", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": "unknown"})

			Convey("Then it should reply 'Not Found'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When called with an invalid JSON object", func() {
			body, err := json.Marshal(invalidObject{})
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, "/api/partners/testPartnerBefore", reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": testPartnerBefore.Name})

			Convey("Then it should reply 'Bad Request'", func() {
				updateAccount(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func TestPartners(t *testing.T) {
	testDb := database.GetTestDatabase()

	Convey("Testing the 'partners' endpoint", t, func() {

		Reset(func() {
			err := testDb.Execute("DELETE FROM 'partners'")
			So(err, ShouldBeNil)
		})

		testListPartners(testDb)
		testGetPartner(testDb)
		testCreatePartner(testDb)
		testDeletePartner(testDb)
		testUpdatePartner(testDb)
		testReplacePartner(testDb)
	})

	_ = testDb.Stop(context.Background())
}
