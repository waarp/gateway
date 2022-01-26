package main

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func addrInfoString(target, redirect string) string {
	return "● Address " + target + " redirects to " + redirect + "\n"
}

func TestGetAddressOverride(t *testing.T) {
	Convey("Testing the address override 'get' command", t, func() {
		out = testFile()
		command := &overrideAddressGet{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil, nil))
			conf.InitTestOverrides(c)
			So(conf.AddIndirection("localhost", "127.0.0.1"), ShouldBeNil)
			So(conf.AddIndirection("waarp.fr", "1.2.3.4"), ShouldBeNil)

			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{"localhost"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display the indirection", func() {
						So(getOutput(), ShouldEqual, addrInfoString("localhost", "127.0.0.1"))
					})
				})
			})
		})
	})
}

func TestSetAddressOverride(t *testing.T) {
	Convey("Testing the address override 'set' command", t, func() {
		out = testFile()
		command := &overrideAddressSet{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil, nil))
			conf.InitTestOverrides(c)

			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{"-t", "localhost", "-r", "127.0.0.1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the user was added", func() {
						So(getOutput(), ShouldEqual, "The indirection for address "+
							command.Target+" was successfully set.\n")
					})

					Convey("Then the new indirection should have been added", func() {
						So(conf.GetIndirection("localhost"), ShouldEqual, "127.0.0.1")
					})
				})
			})
		})
	})
}

func TestListAddressOverrides(t *testing.T) {
	Convey("Testing the address override 'list' command", t, func() {
		out = testFile()
		command := &overrideAddressList{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil, nil))
			conf.InitTestOverrides(c)
			So(conf.AddIndirection("localhost", "127.0.0.1"), ShouldBeNil)
			So(conf.AddIndirection("waarp.fr", "1.2.3.4"), ShouldBeNil)

			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display the indirection", func() {
						So(getOutput(), ShouldContainSubstring, addrInfoString("localhost", "127.0.0.1"))
						So(getOutput(), ShouldContainSubstring, addrInfoString("waarp.fr", "1.2.3.4"))
					})
				})
			})
		})
	})
}

func TestDeleteAddressOverride(t *testing.T) {
	Convey("Testing the address override 'delete' command", t, func() {
		out = testFile()
		command := &overrideAddressDelete{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil, nil))
			conf.InitTestOverrides(c)
			So(conf.AddIndirection("localhost", "127.0.0.1"), ShouldBeNil)
			So(conf.AddIndirection("waarp.fr", "1.2.3.4"), ShouldBeNil)

			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{"localhost"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the user was deleted", func() {
						So(getOutput(), ShouldEqual, "The indirection for address "+
							command.Args.Target+" was successfully deleted.\n")
					})

					Convey("Then the new indirection should have been deleted", func() {
						So(conf.GetIndirection("localhost"), ShouldBeBlank)
						So(conf.GetIndirection("waarp.fr"), ShouldEqual, "1.2.3.4")
					})
				})
			})
		})
	})
}
