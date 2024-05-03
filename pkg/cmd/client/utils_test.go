package wg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mattn/go-colorable"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = time.Now().Local() // init the time.Local variable
}

func resetVars() {
	Server = ""
	Partner = ""
	LocalAccount = ""
	RemoteAccount = ""
}

func writeFile(tb testing.TB, name, content string) string {
	tb.Helper()

	file := filepath.Join(tb.TempDir(), name)
	require.NoError(tb, os.WriteFile(file, []byte(content), 0o600))

	return file
}

func executeCommand(tb testing.TB, w *strings.Builder, command commander, args ...string) error {
	tb.Helper()

	_, err := flags.ParseArgs(command, args)
	require.NoError(tb, err, "failed to parse the command arguments")

	return command.execute(colorable.NewNonColorable(w)) //nolint:wrapcheck //no need to wrap here
}
