package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const testAuthoritiesURI = "http://localhost:8080/api/authorities/"

func jsonEscape(input any) string {
	res, err := json.Marshal(input)
	So(err, ShouldBeNil)

	return string(res)
}

func TestAddAuthAuthority(t *testing.T) {
	Convey("Given the authority add handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_add_authority")
		db := database.TestDatabase(c)
		h := addAuthAuthority(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with an existing authority", func() {
			existing := &model.Authority{
				Name:           "existing",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.LocalhostCert,
				ValidHosts:     []string{"1.2.3.4", "example.com"},
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a valid new authority to insert in the database", func() {
				body := strings.NewReader(`{
					"name":           "new_authority",
					"type":           "` + auth.AuthorityTLS + `",
					"publicIdentity": ` + jsonEscape(testhelpers.OtherLocalhostCert) + `,
					"validHosts":     ["9.8.7.6", "waarp.org"]
				}`)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPost, testAuthoritiesURI, body)
					So(err, ShouldBeNil)

					h.ServeHTTP(w, r)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)

						Convey("Then the 'Location' header should have been set", func() {
							loc := w.Header().Get("Location")
							So(loc, ShouldEqual, testAuthoritiesURI+"new_authority")
						})

						Convey("Then the new authority should have been inserted", func() {
							var authorities model.Authorities
							So(db.Select(&authorities).OrderBy("id", true).Run(), ShouldBeNil)
							So(authorities, ShouldHaveLength, 2)
							So(authorities[0], ShouldResemble, existing)

							So(authorities[1], ShouldResemble, &model.Authority{
								ID:             2,
								Name:           "new_authority",
								Type:           auth.AuthorityTLS,
								PublicIdentity: testhelpers.OtherLocalhostCert,
								ValidHosts:     []string{"9.8.7.6", "waarp.org"},
							})
						})
					})
				})
			})
		})
	})
}

func TestGetAuthAuthority(t *testing.T) {
	Convey("Given the authority get handler", t, func(c C) {
		db := database.TestDatabase(c)
		logger := testhelpers.TestLogger(c, "rest_get_authority")
		h := getAuthAuthority(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with an existing authority", func() {
			existing := &model.Authority{
				Name:           "existing",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.LocalhostCert,
				ValidHosts:     []string{"1.2.3.4", "example.com"},
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a valid authority name", func() {
				url := testAuthoritiesURI + "/" + existing.Name
				r, err := http.NewRequest(http.MethodGet, url, nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"authority": existing.Name})

				Convey("When sending the request to the handler", func() {
					h.ServeHTTP(w, r)

					Convey("Then the response body should contain the requested authority", func() {
						So(w.Body.String(), ShouldEqual, "{"+
							`"name":"`+existing.Name+`",`+
							`"type":"`+existing.Type+`",`+
							`"publicIdentity":`+jsonEscape(existing.PublicIdentity)+`,`+
							`"validHosts":`+jsonEscape(existing.ValidHosts)+
							"}\n")
					})

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)

						Convey("Then the Content-Type should be 'application/json'", func() {
							contentType := w.Header().Get("Content-Type")

							So(contentType, ShouldEqual, "application/json")
						})
					})
				})
			})

			Convey("Given an invalid authority name", func() {
				url := testAuthoritiesURI + "/unknown"
				r, err := http.NewRequest(http.MethodGet, url, nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"authority": "unknown"})

				Convey("When sending the request to the handler", func() {
					h.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)

						Convey("Then the response body should contain an error", func() {
							So(w.Body.String(), ShouldEqual,
								"authentication authority not found\n")
						})
					})
				})
			})
		})
	})
}

func TestListAuthAuthority(t *testing.T) {
	Convey("Given the authority get handler", t, func(c C) {
		db := database.TestDatabase(c)
		logger := testhelpers.TestLogger(c, "rest_list_authority")
		h := listAuthAuthorities(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 existing authorities", func() {
			existing1 := &model.Authority{
				Name:           "existing1",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.LocalhostCert,
				ValidHosts:     []string{"1.1.1.1", "2.2.2.2"},
			}
			So(db.Insert(existing1).Run(), ShouldBeNil)

			existing2 := &model.Authority{
				Name:           "existing2",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.OtherLocalhostCert,
				ValidHosts:     []string{"3.3.3.3", "4.4.4.4"},
			}
			So(db.Insert(existing2).Run(), ShouldBeNil)

			Convey("When sending the request", func() {
				r, err := http.NewRequest(http.MethodGet, testAuthoritiesURI, nil)
				So(err, ShouldBeNil)

				h.ServeHTTP(w, r)

				Convey("Then the response body should contain the requested authorities", func() {
					So(w.Body.String(), ShouldEqual, `{"authorities":[{`+
						`"name":"`+existing1.Name+`",`+
						`"type":"`+existing1.Type+`",`+
						`"publicIdentity":`+jsonEscape(existing1.PublicIdentity)+`,`+
						`"validHosts":`+jsonEscape(existing1.ValidHosts)+
						"},{"+
						`"name":"`+existing2.Name+`",`+
						`"type":"`+existing2.Type+`",`+
						`"publicIdentity":`+jsonEscape(existing2.PublicIdentity)+`,`+
						`"validHosts":`+jsonEscape(existing2.ValidHosts)+
						"}]}\n")
				})

				Convey("Then it should reply 'OK'", func() {
					So(w.Code, ShouldEqual, http.StatusOK)

					Convey("Then the Content-Type should be 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})
				})
			})
		})
	})
}

func TestUpdateAuthAuthority(t *testing.T) {
	Convey("Given the authority update handler", t, func(c C) {
		db := database.TestDatabase(c)
		logger := testhelpers.TestLogger(c, "rest_update_authority")
		h := updateAuthAuthority(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with an existing authority", func() {
			existing := &model.Authority{
				Name:           "existing",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.LocalhostCert,
				ValidHosts:     []string{"1.2.3.4", "example.com"},
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			url := testAuthoritiesURI + "/" + existing.Name

			for i, test := range []struct {
				desc            string
				hosts, expected []string
			}{
				{desc: "no new hosts", hosts: nil, expected: existing.ValidHosts},
				{desc: "nil hosts", hosts: nil, expected: []string{}},
				{desc: "empty new hosts", hosts: []string{}, expected: []string{}},
				{desc: "some new hosts", hosts: []string{"9.8.7.6"}, expected: []string{"9.8.7.6"}},
			} {
				Convey("Given "+test.desc, func() {
					body := &bytes.Buffer{}
					encoder := json.NewEncoder(body)
					input := map[string]any{
						"name":           "new_authority",
						"publicIdentity": testhelpers.OtherLocalhostCert,
					}

					if i != 0 {
						input["validHosts"] = test.hosts
					}

					So(encoder.Encode(input), ShouldBeNil)

					r, err := http.NewRequest(http.MethodPatch, url, body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"authority": existing.Name})

					Convey("When sending the request to the handler", func() {
						h.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)

							Convey("Then the 'Location' header should have been set", func() {
								loc := w.Header().Get("Location")
								So(loc, ShouldEqual, testAuthoritiesURI+"new_authority")
							})

							Convey("Then the authority should have been partially updated", func() {
								var authorities model.Authorities
								So(db.Select(&authorities).OrderBy("id", true).Run(), ShouldBeNil)
								So(authorities, ShouldHaveLength, 1)
								So(authorities[0], ShouldResemble, &model.Authority{
									ID:             existing.ID,
									Name:           "new_authority",
									Type:           existing.Type,
									PublicIdentity: testhelpers.OtherLocalhostCert,
									ValidHosts:     test.expected,
								})
							})
						})
					})
				})
			}
		})
	})
}

func TestReplaceAuthAuthority(t *testing.T) {
	Convey("Given the authority replace handler", t, func(c C) {
		db := database.TestDatabase(c)
		logger := testhelpers.TestLogger(c, "rest_replace_authority")
		h := replaceAuthAuthority(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with an existing authority", func() {
			existing := &model.Authority{
				Name:           "existing",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.LocalhostCert,
				ValidHosts:     []string{"1.2.3.4", "example.com"},
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a valid authority name", func() {
				url := testAuthoritiesURI + "/" + existing.Name

				body := strings.NewReader(`{
					"name":           "new_authority",
					"type":           "` + auth.AuthorityTLS + `",
					"publicIdentity": ` + jsonEscape(testhelpers.OtherLocalhostCert) +
					`}`)

				r, err := http.NewRequest(http.MethodPatch, url, body)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"authority": existing.Name})

				Convey("When sending the request to the handler", func() {
					h.ServeHTTP(w, r)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)

						Convey("Then the 'Location' header should have been set", func() {
							loc := w.Header().Get("Location")
							So(loc, ShouldEqual, testAuthoritiesURI+"new_authority")
						})

						Convey("Then the authority should have been replaced", func() {
							var authorities model.Authorities
							So(db.Select(&authorities).OrderBy("id", true).Run(), ShouldBeNil)
							So(authorities, ShouldHaveLength, 1)
							So(authorities[0], ShouldResemble, &model.Authority{
								ID:             existing.ID,
								Name:           "new_authority",
								Type:           auth.AuthorityTLS,
								PublicIdentity: testhelpers.OtherLocalhostCert,
								ValidHosts:     []string{},
							})
						})
					})
				})
			})
		})
	})
}

func TestDeleteAuthAuthority(t *testing.T) {
	Convey("Given the authority delete handler", t, func(c C) {
		db := database.TestDatabase(c)
		logger := testhelpers.TestLogger(c, "rest_delete_authority")
		h := deleteAuthAuthority(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 existing authorities", func() {
			existing1 := &model.Authority{
				Name:           "existing1",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.LocalhostCert,
				ValidHosts:     []string{"1.1.1.1", "2.2.2.2"},
			}
			So(db.Insert(existing1).Run(), ShouldBeNil)

			existing2 := &model.Authority{
				Name:           "existing2",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.OtherLocalhostCert,
				ValidHosts:     []string{"3.3.3.3", "4.4.4.4"},
			}
			So(db.Insert(existing2).Run(), ShouldBeNil)

			Convey("Given a valid authority name", func() {
				url := testAuthoritiesURI + "/" + existing1.Name
				r, err := http.NewRequest(http.MethodGet, url, nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"authority": existing1.Name})

				Convey("When sending the request to the handler", func() {
					h.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then it should have deleted the authority and its hosts", func() {
						var authorities model.Authorities
						So(db.Select(&authorities).Run(), ShouldBeNil)
						So(authorities, ShouldHaveLength, 1)
						So(authorities[0], ShouldResemble, existing2)

						var hosts model.Hosts
						So(db.Select(&hosts).Run(), ShouldBeNil)
						So(hosts, ShouldHaveLength, 2)
						So(hosts[0].AuthorityID, ShouldEqual, existing2.ID)
						So(hosts[1].AuthorityID, ShouldEqual, existing2.ID)
					})
				})
			})

			Convey("Given an invalid authority name", func() {
				url := testAuthoritiesURI + "/unknown"
				r, err := http.NewRequest(http.MethodGet, url, nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"authority": "unknown"})

				Convey("When sending the request to the handler", func() {
					h.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)

						Convey("Then the response body should contain an error", func() {
							So(w.Body.String(), ShouldEqual,
								"authentication authority not found\n")
						})
					})
				})
			})
		})
	})
}
