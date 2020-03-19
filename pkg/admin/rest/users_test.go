package rest

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
	"golang.org/x/crypto/bcrypt"
)

const usersURI = "http://localhost:8080" + APIPath + UsersPath + "/"

func TestGetUser(t *testing.T) {
	logger := log.NewLogger("rest_user_get_test")

	Convey("Given the user get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 user", func() {
			expected := &model.User{
				Username: "existing",
				Password: []byte("existing"),
			}
			So(db.Create(expected), ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid user ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, usersURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"user": id})

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

						exp, err := json.Marshal(FromUser(expected))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing user ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, usersURI+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"user": "1000"})

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

func TestListUsers(t *testing.T) {
	logger := log.NewLogger("rest_user_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]OutUser) {
		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})

		Convey("Then the response body should contain an array "+
			"of the requested users in JSON format", func() {

			exp, err := json.Marshal(expected)
			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldResemble, string(exp)+"\n")
		})
	}

	Convey("Given the user listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listUsers(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutUser{}

		Convey("Given a database with 4 users", func() {
			So(db.Delete(&model.User{Username: "admin"}), ShouldBeNil)

			u1 := &model.User{
				Username: "user1",
				Password: []byte("user1"),
			}
			u2 := &model.User{
				Username: "user2",
				Password: []byte("user2"),
			}
			u3 := &model.User{
				Username: "user3",
				Password: []byte("user3"),
			}
			u4 := &model.User{
				Username: "user4",
				Password: []byte("user4"),
			}

			So(db.Create(u1), ShouldBeNil)
			So(db.Create(u2), ShouldBeNil)
			So(db.Create(u3), ShouldBeNil)
			So(db.Create(u4), ShouldBeNil)

			user1 := *FromUser(u1)
			user2 := *FromUser(u2)
			user3 := *FromUser(u3)
			user4 := *FromUser(u4)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, usersURI, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["users"] = []OutUser{user1, user2, user3, user4}

					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, usersURI+"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["users"] = []OutUser{user1}

					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, usersURI+"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["users"] = []OutUser{user2, user3, user4}

					check(w, expected)
				})
			})

			Convey("Given a request with a sort parameter", func() {
				r, err := http.NewRequest(http.MethodGet, usersURI+"?sort=username-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["users"] = []OutUser{user4, user3,
						user2, user1}

					check(w, expected)
				})
			})
		})
	})
}

func TestCreateUser(t *testing.T) {
	logger := log.NewLogger("rest_user_create_logger")

	Convey("Given the user creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(parent), ShouldBeNil)

			Convey("Given a new user to insert in the database", func() {
				newUser := &InUser{
					Username: "new_user",
					Password: []byte("new_user"),
				}

				Convey("Given that the new user is valid for insertion", func() {
					body, err := json.Marshal(newUser)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, usersURI,
						bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the "+
							"URI of the new user", func() {

							location := w.Header().Get("Location")
							So(location, ShouldStartWith, usersURI)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new user should be inserted in the "+
							"database", func() {
							clearPwd := newUser.Password
							newUser.Password = nil

							test := newUser.ToModel()
							err := db.Get(test)
							So(err, ShouldBeNil)

							err = bcrypt.CompareHashAndPassword(test.Password, clearPwd)
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

func TestDeleteUser(t *testing.T) {
	logger := log.NewLogger("rest_user_delete_test")

	Convey("Given the user deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 user", func() {
			parent := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(parent), ShouldBeNil)

			existing := &model.User{
				Username: "existing",
				Password: []byte("existing"),
			}
			So(db.Create(existing), ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid user ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, usersURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"user": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the user should no longer be present "+
						"in the database", func() {
						err := existing.BeforeInsert(nil)
						So(err, ShouldBeNil)

						exist, err := db.Exists(existing)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})

			Convey("Given a request with a non-existing user ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, usersURI+
					"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"user": "1000"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that the user "+
						"was not found", func() {
						So(w.Body.String(), ShouldEqual, "Record not found\n")
					})
				})
			})
		})
	})
}

func TestUpdateUser(t *testing.T) {
	logger := log.NewLogger("rest_user_update_logger")

	Convey("Given the user updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 users", func() {
			old := &model.User{
				Username: "old",
				Password: []byte("old"),
			}
			other := &model.User{
				Username: "other",
				Password: []byte("other"),
			}
			So(db.Create(old), ShouldBeNil)
			So(db.Create(other), ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the user with", func() {

				Convey("Given a new login", func() {
					update := InUser{
						Username: "update",
						Password: []byte("update"),
					}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, usersURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"user": id})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated user", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, usersURI+id)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the user should have been updated", func() {
							result := &model.User{ID: old.ID}
							err := db.Get(result)

							So(err, ShouldBeNil)
							So(result.Username, ShouldEqual, update.Username)
							So(bcrypt.CompareHashAndPassword(result.Password, update.Password), ShouldBeNil)
						})
					})
				})

				Convey("Given an invalid user ID", func() {
					update := InUser{
						Username: "update",
						Password: []byte("update"),
					}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, usersURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"user": "1000"})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the user was not found", func() {
							So(w.Body.String(), ShouldEqual, "Record not found\n")
						})

						Convey("Then the old user should still exist", func() {
							exist, err := db.Exists(old)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}
