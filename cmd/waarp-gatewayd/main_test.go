package main

import (
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"

	wgd "code.waarp.fr/apps/gateway/gateway/pkg/cmd/server"
)

func TestInitParser(t *testing.T) {
	Convey("Given a flags parser", t, func() {
		parser := flags.NewNamedParser("parser_test", flags.Default)

		Convey("When calling init parser", func() {
			initParser := func() { wgd.InitParser(parser, &commands{}) }

			Convey("Then it should not panic", func() {
				So(initParser, ShouldNotPanic)
			})
		})
	})
}

func TestCommandParsing(t *testing.T) {
	Convey("Given a flags parser", t, func() {
		parser := flags.NewNamedParser("parser_test", flags.Default)

		Convey("When initializing the parser", func() {
			cmd := &commands{}
			initParser := func() { _ = wgd.InitParser(parser, cmd) }

			Convey("Then it should not panic", func() {
				So(initParser, ShouldNotPanic)

				Convey("When calling main", func() {
					args := []string{"server", "start", "-c", "gatewayd.ini"}
					mainFunc := func() { _ = wgd.Main(parser, args) }

					Convey("Then it should not panic", func() {
						So(mainFunc, ShouldNotPanic)
					})
				})
			})
		})
	})
}
