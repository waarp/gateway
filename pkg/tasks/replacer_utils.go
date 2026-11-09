package tasks

import (
	"strings"
	"time"
)

const (
	defaultTSFormat = "YYYY-MM-DD_HHmmss"
	isoTSFormat     = "YYYY-MM-DDTHH:mm:sszz"
)

// timestampTokens are the elements of the date format of a timestamp marker
// and their Go layout, replaced in this order.
//
//nolint:gochecknoglobals // reference data of the substitutions
var timestampTokens = [][2]string{
	{`YYYY`, `2006`},
	{`YY`, `06`},
	{`MMMM`, `January`},
	{`MMM`, `Jan`},
	{`MM`, `01`},
	{`M`, `1`},
	{`DD`, `02`},
	{`D`, `2`},
	{`dddd`, `Monday`},
	{`ddd`, `Mon`},
	{`HH`, `15`},
	{`hh`, `03`},
	{`h`, `3`},
	{`PM`, `PM`},
	{`pm`, `pm`},
	{`mm`, `04`},
	{`m`, `4`},
	{`ss`, `05`},
	{`s`, `5`},
	{`tz`, `MST`},
	{`zz`, `Z07:00`},
	{`z`, `Z0700`},
}

func formatTime(tsFormat string, t time.Time) string {
	if tsFormat != "" {
		tsFormat = strings.TrimPrefix(tsFormat, "(")
		tsFormat = strings.TrimSuffix(tsFormat, ")")
	}

	if tsFormat == "" {
		tsFormat = defaultTSFormat
	}

	goFormat := tsFormat
	for _, token := range timestampTokens {
		goFormat = strings.ReplaceAll(goFormat, token[0], token[1])
	}

	return t.Local().Format(goFormat)
}
