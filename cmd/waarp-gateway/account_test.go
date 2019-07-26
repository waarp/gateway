package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testPartnerName      = "test_partner"
	incorrectPartnerName = "incorrect"
)

func TestAccountCreate(t *testing.T) {
	testAccount := &model.Account{
		Username: "test_account_create",
		Password: []byte("test_account_create_password"),
	}
	path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName + admin.AccountsURI
	createHandler := testHandler(http.MethodPost, path, testAccount, nil, nil,
		http.StatusCreated)

	Convey("Testing the account creation function", t, func() {
		server := httptest.NewServer(createHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		a := accountCreateCommand{accountCommand: &accountCommand{Partner: testPartnerName}}

		Reset(server.Close)

		Convey("Given correct flags", func() {
			args := []string{"-n", testAccount.Username,
				"-p", string(testAccount.Password),
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given incorrect flags", func() {
			args := []string{"-n", "incorrect",
				"-p", "incorrect",
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusBadRequest))
			})
		})

		Convey("Given an incorrect partner name", func() {
			a.Partner = incorrectPartnerName
			args := []string{"-n", testAccount.Username,
				"-p", string(testAccount.Password),
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestAccountSelect(t *testing.T) {
	testAccount1 := &model.Account{
		Username: "test_account_select1",
		Password: []byte("test_account_select1_password"),
	}
	testAccount2 := &model.Account{
		Username: "test_account_select2",
		Password: []byte("test_account_select2_password"),
	}
	testAccount3 := &model.Account{
		Username: "test_account_select3",
		Password: []byte("test_account_select3_password"),
	}
	testAccount4 := &model.Account{
		Username: "test_account_select4",
		Password: []byte("test_account_select4_password"),
	}
	path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName + admin.AccountsURI

	Convey("Testing the account listing function", t, func() {
		a := accountListCommand{accountCommand: &accountCommand{Partner: testPartnerName}}

		Convey("Given no flags", func() {
			arrayResult := []*model.Account{testAccount1, testAccount2, testAccount3, testAccount4}
			expectedResults := map[string][]*model.Account{"accounts": arrayResult}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				nil, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a limit flag", func() {
			arrayResult := []*model.Account{testAccount1, testAccount2}
			expectedResults := map[string][]*model.Account{"accounts": arrayResult}
			expectedParams := url.Values{"limit": []string{"2"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-l", "2"}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			So(a.Limit, ShouldEqual, 2)
			err = a.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an offset flag", func() {
			arrayResult := []*model.Account{testAccount3, testAccount4}
			expectedResults := map[string][]*model.Account{"accounts": arrayResult}
			expectedParams := url.Values{"offset": []string{"2"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-o", "2"}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a sort flag", func() {
			arrayResult := []*model.Account{testAccount3, testAccount1, testAccount4, testAccount2}
			expectedResults := map[string][]*model.Account{"accounts": arrayResult}
			expectedParams := url.Values{"sortby": []string{"username"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-s", "username"}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an order flag", func() {
			arrayResult := []*model.Account{testAccount4, testAccount3, testAccount2, testAccount1}
			expectedResults := map[string][]*model.Account{"accounts": arrayResult}
			expectedParams := url.Values{"order": []string{"desc"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-d"}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect partner name", func() {
			selectHandler := testHandler(http.MethodGet, path, nil,
				http.StatusText(http.StatusNotFound), nil, http.StatusNotFound)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			a.Partner = incorrectPartnerName
			args := []string{}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestAccountDelete(t *testing.T) {
	testAccountName := "test_account_delete"

	Convey("Testing the account deletion function", t, func() {
		path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName +
			admin.AccountsURI + "/" + testAccountName
		getHandler := testHandler(http.MethodDelete, path, nil, nil, nil,
			http.StatusNoContent)

		server := httptest.NewServer(getHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		a := accountDeleteCommand{accountCommand: &accountCommand{Partner: testPartnerName}}

		Reset(server.Close)

		Convey("Given a correct name", func() {
			args := []string{testAccountName}
			err := a.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect name", func() {
			args := []string{"unknown"}
			err := a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given a no name", func() {
			args := []string{}
			err := a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, "missing account name")
			})
		})

		Convey("Given an incorrect partner name", func() {
			a.Partner = incorrectPartnerName
			args := []string{testAccountName}
			err := a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestAccountUpdate(t *testing.T) {
	testAccountName := "test_account_before"
	testAccountAfter := &model.Account{
		Username: "test_account_after",
		Password: []byte("test_account_after_password"),
	}
	path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName +
		admin.AccountsURI + "/" + testAccountName
	createHandler := testHandler(http.MethodPatch, path, testAccountAfter, nil, nil,
		http.StatusCreated)

	Convey("Testing the account update function", t, func() {
		server := httptest.NewServer(createHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		a := accountUpdateCommand{accountCommand: &accountCommand{Partner: testPartnerName}}

		Reset(server.Close)

		Convey("Given correct flags", func() {
			args := []string{"-n", testAccountAfter.Username,
				"-p", string(testAccountAfter.Password),
				testAccountName,
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given incorrect flags", func() {
			args := []string{"-n", "incorrect",
				"-p", "incorrect",
				testAccountName,
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusBadRequest))
			})
		})

		Convey("Given no username", func() {
			args := []string{"-n", testAccountAfter.Username,
				"-p", string(testAccountAfter.Password),
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, "missing account name")
			})
		})

		Convey("Given an incorrect partner name", func() {
			a.Partner = incorrectPartnerName
			args := []string{"-n", testAccountAfter.Username,
				"-p", string(testAccountAfter.Password),
				testAccountName,
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestDisplayAccount(t *testing.T) {

	Convey("Given an account", t, func() {
		testAccount := &model.Account{
			Username: "test_account",
			Password: []byte("test_account_password"),
		}

		Convey("When calling the 'displayAccount' function", func() {
			out, err := ioutil.TempFile(".", "waarp_gateway")
			So(err, ShouldBeNil)
			err = displayAccount(out, testAccount)
			So(err, ShouldBeNil)
			err = out.Close()
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove(out.Name())
			})

			Convey("Then it should display the account correctly", func() {
				in, err := os.Open(out.Name())
				So(err, ShouldBeNil)
				result, err := ioutil.ReadAll(in)
				So(err, ShouldBeNil)

				expected := "Account '" + testAccount.Username + "'\n"

				So(string(result), ShouldEqual, expected)
			})
		})
	})
}
