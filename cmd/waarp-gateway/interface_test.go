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

func TestInterfaceGet(t *testing.T) {

	Convey("Testing the interface get function", t, func() {
		testInterface := model.Interface{
			Name: "test_interface_get",
			Type: "sftp",
			Port: 1,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		err := testDb.Create(&testInterface)
		So(err, ShouldBeNil)
		id := strconv.FormatUint(testInterface.ID, 10)

		Reset(func() {
			err := testDb.Delete(&testInterface)
			So(err, ShouldBeNil)
		})

		Convey("Given a correct id", func() {
			p := interfaceGetCommand{}
			args := []string{id}
			err := p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect id", func() {
			p := interfaceGetCommand{}
			args := []string{"unknown"}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given no id", func() {
			p := interfaceGetCommand{}
			args := []string{}
			err := p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})

			Convey("Then the error should say that the id is missing", func() {
				So(err.Error(), ShouldEqual, "missing interface ID")
			})
		})
	})
}

func TestInterfaceCreate(t *testing.T) {

	Convey("Testing the interface creation function", t, func() {
		testInterface := model.Interface{
			Name: "test_interface_create",
			Type: "sftp",
			Port: 1,
		}
		existingInterface := model.Interface{
			Name: "test_interface_existing",
			Type: "sftp",
			Port: 2,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		p := interfaceCreateCommand{}
		args := []string{"-n", testInterface.Name,
			"-t", testInterface.Type,
			"-p", strconv.FormatUint(uint64(testInterface.Port), 10),
		}

		err := testDb.Create(&existingInterface)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(&existingInterface)
			So(err, ShouldBeNil)
		})

		Convey("Given correct values", func() {
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Reset(func() {
				_ = testDb.Delete(&testInterface)
			})

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the new interface should have been inserted", func() {
				ok, err := testDb.Exists(&testInterface)

				So(err, ShouldBeNil)
				So(ok, ShouldBeTrue)
			})
		})

		Convey("Given an already existing name", func() {
			args := []string{"-n", existingInterface.Name,
				"-t", existingInterface.Type,
				"-p", strconv.FormatUint(uint64(existingInterface.Port), 10),
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})

			Convey("Then the error should say the name already exist", func() {
				So(err.Error(), ShouldEqual, "An interface with the same ID "+
					"or name already exist")
			})
		})

		Convey("Given an invalid port", func() {
			args := []string{"-n", testInterface.Name,
				"-t", testInterface.Type,
				"-p", "not_a_port",
			}
			_, err := flags.ParseArgs(&p, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given an invalid type", func() {
			args := []string{"-n", testInterface.Name,
				"-t", "not_a_type",
				"-p", strconv.FormatUint(uint64(testInterface.Port), 10),
			}
			_, err := flags.ParseArgs(&p, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given an incorrect address", func() {
			auth = ConnectionOptions{
				Address:  "http://incorrect",
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

func TestInterfaceSelect(t *testing.T) {

	Convey("Testing the interface listing function", t, func() {
		testInterface1 := model.Interface{
			Name: "test_interface_select1",
			Type: "sftp",
			Port: 2,
		}
		testInterface2 := model.Interface{
			Name: "test_interface_select2",
			Type: "sftp",
			Port: 4,
		}
		testInterface3 := model.Interface{
			Name: "test_interface_select3",
			Type: "sftp",
		}
		testInterface4 := model.Interface{
			Name: "test_interface_select4",
			Type: "sftp",
			Port: 3,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		p := interfaceListCommand{}

		err := testDb.Create(&testInterface1)
		So(err, ShouldBeNil)
		err = testDb.Create(&testInterface2)
		So(err, ShouldBeNil)
		err = testDb.Create(&testInterface3)
		So(err, ShouldBeNil)
		err = testDb.Create(&testInterface4)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(&testInterface1)
			So(err, ShouldBeNil)
			err = testDb.Delete(&testInterface2)
			So(err, ShouldBeNil)
			err = testDb.Delete(&testInterface3)
			So(err, ShouldBeNil)
			err = testDb.Delete(&testInterface4)
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
			args := []string{"-s", "type"}
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

		Convey("Given a type flag", func() {
			args := []string{"-t", testInterface1.Type, "-t", testInterface3.Type}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a port flag", func() {
			args := []string{"-p", strconv.FormatUint(uint64(testInterface2.Port), 10),
				"-p", strconv.FormatUint(uint64(testInterface4.Port), 10)}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestInterfaceDelete(t *testing.T) {

	Convey("Testing the interface deletion function", t, func() {
		testInterface := model.Interface{
			Name: "test_interface_delete",
			Type: "sftp",
			Port: 1,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		p := interfaceDeleteCommand{}

		err := testDb.Create(&testInterface)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(&testInterface)
			So(err, ShouldBeNil)
		})

		Convey("Given a correct id", func() {
			args := []string{strconv.FormatUint(testInterface.ID, 10)}
			err := p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the interface should no longer exist", func() {
				ok, err := testDb.Exists(&testInterface)

				So(err, ShouldBeNil)
				So(ok, ShouldBeFalse)
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

func TestInterfaceUpdate(t *testing.T) {

	Convey("Testing the interface update function", t, func() {
		testInterfaceBefore := model.Interface{
			Name: "test_interface_update_before",
			Type: "sftp",
			Port: 1,
		}
		testInterfaceAfter := model.Interface{
			Name: "test_interface_update_after",
			Type: "sftp",
			Port: 1,
		}
		newPort := strconv.FormatUint(uint64(testInterfaceAfter.Port), 10)

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		p := interfaceUpdateCommand{}

		err := testDb.Create(&testInterfaceBefore)
		So(err, ShouldBeNil)
		id := strconv.FormatUint(testInterfaceBefore.ID, 10)

		Reset(func() {
			err := testDb.Delete(&testInterfaceBefore)
			So(err, ShouldBeNil)
			err = testDb.Delete(&testInterfaceAfter)
			So(err, ShouldBeNil)
		})

		Convey("Given correct values", func() {
			args := []string{"-n", testInterfaceAfter.Name,
				"-t", testInterfaceAfter.Type,
				"-p", newPort,
				id,
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the interface should have been updated", func() {
				ok, err := testDb.Exists(&testInterfaceAfter)

				So(err, ShouldBeNil)
				So(ok, ShouldBeTrue)
			})
		})

		Convey("Given an incorrect port", func() {
			args := []string{"-n", testInterfaceAfter.Name,
				"-t", testInterfaceAfter.Type,
				"-p", "not_a_port",
				id,
			}
			_, err := flags.ParseArgs(&p, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given no id", func() {
			args := []string{"-n", testInterfaceAfter.Name,
				"-t", testInterfaceAfter.Type,
				"-p", newPort,
			}
			args, err := flags.ParseArgs(&p, args)
			So(err, ShouldBeNil)
			err = p.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})

			Convey("Then the error should say the id is missing", func() {
				So(err.Error(), ShouldEqual, "missing interface ID")
			})
		})

		Convey("Given an invalid type", func() {
			args := []string{"-n", testInterfaceAfter.Name,
				"-t", "not_a_type",
				"-p", newPort,
				id,
			}
			args, err := flags.ParseArgs(&p, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestDisplayInterface(t *testing.T) {

	Convey("Given a interface", t, func() {
		testInterface := model.Interface{
			ID:   1234,
			Name: "test_interface",
			Type: "sftp",
			Port: 1,
		}
		id := strconv.FormatUint(testInterface.ID, 10)

		Convey("When calling the 'displayInterface' function", func() {
			out, err := ioutil.TempFile(".", "waarp_gateway")
			So(err, ShouldBeNil)

			displayInterface(out, testInterface)

			err = out.Close()
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove(out.Name())
			})

			Convey("Then it should display the interface correctly", func() {
				in, err := os.Open(out.Name())
				So(err, ShouldBeNil)
				result, err := ioutil.ReadAll(in)
				So(err, ShouldBeNil)

				expected :=
					"Interface n°" + id + ":\n" +
						"├─Name: " + testInterface.Name + "\n" +
						"├─Type: " + testInterface.Type + "\n" +
						"└─Port: " + strconv.FormatUint(uint64(testInterface.Port), 10) + "\n"

				So(string(result), ShouldEqual, expected)
			})
		})
	})
}
