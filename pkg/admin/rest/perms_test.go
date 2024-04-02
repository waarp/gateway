package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestMaskToPerm(t *testing.T) {
	Convey("Testing the permission mask converter", t, func() {
		Convey("Given a permission mask", func() {
			mask := model.PermTransfersRead | model.PermTransfersWrite |
				model.PermServersWrite |
				model.PermPartnersRead | model.PermPartnersDelete |
				model.PermRulesRead | model.PermRulesWrite | model.PermRulesDelete |
				model.PermUsersWrite | model.PermUsersDelete

			Convey("When calling the maskToPerms function", func() {
				perms := maskToPerms(mask)

				Convey("Then it should return the correct permissions", func() {
					exp := &api.Perms{
						Transfers:      "rw-",
						Servers:        "-w-",
						Partners:       "r-d",
						Rules:          "rwd",
						Users:          "-wd",
						Administration: "---",
					}
					So(perms, ShouldResemble, exp)
				})
			})
		})

		Convey("Given a full mask", func() {
			mask := model.PermAll

			Convey("When calling the maskToPerms function", func() {
				perms := maskToPerms(mask)

				Convey("Then it should return the correct permissions", func() {
					exp := &api.Perms{
						Transfers:      "rw-",
						Servers:        "rwd",
						Partners:       "rwd",
						Rules:          "rwd",
						Users:          "rwd",
						Administration: "rwd",
					}
					So(perms, ShouldResemble, exp)
				})
			})
		})
	})
}

func TestPermsToMask(t *testing.T) {
	Convey("Testing the permission mask converter", t, func() {
		Convey("Given a permission mask and a permission string", func() {
			mask := model.PermTransfersRead | model.PermTransfersWrite |
				model.PermServersWrite |
				model.PermPartnersRead | model.PermPartnersDelete |
				model.PermRulesRead | model.PermRulesWrite | model.PermRulesDelete |
				model.PermUsersWrite | model.PermUsersDelete |
				model.PermAdminRead
			perms := api.Perms{
				Transfers:      "-w=w",
				Servers:        "+r",
				Partners:       "-d",
				Rules:          "-wd+w",
				Users:          "=rw+d",
				Administration: "-r+rw",
			}

			Convey("When calling the permsToMask function", func() {
				newMask, err := permsToMask(mask, &perms)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct mask", func() {
					exp := model.PermTransfersWrite |
						model.PermServersWrite | model.PermServersRead |
						model.PermPartnersRead |
						model.PermRulesRead | model.PermRulesWrite |
						model.PermUsersRead | model.PermUsersWrite | model.PermUsersDelete |
						model.PermAdminRead | model.PermAdminWrite

					actual := maskToPerms(newMask)
					expected := maskToPerms(exp)
					So(actual, ShouldResemble, expected)
				})
			})
		})
	})
}

func TestPermMiddleware(t *testing.T) {
	Convey("Given a database with 2 users", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_perm_middleware")
		db := database.TestDatabase(c)

		success := model.User{
			Username:     "success",
			PasswordHash: hash("success"),
			Permissions:  model.PermAll,
		}
		So(db.Insert(&success).Run(), ShouldBeNil)
		fail := model.User{
			Username:     "fail",
			PasswordHash: hash("fail"),
			Permissions:  0,
		}
		So(db.Insert(&fail).Run(), ShouldBeNil)

		Convey("Given a dummy handler", func() {
			f := func(*log.Logger, *database.DB) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}
			}
			router := mux.NewRouter()
			fact := makeHandlerFactory(logger, db, router)

			fact("/", f, model.PermUsersRead, http.MethodGet)

			Convey("When sending a request", func() {
				w := &httptest.ResponseRecorder{}
				r, err := http.NewRequest(http.MethodGet, "/", nil)
				So(err, ShouldBeNil)

				Convey("If the user is authorized", func() {
					r.SetBasicAuth(success.Username, success.PasswordHash)

					Convey("Then it should reply 'OK'", func() {
						router.ServeHTTP(w, r)
						res := w.Result()

						defer res.Body.Close()

						So(res.StatusCode, ShouldEqual, http.StatusOK)
					})
				})

				Convey("If the user is NOT authorized", func() {
					r.SetBasicAuth(fail.Username, fail.PasswordHash)

					Convey("Then it should reply 'FORBIDDEN'", func() {
						router.ServeHTTP(w, r)
						res := w.Result()

						defer res.Body.Close()

						So(res.StatusCode, ShouldEqual, http.StatusForbidden)
					})
				})
			})
		})
	})
}
