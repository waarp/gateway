package testhelpers

import (
	"os"
	"testing"

	"code.waarp.fr/lib/log"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
)

func TestLogger(c convey.C, name string) *log.Logger {
	level := log.LevelDebug

	if envLvl := os.Getenv("WAARP_TEST_LOG_LEVEL"); envLvl != "" {
		level = logging.LevelFromString(envLvl)
	}

	back, err := log.NewBackend(level, log.Stdout, "", "")
	c.So(err, convey.ShouldBeNil)

	return back.NewLogger(name)
}

func TestLoggerWithLevel(c convey.C, name string, level log.Level) *log.Logger {
	back, err := log.NewBackend(level, log.Stdout, "", "")
	c.So(err, convey.ShouldBeNil)

	return back.NewLogger(name)
}

func GetTestLogger(tb testing.TB) *log.Logger {
	tb.Helper()

	return GetTestLoggerWithLevel(tb, log.LevelDebug)
}

func GetTestLoggerWithLevel(tb testing.TB, level log.Level) *log.Logger {
	tb.Helper()

	back, err := log.NewBackend(level, log.Stdout, "", "")
	require.NoError(tb, err)

	return back.NewLogger(tb.Name())
}
