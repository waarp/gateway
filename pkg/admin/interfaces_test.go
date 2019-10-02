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

const interfacePath = RestURI + InterfacesURI + "/"

func TestListInterfaces(t *testing.T) {
	logger := log.NewLogger("rest_interface_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]model.Interface) {
		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})

		Convey("Then the response body should contain an array "+
			"of the requested interfaces in JSON format", func() {

			response := map[string][]model.Interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)

			if err != nil {
				So(w.Body.String(), ShouldBeBlank)
			}
			So(response, ShouldResemble, expected)
		})
	}

	Convey("Given the interfaces listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listInterfaces(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]model.Interface{}

		Convey("Given a database with 4 interfaces", func() {
			interface1 := model.Interface{
				ID:   1,
				Name: "interface1",
				Type: "sftp",
				Port: 1,
			}
			interface2 := model.Interface{
				ID:   2,
				Name: "interface2",
				Type: "sftp",
				Port: 1,
			}
			interface3 := model.Interface{
				ID:   3,
				Name: "interface3",
				Type: "http",
				Port: 1,
			}
			interface4 := model.Interface{
				ID:   4,
				Name: "interface4",
				Type: "r66",
				Port: 1,
			}

			err := db.Create(&interface1)
			So(err, ShouldBeNil)
			err = db.Create(&interface2)
			So(err, ShouldBeNil)
			err = db.Create(&interface3)
			So(err, ShouldBeNil)
			err = db.Create(&interface4)
			So(err, ShouldBeNil)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, interfacePath, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["interfaces"] = []model.Interface{interface1, interface2,
						interface3, interface4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, interfacePath+"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["interfaces"] = []model.Interface{interface1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, interfacePath+"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["interfaces"] = []model.Interface{interface2, interface3,
						interface4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet,
					interfacePath+"?sortby=type&order=desc", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["interfaces"] = []model.Interface{interface1, interface2,
						interface4, interface3}
					check(w, expected)
				})
			})

			Convey("Given a request with type parameters", func() {
				r, err := http.NewRequest(http.MethodGet,
					interfacePath+"?type=http&type=r66", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["interfaces"] = []model.Interface{interface3, interface4}
					check(w, expected)
				})
			})
		})
	})
}

func TestGetInterface(t *testing.T) {
	logger := log.NewLogger("rest_interface_get_test")

	Convey("Given the interface get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getInterface(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 interface", func() {
			expected := model.Interface{
				ID:   1,
				Name: "existing",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid interface ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, interfacePath+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"interface": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested interface "+
						"in JSON format", func() {

						res := model.Interface{}
						err := json.Unmarshal(w.Body.Bytes(), &res)

						So(err, ShouldBeNil)
						So(res, ShouldResemble, expected)
					})
				})
			})

			Convey("Given a request with a non-existing interface ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, interfacePath+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"interface": "1000"})

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

func TestCreateInterface(t *testing.T) {
	logger := log.NewLogger("rest_interface_create_logger")

	Convey("Given the interface creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createInterface(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 interface", func() {
			existingInterface := model.Interface{
				ID:   1,
				Name: "existing",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&existingInterface)
			So(err, ShouldBeNil)

			Convey("Given a new interface to insert in the database", func() {
				newInterface := model.Interface{
					ID:   2,
					Name: "new_interface",
					Type: "sftp",
					Port: 2,
				}

				Convey("Given that the new interface is valid for insertion", func() {
					body, err := json.Marshal(newInterface)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, interfacePath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new interface", func() {

							location := w.Header().Get("Location")
							expected := interfacePath +
								strconv.FormatUint(newInterface.ID, 10)
							So(location, ShouldEqual, expected)
						})

						Convey("Then the new interface should be inserted in "+
							"the database", func() {
							exist, err := db.Exists(&newInterface)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing interface should still be "+
							"present as well", func() {
							exist, err := db.Exists(&existingInterface)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})
					})
				})

				Convey("Given that the new interface's ID already exist", func() {
					newInterface.ID = existingInterface.ID

					body, err := json.Marshal(newInterface)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, interfacePath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message "+
							"stating that the ID already exist", func() {

							So(w.Body.String(), ShouldEqual,
								"An interface with the same ID or name already exist\n")
						})

						Convey("Then the new interface should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newInterface)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new interface's name already exist", func() {
					newInterface.Name = existingInterface.Name

					body, err := json.Marshal(newInterface)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, interfacePath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the name already exist", func() {

							So(w.Body.String(), ShouldEqual, "An interface with "+
								"the same ID or name already exist\n")
						})

						Convey("Then the new interface should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newInterface)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})
			})
		})
	})
}

func TestDeleteInterface(t *testing.T) {
	logger := log.NewLogger("rest_interface_delete_test")

	Convey("Given the interface deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteInterface(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 interface", func() {
			existing := model.Interface{
				ID:   1,
				Name: "existing",
				Type: "sftp",
				Port: 1,
			}
			err := db.Create(&existing)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid interface ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, interfacePath+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"interface": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the interface should no longer be present "+
						"in the database", func() {

						exist, err := db.Exists(&existing)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})

			Convey("Given a request with a non-existing interface ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, interfacePath+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"interface": "1000"})

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

func TestUpdateInterface(t *testing.T) {
	logger := log.NewLogger("rest_interface_update_logger")

	Convey("Given the interface updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateInterface(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 interfaces", func() {
			old := model.Interface{
				ID:   1,
				Name: "old",
				Type: "sftp",
				Port: 1,
			}
			other := model.Interface{
				ID:   2,
				Name: "other",
				Type: "sftp",
				Port: 2,
			}
			err := db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the interface with", func() {

				Convey("Given a new name", func() {
					update := struct{ Name string }{Name: "update"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					expected := model.Interface{
						ID:   old.ID,
						Name: update.Name,
						Type: "sftp",
						Port: old.Port,
					}

					checkValidUpdate(db, w, http.MethodPatch, interfacePath,
						id, "interface", body, handler, &old, &expected)
				})

				Convey("Given an already existing name", func() {
					update := struct{ Name string }{Name: other.Name}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "An interface with the same name already exist\n"
					checkInvalidUpdate(db, handler, w, body, interfacePath, id,
						"interface", &old, msg)
				})

				Convey("Given an invalid type", func() {
					update := struct{ Type string }{Type: "not_a_type"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "The interface's type must be one of [http r66 sftp]\n"
					checkInvalidUpdate(db, handler, w, body, interfacePath, id,
						"interface", &old, msg)
				})
			})
		})
	})
}

func TestReplaceInterface(t *testing.T) {
	logger := log.NewLogger("rest_interface_replace_logger")

	Convey("Given the interface replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateInterface(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 interfaces", func() {
			old := model.Interface{
				ID:   1,
				Name: "old",
				Type: "sftp",
				Port: 1,
			}
			other := model.Interface{
				ID:   2,
				Name: "other",
				Type: "sftp",
				Port: 2,
			}
			err := db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given a valid new interface", func() {
				replace := struct {
					Name, Type string
					Port       uint16
				}{
					Name: "replace",
					Type: "sftp",
					Port: 3,
				}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.Interface{
					ID:   old.ID,
					Name: replace.Name,
					Type: replace.Type,
					Port: replace.Port,
				}

				checkValidUpdate(db, w, http.MethodPut, interfacePath,
					id, "interface", body, handler, &old, &expected)
			})

			Convey("Given a non-existing interface ID parameter", func() {
				replace := struct{}{}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPut, interfacePath+"1000",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"interface": "1000"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}
