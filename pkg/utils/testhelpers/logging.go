package testhelpers

import (
	"fmt"
	"testing"

	"code.waarp.fr/lib/log"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"
)

func TestLogger(c convey.C, name string, args ...any) *log.Logger {
	back, err := log.NewBackend(log.LevelDebug, log.Stdout, "", "")
	c.So(err, convey.ShouldBeNil)

	return back.NewLogger(fmt.Sprintf(name, args...))
}

func TestLoggerWithLevel(c convey.C, name string, level log.Level) *log.Logger {
	back, err := log.NewBackend(level, log.Stdout, "", "")
	c.So(err, convey.ShouldBeNil)

	return back.NewLogger(name)
}

func GetTestLogger(tb testing.TB) *log.Logger {
	tb.Helper()

	back, err := log.NewBackend(log.LevelDebug, log.Stdout, "", "")
	require.NoError(tb, err)

	return back.NewLogger(tb.Name())
}
