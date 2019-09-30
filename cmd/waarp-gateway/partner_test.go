package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

var testPartnerInterface model.Interface

func init() {
	testPartnerInterface = model.Interface{
		Name: "test_partner_interface",
		Port: 1,
		Type: "sftp",
	}
	if err := testDb.Create(&testPartnerInterface); err != nil {
		panic(err)
	}
}

func TestPartnerGet(t *testing.T) {

	Convey("Testing the partner get function", t, func() {
		testPartner := model.Partner{
			Name:        "test_partner_get",
			Address:     "test_partner_get_address",
			Port:        1,
			InterfaceID: testPartnerInterface.ID,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		err := testDb.Create(&testPartner)
		So(err, ShouldBeNil)
		id := strconv.FormatUint(testPartner.ID, 10)

		Reset(func() {
			err := testDb.Delete(&testPartner)
			So(err, ShouldBeNil)
		})

		Convey("Given a correct id", func() {
			p := partnerGetCommand{}
			args := []string{id}
			err := p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect id", func() {
			p := partnerGetCommand{}
			args := []string{"unknown"}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given no id", func() {
			p := partnerGetCommand{}
			args := []string{}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestPartnerCreate(t *testing.T) {

	Convey("Testing the partner creation function", t, func() {
		testPartner := model.Partner{
			Name:        "test_partner_create",
			Address:     "test_partner_create_address",
			Port:        1,
			InterfaceID: testPartnerInterface.ID,
		}
		existingPartner := model.Partner{
			Name:        "test_partner_existing",
			Address:     "test_partner_existing_address",
			Port:        2,
			InterfaceID: testPartnerInterface.ID,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		p := partnerCreateCommand{}
		args := []string{"-n", testPartner.Name,
			"-a", testPartner.Address,
			"-p", strconv.FormatUint(uint64(testPartner.Port), 10),
			"-i", strconv.FormatUint(testPartner.InterfaceID, 10),
		}

		err := testDb.Create(&existingPartner)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(&existingPartner)
			So(err, ShouldBeNil)
		})

		Convey("Given correct values", func() {
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Reset(func() {
				_ = testDb.Delete(&testPartner)
			})

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given already existing values", func() {
			args := []string{"-n", existingPartner.Name,
				"-a", existingPartner.Address,
				"-p", strconv.FormatUint(uint64(existingPartner.Port), 10),
				"-i", strconv.FormatUint(existingPartner.InterfaceID, 10),
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given an invalid port", func() {
			args := []string{"-n", testPartner.Name,
				"-a", testPartner.Address,
				"-p", "not_a_port",
			}
			_, err := flags.ParseArgs(&p, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given an incorrect address", func() {
			auth = ConnectionOptions{
				Address:  "incorrect",
				Username: "test",
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given incorrect credentials", func() {
			auth = ConnectionOptions{
				Address:  testServer.URL,
				Username: "unknown",
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestPartnerSelect(t *testing.T) {

	Convey("Testing the partner listing function", t, func() {
		testPartner1 := model.Partner{
			Name:        "test_partner_select1",
			Address:     "test_partner_select1_address",
			Port:        2,
			InterfaceID: testPartnerInterface.ID,
		}
		testPartner2 := model.Partner{
			Name:        "test_partner_select2",
			Address:     "test_partner_select3_address",
			Port:        4,
			InterfaceID: testPartnerInterface.ID,
		}
		testPartner3 := model.Partner{
			Name:        "test_partner_select3",
			Address:     "test_partner_select4_address",
			Port:        1,
			InterfaceID: testPartnerInterface.ID,
		}
		testPartner4 := model.Partner{
			Name:        "test_partner_select4",
			Address:     "test_partner_select2_address",
			Port:        3,
			InterfaceID: testPartnerInterface.ID,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		p := partnerListCommand{}

		err := testDb.Create(&testPartner1)
		So(err, ShouldBeNil)
		err = testDb.Create(&testPartner2)
		So(err, ShouldBeNil)
		err = testDb.Create(&testPartner3)
		So(err, ShouldBeNil)
		err = testDb.Create(&testPartner4)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(&testPartner1)
			So(err, ShouldBeNil)
			err = testDb.Delete(&testPartner2)
			So(err, ShouldBeNil)
			err = testDb.Delete(&testPartner3)
			So(err, ShouldBeNil)
			err = testDb.Delete(&testPartner4)
			So(err, ShouldBeNil)
		})

		Convey("Given no flags", func() {
			args := []string{}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a limit flag", func() {
			args := []string{"-l", "2"}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			So(p.Limit, ShouldEqual, 2)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an offset flag", func() {
			args := []string{"-o", "2"}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a sort flag", func() {
			args := []string{"-s", "address"}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an order flag", func() {
			args := []string{"-d"}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an address flag", func() {
			args := []string{"-a", testPartner1.Address, "-a", testPartner3.Address}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an interface flag", func() {
			args := []string{"-i", strconv.FormatUint(testPartnerInterface.ID, 10)}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestPartnerDelete(t *testing.T) {

	Convey("Testing the partner deletion function", t, func() {
		testPartner := model.Partner{
			Name:        "test_partner_delete",
			Address:     "test_partner_delete_address",
			Port:        1,
			InterfaceID: testPartnerInterface.ID,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		p := partnerDeleteCommand{}

		err := testDb.Create(&testPartner)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(&testPartner)
			So(err, ShouldBeNil)
		})

		Convey("Given a correct id", func() {
			args := []string{strconv.FormatUint(testPartner.ID, 10)}
			err := p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect id", func() {
			args := []string{"unknown"}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a no id", func() {
			args := []string{}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestPartnerUpdate(t *testing.T) {

	Convey("Testing the partner update function", t, func() {
		testPartnerBefore := model.Partner{
			Name:        "test_partner_update_before",
			Address:     "test_partner_update_before_address",
			Port:        1,
			InterfaceID: testPartnerInterface.ID,
		}
		testPartnerAfter := model.Partner{
			Name:        "test_partner_update_after",
			Address:     "test_partner_update_after_address",
			Port:        1,
			InterfaceID: testPartnerInterface.ID,
		}
		newPort := strconv.FormatUint(uint64(testPartnerAfter.Port), 10)
		newInterface := strconv.FormatUint(testPartnerAfter.InterfaceID, 10)

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		p := partnerUpdateCommand{}

		err := testDb.Create(&testPartnerBefore)
		So(err, ShouldBeNil)
		id := strconv.FormatUint(testPartnerBefore.ID, 10)

		Reset(func() {
			err := testDb.Delete(&testPartnerBefore)
			So(err, ShouldBeNil)
			err = testDb.Delete(&testPartnerAfter)
			So(err, ShouldBeNil)
		})

		Convey("Given correct values", func() {
			args := []string{"-n", testPartnerAfter.Name,
				"-a", testPartnerAfter.Address,
				"-p", newPort,
				"-i", newInterface,
				id,
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect port", func() {
			args := []string{"-n", testPartnerAfter.Name,
				"-a", testPartnerAfter.Address,
				"-p", "not_a_port",
				"-i", newInterface,
				id,
			}
			_, err := flags.ParseArgs(&p, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given no id", func() {
			args := []string{"-n", testPartnerAfter.Name,
				"-a", testPartnerAfter.Address,
				"-p", newPort,
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a non-existent interface id", func() {
			args := []string{"-n", testPartnerAfter.Name,
				"-a", testPartnerAfter.Address,
				"-p", newPort,
				"-i", "1000",
				id,
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a non-numeric interface id", func() {
			args := []string{"-n", testPartnerAfter.Name,
				"-a", testPartnerAfter.Address,
				"-p", newPort,
				"-i", "not_an_id",
				id,
			}
			_, err := flags.ParseArgs(&p, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestDisplayPartner(t *testing.T) {

	Convey("Given a partner", t, func() {
		testPartner := model.Partner{
			ID:          1234,
			Name:        "test_partner",
			Address:     "test_partner_address",
			Port:        1,
			InterfaceID: testPartnerInterface.ID,
		}
		id := strconv.FormatUint(testPartner.ID, 10)

		Convey("When calling the 'displayPartner' function", func() {
			out, err := ioutil.TempFile(".", "waarp_gateway")
			So(err, ShouldBeNil)

			displayPartner(out, testPartner)

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
					"Partner n°" + id + ":\n" +
						"├─Name: " + testPartner.Name + "\n" +
						"├─InterfaceID: " + strconv.FormatUint(testPartner.InterfaceID, 10) + "\n" +
						"├─Address: " + testPartner.Address + "\n" +
						"└─Port: " + strconv.FormatUint(uint64(testPartner.Port), 10) + "\n"

				So(string(result), ShouldEqual, expected)
			})
		})
	})
}
