package testhelpers

import (
	"code.waarp.fr/lib/log"
	"github.com/smartystreets/goconvey/convey"
)

func TestLogger(c convey.C, name string) *log.Logger {
	back, err := log.NewBackend(log.LevelDebug, log.Stdout, "", "")
	c.So(err, convey.ShouldBeNil)

	return back.NewLogger(name)
}

func TestLoggerWithLevel(c convey.C, name string, level log.Level) *log.Logger {
	back, err := log.NewBackend(level, log.Stdout, "", "")
	c.So(err, convey.ShouldBeNil)

	return back.NewLogger(name)
}
