package wg

import (
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInitParser(t *testing.T) {
	Convey("Given a flags parser", t, func() {
		parser := flags.NewNamedParser("parser_test", flags.Default)

		Convey("When calling init parser", func() {
			initParser := func() { InitParser(parser) }

			Convey("Then it should not panic", func() {
				So(initParser, ShouldNotPanic)
			})
		})
	})
}
