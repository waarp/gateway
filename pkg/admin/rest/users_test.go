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
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const usersURI = "http://localhost:8080/api/users/"

func TestGetUser(t *testing.T) {
	Convey("Given the user get handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_user_get_test")
		db := database.TestDatabase(c)
		handler := getUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 users", func() {
			// add a user from another gateway
			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "foobar"
			other := &model.User{
				Username:     "existing",
				PasswordHash: hash("existing1"),
				Permissions:  model.PermTransfersWrite,
			}
			So(db.Insert(other).Run(), ShouldBeNil)
			conf.GlobalConfig.GatewayName = owner

			expected := &model.User{
				Username:     other.Username,
				PasswordHash: hash("existing2"),
				Permissions:  model.PermAll,
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
						exp, err := json.Marshal(DBUserToREST(expected))

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
		logger := testhelpers.TestLogger(c, "rest_user_list_test")
		db := database.TestDatabase(c)
		handler := listUsers(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutUser{}

		Convey("Given a database with 5 users", func() {
			u1 := &model.User{
				Username:     "user1",
				PasswordHash: hash("user1"),
			}
			u2 := &model.User{
				Username:     "user2",
				PasswordHash: hash("user2"),
			}
			u3 := &model.User{
				Username:     "user3",
				PasswordHash: hash("user3"),
			}
			u4 := &model.User{
				Username:     "user4",
				PasswordHash: hash("user4"),
			}

			So(db.Insert(u1).Run(), ShouldBeNil)
			So(db.Insert(u2).Run(), ShouldBeNil)
			So(db.Insert(u3).Run(), ShouldBeNil)
			So(db.Insert(u4).Run(), ShouldBeNil)
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			user1 := *DBUserToREST(u1)
			user2 := *DBUserToREST(u2)
			user3 := *DBUserToREST(u3)
			user4 := *DBUserToREST(u4)

			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "foobar"
			u5 := &model.User{
				Username:     "user5",
				PasswordHash: hash("user5"),
			}
			So(db.Insert(u5).Run(), ShouldBeNil)
			conf.GlobalConfig.GatewayName = owner

			Convey("Given a request with no parameters", func() {
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
	Convey("Given the user creation handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_user_create_logger")
		db := database.TestDatabase(c)
		handler := addUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 user", func() {
			existing := &model.User{
				Username:     "old",
				PasswordHash: hash("old_password"),
			}
			So(db.Insert(existing).Run(), ShouldBeNil)
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			Convey("Given a new user to insert in the database", func() {
				body := strings.NewReader(`{
					"username": "toto",
					"password": "sesame",
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
							So(db.Select(&users).OrderBy("id", true).Run(), ShouldBeNil)
							So(users, ShouldHaveLength, 2)

							So(bcrypt.CompareHashAndPassword([]byte(users[1].PasswordHash),
								[]byte("sesame")), ShouldBeNil)
							So(users[1], ShouldResemble, &model.User{
								ID:           3,
								Owner:        conf.GlobalConfig.GatewayName,
								Username:     "toto",
								PasswordHash: users[1].PasswordHash,
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
	Convey("Given the user deletion handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_user_delete_test")
		db := database.TestDatabase(c)
		handler := deleteUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 users", func() {
			existing := &model.User{
				Username:     "existing",
				PasswordHash: hash("existing"),
			}
			other := &model.User{
				Username:     "other",
				PasswordHash: hash("other_password"),
			}
			So(db.Insert(existing).Run(), ShouldBeNil)
			So(db.Insert(other).Run(), ShouldBeNil)
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
							So(w.Body.String(), ShouldResemble, "a user cannot delete themself\n")
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
	Convey("Given the user updating handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_user_update_logger")
		db := database.TestDatabase(c)
		handler := updateUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 users", func() {
			old := &model.User{
				Username:     "old",
				PasswordHash: hash("old_password"),
				Permissions: model.PermTransfersRead | model.PermTransfersWrite |
					model.PermServersRead | model.PermServersDelete |
					model.PermPartnersWrite |
					model.PermRulesRead | model.PermRulesWrite |
					model.PermUsersRead,
			}
			other := &model.User{
				Username:     "other",
				PasswordHash: hash("other_password"),
				Permissions:  model.PermAll,
			}
			So(db.Insert(old).Run(), ShouldBeNil)
			So(db.Insert(other).Run(), ShouldBeNil)
			So(db.DeleteAll(&model.User{}).Where("username='admin'").Run(), ShouldBeNil)

			Convey("Given new values to update the user with", func() {
				body := strings.NewReader(`{
					"username": "toto",
					"password": "sesame",
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

							So(bcrypt.CompareHashAndPassword([]byte(users[0].PasswordHash),
								[]byte("sesame")), ShouldBeNil)
							So(users[0], ShouldResemble, &model.User{
								ID:           2,
								Owner:        conf.GlobalConfig.GatewayName,
								Username:     "toto",
								PasswordHash: users[0].PasswordHash,
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

							So(bcrypt.CompareHashAndPassword([]byte(users[0].PasswordHash),
								[]byte("old_password")), ShouldBeNil)
							So(model.MaskToPerms(users[0].Permissions), ShouldResemble,
								model.MaskToPerms(old.Permissions))
							So(users[0], ShouldResemble, &model.User{
								ID:           2,
								Owner:        conf.GlobalConfig.GatewayName,
								Username:     "upd_user",
								PasswordHash: users[0].PasswordHash,
								Permissions:  old.Permissions,
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

							So(bcrypt.CompareHashAndPassword([]byte(users[0].PasswordHash),
								[]byte("upd_password")), ShouldBeNil)
							So(users[0], ShouldResemble, &model.User{
								ID:           2,
								Owner:        conf.GlobalConfig.GatewayName,
								Username:     "old",
								PasswordHash: users[0].PasswordHash,
								Permissions:  old.Permissions,
							})
						})
					})
				})
			})
		})
	})
}

func TestReplaceUser(t *testing.T) {
	Convey("Given the user replacing handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_user_replace")
		db := database.TestDatabase(c)
		handler := replaceUser(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 users", func() {
			old := &model.User{
				Username:     "old",
				PasswordHash: hash("old"),
			}
			other := &model.User{
				Username:     "other",
				PasswordHash: hash("other"),
			}
			So(db.Insert(old).Run(), ShouldBeNil)
			So(db.Insert(other).Run(), ShouldBeNil)
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

							So(bcrypt.CompareHashAndPassword([]byte(users[0].PasswordHash),
								[]byte("upd_password")), ShouldBeNil)
							So(users[0], ShouldResemble, &model.User{
								ID:           2,
								Owner:        conf.GlobalConfig.GatewayName,
								Username:     "upd_user",
								PasswordHash: users[0].PasswordHash,
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
