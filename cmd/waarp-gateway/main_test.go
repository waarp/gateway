package main

import (
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"

	wg "code.waarp.fr/apps/gateway/gateway/pkg/cmd/client"
)

func TestCommandParsing(t *testing.T) {
	Convey("Given a flags parser", t, func() {
		parser := flags.NewNamedParser("parser_test", flags.Default)

		Convey("When initializing the parser", func() {
			cmd := &commands{}
			initParser := func() { wg.InitParser(parser, cmd) }

			Convey("Then it should not panic", func() {
				So(initParser, ShouldNotPanic)

				Convey("When calling main", func() {
					args := []string{
						"-a", "admin:admin_password@localhost:8080",
						"account", "local", "test", "-h",
					}
					mainFunc := func() { _ = wg.Main(parser, args) }

					Convey("Then it should not panic", func() {
						So(mainFunc, ShouldNotPanic)
					})
				})
			})
		})
	})
}
