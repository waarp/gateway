package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	. "code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const usersURI = "http://localhost:8080/api/users/"

func TestGetUser(t *testing.T) {
	logger := log.NewLogger("rest_user_get_test")

	Convey("Given the user get handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := getUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 user", func() {
			expected := &model.User{
				Username: "existing",
				Password: []byte("existing"),
			}
			So(db.Insert(expected).Run(), ShouldBeNil)

			Convey("Given a request with the valid username parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"user": expected.Username})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then the body should contain the requested partner "+
						"in JSON format", func() {
						exp, err := json.Marshal(FromUser(expected))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain "+
						"'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})
				})
			})

			Convey("Given a request with a non-existing username parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"user": "toto"})

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
		Convey("Then the response body should contain an array "+
			"of the requested users in JSON format", func() {
			decoder := json.NewDecoder(w.Body)
			var actual map[string][]OutUser
			So(decoder.Decode(&actual), ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})
	}

	Convey("Given the user listing handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := listUsers(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutUser{}

		Convey("Given a database with 4 users", func() {
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

			So(db.Insert(u1).Run(), ShouldBeNil)
			So(db.Insert(u2).Run(), ShouldBeNil)
			So(db.Insert(u3).Run(), ShouldBeNil)
			So(db.Insert(u4).Run(), ShouldBeNil)
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			user1 := *FromUser(u1)
			user2 := *FromUser(u2)
			user3 := *FromUser(u3)
			user4 := *FromUser(u4)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["users"] = []OutUser{user1, user2, user3, user4}

					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["users"] = []OutUser{user1}

					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["users"] = []OutUser{user2, user3, user4}

					check(w, expected)
				})
			})

			Convey("Given a request with a sort parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=username-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)
					expected["users"] = []OutUser{user4, user3, user2, user1}

					check(w, expected)
				})
			})
		})
	})
}

func TestCreateUser(t *testing.T) {
	logger := log.NewLogger("rest_user_create_logger")

	Convey("Given the user creation handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := addUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 user", func() {
			existing := model.User{
				Username: "old",
				Password: []byte("old_password"),
			}
			So(db.Insert(&existing).Run(), ShouldBeNil)
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			Convey("Given a new user to insert in the database", func() {
				body := strings.NewReader(`{
					"username": "toto",
					"password": "password",
					"perms": {
						"transfers": "rw",
						"servers": "rd",
						"partners": "rw",
						"rules": "rwd",
						"users": "r"
					}
				}`)

				Convey("Given that the new user is valid for insertion", func() {
					r, err := http.NewRequest(http.MethodPost, usersURI, body)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the 'Location' header should contain the "+
							"URI of the new user", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, usersURI+"toto")
						})

						Convey("Then the new user should be inserted in the "+
							"database", func() {
							var users model.Users
							So(db.Select(&users).Run(), ShouldBeNil)
							So(users, ShouldHaveLength, 2)

							So(bcrypt.CompareHashAndPassword(users[1].Password,
								[]byte("password")), ShouldBeNil)
							So(users[1], ShouldResemble, model.User{
								ID:       3,
								Owner:    database.Owner,
								Username: "toto",
								Password: users[1].Password,
								Permissions: model.PermTransfersRead | model.PermTransfersWrite |
									model.PermServersRead | model.PermServersDelete |
									model.PermPartnersRead | model.PermPartnersWrite |
									model.PermRulesRead | model.PermRulesWrite | model.PermRulesDelete |
									model.PermUsersRead,
							})
						})

						Convey("Then the existing user should still exist", func() {
							var users model.Users
							So(db.Select(&users).Run(), ShouldBeNil)
							So(users, ShouldContain, existing)
						})
					})
				})
			})
		})
	})
}

func TestDeleteUser(t *testing.T) {
	logger := log.NewLogger("rest_user_delete_test")

	Convey("Given the user deletion handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := deleteUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 users", func() {
			existing := model.User{
				Username: "existing",
				Password: []byte("existing"),
			}
			other := model.User{
				Username: "other",
				Password: []byte("other_password"),
			}
			So(db.Insert(&existing).Run(), ShouldBeNil)
			So(db.Insert(&other).Run(), ShouldBeNil)
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			Convey("Given a request with the valid username parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"user": existing.Username})

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
						var users model.Users
						So(db.Select(&users).Run(), ShouldBeNil)
						So(users, ShouldNotContain, existing)
					})

					Convey("Then the other user should still be present", func() {
						var users model.Users
						So(db.Select(&users).Run(), ShouldBeNil)
						So(users, ShouldContain, other)
					})
				})

				Convey("Given the request using the deleted user as authentication", func() {
					r.SetBasicAuth(existing.Username, "")

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Forbidden error' error", func() {
							So(w.Code, ShouldEqual, http.StatusForbidden)
						})

						Convey("Then the body should contain the error message", func() {
							So(w.Body.String(), ShouldResemble, "user cannot delete self\n")
						})

						Convey("Then the users should still exist in the database", func() {
							var users model.Users
							So(db.Select(&users).Run(), ShouldBeNil)
							So(users, ShouldContain, existing)
							So(users, ShouldContain, other)
						})
					})
				})
			})

			Convey("Given a request with a non-existing username parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"user": "toto"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that the user "+
						"was not found", func() {
						So(w.Body.String(), ShouldEqual, "user 'toto' not found\n")
					})

					Convey("Then the user should still be present in the "+
						"database", func() {
						var users model.Users
						So(db.Select(&users).Run(), ShouldBeNil)
						So(users, ShouldContain, existing)
						So(users, ShouldContain, other)
					})
				})
			})
		})
	})
}

func TestUpdateUser(t *testing.T) {
	logger := log.NewLogger("rest_user_update_logger")

	Convey("Given the user updating handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := updateUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 users", func() {
			old := model.User{
				Username: "old",
				Password: []byte("old_password"),
				Permissions: model.PermTransfersRead | model.PermTransfersWrite |
					model.PermServersRead | model.PermServersDelete |
					model.PermPartnersWrite |
					model.PermRulesRead | model.PermRulesWrite |
					model.PermUsersRead,
			}
			other := model.User{
				Username:    "other",
				Password:    []byte("other_password"),
				Permissions: model.PermAll,
			}
			So(db.Insert(&old).Run(), ShouldBeNil)
			So(db.Insert(&other).Run(), ShouldBeNil)
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			Convey("Given new values to update the user with", func() {
				body := strings.NewReader(`{
					"username": "toto",
					"password": "password",
					"perms": {
						"transfers": "-w",
						"servers": "=rw",
						"partners": "+d",
						"rules": "=rd",
						"users": "+w"
					}
				}`)

				Convey("Given an existing username parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, usersURI+old.Username, body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"user": old.Username})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated user", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, usersURI+"toto")
						})

						Convey("Then the user should have been updated", func() {
							var users model.Users
							So(db.Select(&users).OrderBy("id", true).Run(), ShouldBeNil)
							So(users, ShouldHaveLength, 2)

							So(bcrypt.CompareHashAndPassword(users[0].Password,
								[]byte("password")), ShouldBeNil)
							So(users[0], ShouldResemble, model.User{
								ID:       2,
								Owner:    database.Owner,
								Username: "toto",
								Password: users[0].Password,
								Permissions: model.PermTransfersRead |
									model.PermServersRead | model.PermServersWrite |
									model.PermPartnersWrite | model.PermPartnersDelete |
									model.PermRulesRead | model.PermRulesDelete |
									model.PermUsersRead | model.PermUsersWrite,
							})

							Convey("Then the other user should be unchanged", func() {
								var users model.Users
								So(db.Select(&users).Run(), ShouldBeNil)
								So(users, ShouldContain, other)
							})
						})
					})
				})

				Convey("Given an invalid username parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, usersURI+"toto", body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"user": "toto"})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the user was not found", func() {
							So(w.Body.String(), ShouldEqual, "user 'toto' not found\n")
						})

						Convey("Then the old users should be unchanged", func() {
							var users model.Users
							So(db.Select(&users).Run(), ShouldBeNil)
							So(users, ShouldContain, old)
							So(users, ShouldContain, other)
						})
					})
				})
			})

			Convey("Given that a password is not given", func() {
				body := strings.NewReader(`{"username": "upd_user"}`)

				Convey("Given an existing username parameter", func() {
					r, err := http.NewRequest(http.MethodPut, usersURI+old.Username, body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"user": old.Username})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated user", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, usersURI+"upd_user")
						})

						Convey("Then the user should have been updated but the "+
							"password should stay the same", func() {
							var users model.Users
							So(db.Select(&users).OrderBy("id", true).Run(), ShouldBeNil)
							So(len(users), ShouldEqual, 2)

							So(bcrypt.CompareHashAndPassword(users[0].Password,
								[]byte("old_password")), ShouldBeNil)
							So(maskToPerms(users[0].Permissions), ShouldResemble,
								maskToPerms(old.Permissions))
							So(users[0], ShouldResemble, model.User{
								ID:          2,
								Owner:       database.Owner,
								Username:    "upd_user",
								Password:    users[0].Password,
								Permissions: old.Permissions,
							})
						})

						Convey("Then the other user should be unchanged", func() {
							var users model.Users
							So(db.Select(&users).Run(), ShouldBeNil)
							So(users, ShouldContain, other)
						})
					})
				})
			})

			Convey("Given that a username is not given", func() {
				body := strings.NewReader(`{"password": "upd_password"}`)

				Convey("Given an existing username parameter", func() {
					r, err := http.NewRequest(http.MethodPut, usersURI+old.Username, body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"user": old.Username})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated user", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, usersURI+old.Username)
						})

						Convey("Then the user should have been updated but the "+
							"username should stay the same", func() {
							var users model.Users
							So(db.Select(&users).OrderBy("id", true).Run(), ShouldBeNil)
							So(len(users), ShouldEqual, 2)

							So(bcrypt.CompareHashAndPassword(users[0].Password,
								[]byte("upd_password")), ShouldBeNil)
							So(users[0], ShouldResemble, model.User{
								ID:          2,
								Owner:       database.Owner,
								Username:    "old",
								Password:    users[0].Password,
								Permissions: old.Permissions,
							})
						})
					})
				})
			})
		})
	})
}

func TestReplaceUser(t *testing.T) {
	logger := log.NewLogger("rest_user_replace")

	Convey("Given the user replacing handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := replaceUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 users", func() {
			old := model.User{
				Username: "old",
				Password: []byte("old"),
			}
			other := model.User{
				Username: "other",
				Password: []byte("other"),
			}
			So(db.Insert(&old).Run(), ShouldBeNil)
			So(db.Insert(&other).Run(), ShouldBeNil)
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			Convey("Given new values to update the user with", func() {
				body := strings.NewReader(`{
					"username": "upd_user",
					"password": "upd_password"
				}`)

				Convey("Given an existing username parameter", func() {
					r, err := http.NewRequest(http.MethodPut, usersURI+old.Username, body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"user": old.Username})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated user", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, usersURI+"upd_user")
						})

						Convey("Then the user should have been updated", func() {
							var users model.Users
							So(db.Select(&users).OrderBy("id", true).Run(), ShouldBeNil)
							So(len(users), ShouldEqual, 2)

							So(bcrypt.CompareHashAndPassword(users[0].Password,
								[]byte("upd_password")), ShouldBeNil)
							So(users[0], ShouldResemble, model.User{
								ID:       2,
								Owner:    database.Owner,
								Username: "upd_user",
								Password: users[0].Password,
							})
						})

						Convey("Then the other user should be unchanged", func() {
							var users model.Users
							So(db.Select(&users).Run(), ShouldBeNil)
							So(users, ShouldContain, other)
						})
					})
				})

				Convey("Given an invalid username parameter", func() {
					r, err := http.NewRequest(http.MethodPut, usersURI+"toto", body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"user": "toto"})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should state that "+
							"the user was not found", func() {
							So(w.Body.String(), ShouldEqual, "user 'toto' not found\n")
						})

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the users should be unchanged", func() {
							var users model.Users
							So(db.Select(&users).Run(), ShouldBeNil)
							So(users, ShouldContain, old)
							So(users, ShouldContain, other)
						})
					})
				})
			})

			Convey("Given that a password is not given", func() {
				body := strings.NewReader(`{"username": "upd_user"}`)

				Convey("Given an existing username parameter", func() {
					r, err := http.NewRequest(http.MethodPut, usersURI+old.Username, body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"user": old.Username})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should state that "+
							"a password is required", func() {
							So(w.Body.String(), ShouldEqual,
								"the user password cannot be empty\n")
						})

						Convey("Then it should reply 'BadRequest'", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the users should be unchanged", func() {
							var users model.Users
							So(db.Select(&users).Run(), ShouldBeNil)
							So(users, ShouldContain, old)
							So(users, ShouldContain, other)
						})
					})
				})
			})
		})
	})
}
