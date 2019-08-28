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

type invalidObject struct {
	InvalidField1 int
	InvalidField2 string
}

const partnerPath = RestURI + PartnersURI + "/"

func testListPartners(db *database.Db) {
	testPartner1 := &model.Partner{
		Name:    "test_partner1",
		Address: "test_partner_address1",
		Port:    1,
		Type:    "sftp",
	}
	testPartner2 := &model.Partner{
		Name:    "test_partner2",
		Address: "test_partner_address3",
		Port:    1,
		Type:    "http",
	}
	testPartner3 := &model.Partner{
		Name:    "test_partner3",
		Address: "test_partner_address2",
		Port:    1,
		Type:    "http",
	}
	testPartner4 := &model.Partner{
		Name:    "test_partner4",
		Address: "test_partner_address4",
		Port:    1,
		Type:    "r66",
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
			r, err := http.NewRequest(http.MethodGet, partnerPath, nil)
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
			r, err := http.NewRequest(http.MethodGet, partnerPath+"?limit=1", nil)
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
			r, err := http.NewRequest(http.MethodGet, partnerPath+"?offset=1", nil)
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
				partnerPath+"?sortby=address&order=desc", nil)
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
				partnerPath+"?type=sftp&type=http", nil)
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
				partnerPath+"?address=test_partner_address2&address=test_partner_address3", nil)
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
				partnerPath+"?address=test_partner_address2&type=http", nil)
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
		Name:    "test_partner",
		Address: "test_partner_address",
		Port:    1,
		Type:    "sftp",
	}

	Convey("Given a partner get function", func() {
		err := db.Create(testPartner)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid name", func() {
			id := strconv.FormatUint(testPartner.ID, 10)
			r, err := http.NewRequest(http.MethodGet, partnerPath+id, nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": id})

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
			r, err := http.NewRequest(http.MethodGet, partnerPath+"unknown", nil)
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
		Name:    "test_partner",
		Address: "test_partner_address",
		Port:    1,
		Type:    "sftp",
	}
	testPartnerFail := &model.Partner{
		Name:    "test_partner_fail",
		Address: "test_partner_address_fail",
		Port:    1,
		Type:    "http",
	}

	Convey("Given a partner creation function", func() {
		err := db.Create(testPartnerFail)
		So(err, ShouldBeNil)

		Convey("When calling it with a valid JSON partner", func() {
			body, err := json.Marshal(testPartner)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, partnerPath, reader)
			So(err, ShouldBeNil)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			Convey("Then it should create the partner and reply 'Created'", func() {
				createPartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)

				exist, err := db.Exists(testPartner)
				So(err, ShouldBeNil)
				So(exist, ShouldBeTrue)

				err = db.Get(testPartner)
				So(err, ShouldBeNil)
				id := strconv.FormatUint(testPartner.ID, 10)
				So(w.Header().Get("Location"), ShouldResemble, partnerPath+id)
			})
		})

		Convey("When calling it with an already existing name", func() {
			body, err := json.Marshal(testPartnerFail)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPost, partnerPath, reader)
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
			r, err := http.NewRequest(http.MethodPost, partnerPath, reader)
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
		Name:    "test_partner",
		Address: "test_partner_address",
		Port:    1,
		Type:    "sftp",
	}

	Convey("Given a partner deletion function", func() {
		err := db.Create(testPartner)
		So(err, ShouldBeNil)

		Convey("When called with an existing name", func() {
			id := strconv.FormatUint(testPartner.ID, 10)
			r, err := http.NewRequest(http.MethodDelete, partnerPath+id, nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": id})

			Convey("Then it should delete the partner and reply 'No Content'", func() {
				deletePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNoContent)

				exist, err := db.Exists(testPartner)
				So(err, ShouldBeNil)
				So(exist, ShouldBeFalse)
			})
		})

		Convey("When called with an unknown name", func() {
			r, err := http.NewRequest(http.MethodDelete, partnerPath+"unknown", nil)
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
		Name:    "test_partner_before",
		Address: "test_partner_address_before",
		Port:    1,
		Type:    "sftp",
	}
	testPartnerAfter := &model.Partner{
		Name:    "test_partner_after",
		Address: "test_partner_address_after",
	}

	Convey("Given a partner update function", func() {
		err := db.Create(testPartnerBefore)
		So(err, ShouldBeNil)

		id := strconv.FormatUint(testPartnerBefore.ID, 10)

		Convey("When called with an existing name", func() {
			body, err := json.Marshal(testPartnerAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPatch, partnerPath+id, reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": id})

			Convey("Then it should update the partner and reply 'Created'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)
				So(w.Header().Get("Location"), ShouldResemble, partnerPath+id)

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
			r, err := http.NewRequest(http.MethodPatch, partnerPath+"unknown", reader)
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
			r, err := http.NewRequest(http.MethodPatch, partnerPath+id, reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": id})

			Convey("Then it should reply 'Bad Request'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func testReplacePartner(db *database.Db) {
	testPartnerBefore := &model.Partner{
		Name:    "test_partner_before",
		Address: "test_partner_address-before",
		Port:    1,
		Type:    "sftp",
	}
	testPartnerAfter := &model.Partner{
		Name:    "test_partner_after",
		Address: "test_partner_address-after",
		Type:    "http",
	}

	Convey("Given a partner replacing function", func() {
		err := db.Create(testPartnerBefore)
		So(err, ShouldBeNil)

		id := strconv.FormatUint(testPartnerBefore.ID, 10)

		Convey("When called with an existing name", func() {
			body, err := json.Marshal(testPartnerAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, partnerPath+id, reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": id})

			Convey("Then it should update the partner and reply 'Created'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusCreated)

				existAfter, err := db.Exists(testPartnerAfter)
				So(err, ShouldBeNil)
				So(existAfter, ShouldBeTrue)

				existBefore, err := db.Exists(testPartnerBefore)
				So(err, ShouldBeNil)
				So(existBefore, ShouldBeFalse)

				err = db.Get(testPartnerAfter)
				So(err, ShouldBeNil)
				newID := strconv.FormatUint(testPartnerAfter.ID, 10)
				So(w.Header().Get("Location"), ShouldResemble, partnerPath+newID)
			})
		})

		Convey("When called with an non-existing name", func() {
			body, err := json.Marshal(testPartnerAfter)
			So(err, ShouldBeNil)
			reader := bytes.NewReader(body)
			r, err := http.NewRequest(http.MethodPut, partnerPath+"unknown", reader)
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
			r, err := http.NewRequest(http.MethodPut, partnerPath+id, reader)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			r = mux.SetURLVars(r, map[string]string{"partner": id})

			Convey("Then it should reply 'Bad Request'", func() {
				updatePartner(testLogger, db).ServeHTTP(w, r)
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
