package wg

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = time.Now().Local() // init the time.Local variable
}

func writeFile(tb testing.TB, name, content string) string {
	tb.Helper()

	file := filepath.Join(tb.TempDir(), name)
	require.NoError(tb, os.WriteFile(file, []byte(content), 0o600))

	return file
}

type commander interface{ execute(w io.Writer) error }

func executeCommand(tb testing.TB, w *strings.Builder, command commander, args ...string) error {
	tb.Helper()

	_, err := flags.ParseArgs(command, args)
	require.NoError(tb, err, "failed to parse the command arguments")

	return command.execute(w) //nolint:wrapcheck //no need to wrap here
}

func newTestOutput() *strings.Builder {
	return &strings.Builder{}
}

func parseAsLocalTime(tb testing.TB, str string) time.Time {
	tb.Helper()

	date, err := time.Parse(time.RFC3339Nano, str)
	require.NoErrorf(tb, err, "failed to parse date %q", str)

	return date.Local()
}
