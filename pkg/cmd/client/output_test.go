package wg

import (
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/require"
)

func expectedOutput(tb testing.TB, data any, text ...string) string {
	tb.Helper()

	var builder strings.Builder

	functions := template.FuncMap{
		"join":       join,
		"flatten":    mapFlatten,
		"getServer":  func() string { return Server },
		"getPartner": func() string { return Partner },
	}

	fulltext := strings.Join(text, "\n") + "\n"
	temp, err := template.New(tb.Name()).Funcs(functions).Parse(fulltext)
	require.NoError(tb, err)
	require.NoError(tb, temp.Execute(&builder, data))

	return builder.String()
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

func enabledStatus(enabled bool) string {
	return ifElse(enabled, TextEnabled, TextDisabled)
}
