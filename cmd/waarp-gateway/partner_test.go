package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPartnerCreate(t *testing.T) {
	testPartner := &model.Partner{
		Name:    "test_partner_create",
		Address: "test_partner_create_address",
		Port:    1,
		Type:    "sftp",
	}
	path := admin.RestURI + admin.PartnersURI
	createHandler := testHandler(http.MethodPost, path, testPartner, nil, nil,
		http.StatusCreated)

	Convey("Testing the partner creation function", t, func() {
		server := httptest.NewServer(createHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}

		Reset(server.Close)

		Convey("Given correct flags", func() {
			p := partnerCreateCommand{}
			args := []string{"-n", testPartner.Name,
				"-a", testPartner.Address,
				"-p", strconv.Itoa(int(testPartner.Port)),
				"-t", testPartner.Type,
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given incorrect flags", func() {
			p := partnerCreateCommand{}
			args := []string{"-n", "incorrect",
				"-a", "incorrect",
				"-p", "1000",
				"-t", "sftp",
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusBadRequest))
			})
		})
	})
}

func TestPartnerSelect(t *testing.T) {
	testPartner1 := &model.Partner{
		Name:    "test_partner_select1",
		Address: "test_partner_select1_address",
		Port:    2,
		Type:    "type3",
	}
	testPartner2 := &model.Partner{
		Name:    "test_partner_select2",
		Address: "test_partner_select3_address",
		Port:    4,
		Type:    "type1",
	}
	testPartner3 := &model.Partner{
		Name:    "test_partner_select3",
		Address: "test_partner_select4_address",
		Port:    1,
		Type:    "type2",
	}
	testPartner4 := &model.Partner{
		Name:    "test_partner_select4",
		Address: "test_partner_select2_address",
		Port:    3,
		Type:    "type4",
	}
	path := admin.RestURI + admin.PartnersURI

	Convey("Testing the partner listing function", t, func() {
		p := partnerListCommand{}

		Convey("Given no flags", func() {
			arrayResult := []*model.Partner{testPartner1, testPartner2, testPartner3, testPartner4}
			expectedResults := map[string][]*model.Partner{"partners": arrayResult}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				nil, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a limit flag", func() {
			arrayResult := []*model.Partner{testPartner1, testPartner2}
			expectedResults := map[string][]*model.Partner{"partners": arrayResult}
			expectedParams := url.Values{"limit": []string{"2"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-l", "2"}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			So(p.Limit, ShouldEqual, 2)
			err = p.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an offset flag", func() {
			arrayResult := []*model.Partner{testPartner3, testPartner4}
			expectedResults := map[string][]*model.Partner{"partners": arrayResult}
			expectedParams := url.Values{"offset": []string{"2"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-o", "2"}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a sort flag", func() {
			arrayResult := []*model.Partner{testPartner3, testPartner4, testPartner2, testPartner3}
			expectedResults := map[string][]*model.Partner{"partners": arrayResult}
			expectedParams := url.Values{"sortby": []string{"address"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-s", "address"}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an order flag", func() {
			arrayResult := []*model.Partner{testPartner4, testPartner3, testPartner2, testPartner1}
			expectedResults := map[string][]*model.Partner{"partners": arrayResult}
			expectedParams := url.Values{"order": []string{"desc"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-d"}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an address flag", func() {
			arrayResult := []*model.Partner{testPartner1, testPartner3}
			expectedResults := map[string][]*model.Partner{"partners": arrayResult}
			expectedParams := url.Values{"address": []string{testPartner1.Address, testPartner3.Address}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-a", testPartner1.Address, "-a", testPartner3.Address}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Reset(server.Close)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a type flag", func() {
			arrayResult := []*model.Partner{testPartner2, testPartner4}
			expectedResults := map[string][]*model.Partner{"partners": arrayResult}
			expectedParams := url.Values{"type": []string{testPartner2.Type, testPartner4.Type}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusCreated)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-t", testPartner2.Type, "-t", testPartner4.Type}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestPartnerGet(t *testing.T) {
	testPartner := &model.Partner{
		Name:    "test_partner_get",
		Address: "test_partner_get_address",
		Port:    1,
		Type:    "sftp",
	}

	Convey("Testing the partner creation function", t, func() {
		path := admin.RestURI + admin.PartnersURI + "/" + testPartner.Name
		getHandler := testHandler(http.MethodGet, path, nil, testPartner, nil,
			http.StatusOK)

		server := httptest.NewServer(getHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}

		Reset(server.Close)

		Convey("Given a correct name", func() {
			p := partnerGetCommand{}
			args := []string{testPartner.Name}
			err := p.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect name", func() {
			p := partnerGetCommand{}
			args := []string{"unknown"}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given no name", func() {
			p := partnerGetCommand{}
			args := []string{}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, "missing partner name")
			})
		})
	})
}

func TestPartnerDelete(t *testing.T) {

	Convey("Testing the partner deletion function", t, func() {
		path := admin.RestURI + admin.PartnersURI + "/testPartner"
		getHandler := testHandler(http.MethodDelete, path, nil, nil, nil,
			http.StatusNoContent)

		server := httptest.NewServer(getHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}

		Reset(server.Close)

		Convey("Given a correct name", func() {
			p := partnerDeleteCommand{}
			args := []string{"testPartner"}
			err := p.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect name", func() {
			p := partnerDeleteCommand{}
			args := []string{"unknown"}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given a no name", func() {
			p := partnerDeleteCommand{}
			args := []string{}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, "missing partner name")
			})
		})
	})
}

func TestPartnerUpdate(t *testing.T) {
	testPartnerAfter := &model.Partner{
		Name:    "test_partner_update_after",
		Address: "test_partner_update_after_address",
		Port:    1,
		Type:    "sftp",
	}
	path := admin.RestURI + admin.PartnersURI + "/testPartner"
	createHandler := testHandler(http.MethodPatch, path, testPartnerAfter, nil, nil,
		http.StatusCreated)

	Convey("Testing the partner update function", t, func() {
		server := httptest.NewServer(createHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}

		Reset(server.Close)

		Convey("Given correct flags", func() {
			p := partnerUpdateCommand{}
			args := []string{"-n", testPartnerAfter.Name,
				"-a", testPartnerAfter.Address,
				"-p", strconv.Itoa(int(testPartnerAfter.Port)),
				"-t", testPartnerAfter.Type,
				"testPartner",
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given incorrect flags", func() {
			p := partnerUpdateCommand{}
			args := []string{"-n", "incorrect",
				"-a", "incorrect",
				"-p", "1000",
				"-t", "sftp",
				"testPartner",
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusBadRequest))
			})
		})

		Convey("Given no name", func() {
			p := partnerUpdateCommand{}
			args := []string{"-n", testPartnerAfter.Name,
				"-a", testPartnerAfter.Address,
				"-p", strconv.Itoa(int(testPartnerAfter.Port)),
				"-t", testPartnerAfter.Type,
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, "missing partner name")
			})
		})
	})
}

func TestDisplayPartner(t *testing.T) {

	Convey("Given a partner", t, func() {
		testPartner := &model.Partner{
			Name:    "test_partner",
			Address: "test_partner_address",
			Port:    1,
			Type:    "sftp",
		}

		Convey("When calling the 'displayPartner' function", func() {
			out, err := ioutil.TempFile(".", "waarp_gateway")
			So(err, ShouldBeNil)
			err = displayPartner(out, testPartner)
			So(err, ShouldBeNil)
			err = out.Close()
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove(out.Name())
			})

			Convey("Then it should display the partner correctly", func() {
				in, err := os.Open(out.Name())
				So(err, ShouldBeNil)
				result, err := ioutil.ReadAll(in)
				So(err, ShouldBeNil)

				expected :=
					"Partner '" + testPartner.Name + "':\n" +
						"├─Address: " + testPartner.Address + "\n" +
						"├─Port: " + strconv.Itoa(int(testPartner.Port)) + "\n" +
						"└─Type: " + testPartner.Type + "\n"

				So(string(result), ShouldEqual, expected)
			})
		})
	})
}
