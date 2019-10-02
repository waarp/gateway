package admin

import (
	"bytes"
	"encoding/json"
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

const partnerPath = RestURI + PartnersURI + "/"

func TestListPartners(t *testing.T) {
	logger := log.NewLogger("rest_partner_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]model.Partner) {
		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})

		Convey("Then the response body should contain an array "+
			"of the requested partners in JSON format", func() {

			response := map[string][]model.Partner{}
			err := json.Unmarshal(w.Body.Bytes(), &response)

			So(err, ShouldBeNil)
			So(response, ShouldResemble, expected)
		})
	}

	Convey("Given the partners listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listPartners(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]model.Partner{}

		Convey("Given a database with 4 partners", func() {
			parent := model.Interface{
				ID:   1,
				Name: "parent",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			partner1 := model.Partner{
				ID:          1,
				Name:        "partner1",
				InterfaceID: parent.ID,
				Address:     "address1",
				Port:        1,
			}
			partner2 := model.Partner{
				ID:          2,
				Name:        "partner2",
				InterfaceID: parent.ID,
				Address:     "address3",
				Port:        1,
			}
			partner3 := model.Partner{
				ID:          3,
				Name:        "partner3",
				InterfaceID: parent.ID,
				Address:     "address2",
				Port:        1,
			}
			partner4 := model.Partner{
				ID:          4,
				Name:        "partner4",
				InterfaceID: parent.ID,
				Address:     "address4",
				Port:        1,
			}

			err = db.Create(&partner1)
			So(err, ShouldBeNil)
			err = db.Create(&partner2)
			So(err, ShouldBeNil)
			err = db.Create(&partner3)
			So(err, ShouldBeNil)
			err = db.Create(&partner4)
			So(err, ShouldBeNil)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, partnerPath, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []model.Partner{partner1, partner2,
						partner3, partner4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, partnerPath+"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []model.Partner{partner1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, partnerPath+"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []model.Partner{partner2, partner3,
						partner4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet,
					partnerPath+"?sortby=address&order=desc", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []model.Partner{partner4, partner2,
						partner3, partner1}
					check(w, expected)
				})
			})

			Convey("Given a request with address parameters", func() {
				r, err := http.NewRequest(http.MethodGet,
					partnerPath+"?address=address2&address=address3", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []model.Partner{partner2, partner3}
					check(w, expected)
				})
			})
		})
	})
}

func TestGetPartner(t *testing.T) {
	logger := log.NewLogger("rest_partner_get_test")

	Convey("Given the partner get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getPartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner", func() {
			parent := model.Interface{
				ID:   1,
				Name: "parent",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			expected := model.Partner{
				ID:          1,
				Name:        "existing",
				InterfaceID: parent.ID,
				Address:     "address",
				Port:        1,
			}
			err = db.Create(&expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid partner ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, partnerPath+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"partner": id})

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

						res := model.Partner{}
						err := json.Unmarshal(w.Body.Bytes(), &res)

						So(err, ShouldBeNil)
						So(res, ShouldResemble, expected)
					})
				})
			})

			Convey("Given a request with a non-existing partner ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, partnerPath+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"partner": "1000"})

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

func TestCreatePartner(t *testing.T) {
	logger := log.NewLogger("rest_partner_create_logger")

	Convey("Given the partner creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createPartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner", func() {
			parent := model.Interface{
				ID:   1,
				Name: "parent",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existingPartner := model.Partner{
				ID:          1,
				Name:        "existing",
				InterfaceID: parent.ID,
				Address:     "address1",
				Port:        1,
			}
			err = db.Create(&existingPartner)
			So(err, ShouldBeNil)

			Convey("Given a new partner to insert in the database", func() {
				newPartner := model.Partner{
					ID:          2,
					Name:        "new_partner",
					InterfaceID: parent.ID,
					Address:     "address2",
					Port:        2,
				}

				Convey("Given that the new partner is valid for insertion", func() {
					body, err := json.Marshal(newPartner)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, partnerPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new partner", func() {

							location := w.Header().Get("Location")
							expected := partnerPath + strconv.FormatUint(newPartner.ID, 10)
							So(location, ShouldEqual, expected)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new partner should be inserted in the database", func() {
							exist, err := db.Exists(&newPartner)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing partner should still be present as well", func() {
							exist, err := db.Exists(&existingPartner)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the new partner's ID already exist", func() {
					newPartner.ID = existingPartner.ID

					body, err := json.Marshal(newPartner)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, partnerPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the ID already exist", func() {

							So(w.Body.String(), ShouldEqual,
								"A partner with the same ID already exist\n")
						})

						Convey("Then the new partner should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newPartner)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new partner's name already exist", func() {
					newPartner.Name = existingPartner.Name

					body, err := json.Marshal(newPartner)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, partnerPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the name already exist", func() {

							So(w.Body.String(), ShouldEqual,
								"A partner with the same name already exist for this interface\n")
						})

						Convey("Then the new partner should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newPartner)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given a non-existing interface ID", func() {
					newPartner.InterfaceID = 1000

					body, err := json.Marshal(newPartner)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, partnerPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the interface ID does not exist", func() {

							So(w.Body.String(), ShouldEqual, "No interface found "+
								"with id '1000'\n")
						})

						Convey("Then the new partner should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newPartner)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})
			})
		})
	})
}

func TestDeletePartner(t *testing.T) {
	logger := log.NewLogger("rest_partner_delete_test")

	Convey("Given the partner deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deletePartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner", func() {
			parent := model.Interface{
				ID:   1,
				Name: "parent",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.Partner{
				ID:          1,
				Name:        "existing",
				InterfaceID: parent.ID,
				Address:     "address",
				Port:        1,
			}
			err = db.Create(&existing)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid partner ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, partnerPath+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"partner": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the partner should no longer be present "+
						"in the database", func() {

						exist, err := db.Exists(&existing)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})

			Convey("Given a request with a non-existing partner ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, partnerPath+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"partner": "1000"})

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

func checkInvalidUpdate(db *database.Db, handler http.Handler, w *httptest.ResponseRecorder,
	body []byte, path, id, parameter string, old interface{}, errorMsg string) {

	Convey("When sending the request to the handler", func() {
		r, err := http.NewRequest(http.MethodPatch, path+id, bytes.NewReader(body))
		So(err, ShouldBeNil)
		r = mux.SetURLVars(r, map[string]string{parameter: id})

		handler.ServeHTTP(w, r)

		Convey("Then it should reply with a 'Bad Request' error", func() {
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Then the response body should contain a message stating "+
			"the error", func() {

			So(w.Body.String(), ShouldEqual, errorMsg)
		})

		Convey("Then the old "+parameter+" should stay unchanged", func() {
			exist, err := db.Exists(old)
			So(err, ShouldBeNil)
			So(exist, ShouldBeTrue)
		})
	})
}

func TestUpdatePartner(t *testing.T) {
	logger := log.NewLogger("rest_partner_update_logger")

	Convey("Given the partner updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updatePartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 partners", func() {
			parent := model.Interface{
				ID:   1,
				Name: "parent",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.Partner{
				ID:          1,
				Name:        "old",
				InterfaceID: parent.ID,
				Address:     "address1",
				Port:        1,
			}
			other := model.Partner{
				ID:          2,
				Name:        "other",
				InterfaceID: parent.ID,
				Address:     "address2",
				Port:        2,
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the partner with", func() {

				Convey("Given a new name", func() {
					update := struct{ Name string }{Name: "update"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					expected := model.Partner{
						ID:          old.ID,
						Name:        update.Name,
						InterfaceID: old.InterfaceID,
						Address:     old.Address,
						Port:        old.Port,
					}

					checkValidUpdate(db, w, http.MethodPatch, partnerPath,
						id, "partner", body, handler, &old, &expected)
				})

				Convey("Given an already existing name", func() {
					update := struct{ Name string }{Name: other.Name}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "A partner with the same name already exist for this interface\n"
					checkInvalidUpdate(db, handler, w, body, partnerPath, id,
						"partner", &old, msg)
				})

				Convey("Given a non-existing interface ID", func() {
					update := struct{ InterfaceID uint64 }{InterfaceID: 1000}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "No interface found with id '1000'\n"
					checkInvalidUpdate(db, handler, w, body, partnerPath, id,
						"partner", &old, msg)
				})

				Convey("Given an invalid partner ID parameter", func() {
					update := struct{}{}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, partnerPath+"1000",
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"partner": "1000"})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Not Found' error", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})
					})
				})
			})
		})
	})
}

func TestReplacePartner(t *testing.T) {
	logger := log.NewLogger("rest_partner_replace_logger")

	Convey("Given the partner replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updatePartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 partners", func() {
			parent := model.Interface{
				ID:   1,
				Name: "parent",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.Partner{
				ID:          1,
				Name:        "old",
				InterfaceID: parent.ID,
				Address:     "address1",
				Port:        1,
			}
			other := model.Partner{
				ID:          2,
				Name:        "other",
				InterfaceID: parent.ID,
				Address:     "address2",
				Port:        2,
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given a valid new partner", func() {
				replace := struct {
					Name, Address string
					Port          uint16
					InterfaceID   uint64
				}{
					Name:        "replace",
					Address:     "address3",
					Port:        3,
					InterfaceID: parent.ID,
				}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.Partner{
					ID:      old.ID,
					Name:    replace.Name,
					Address: replace.Address,
					Port:    replace.Port,
				}

				checkValidUpdate(db, w, http.MethodPut, partnerPath,
					id, "partner", body, handler, &old, &expected)
			})

			Convey("Given a non-existing partner ID parameter", func() {
				replace := struct{}{}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPut, partnerPath+"1000",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"partner": "1000"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}
