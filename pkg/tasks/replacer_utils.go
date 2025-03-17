package tasks

import (
	"strings"
	"time"
)

const defaultTSFormat = "YYYY-MM-DD_HHmmss"

func formatTime(tsFormat string, t time.Time) string {
	tsTokens := []string{
		`YYYY`, `YY`, `MMMM`, `MMM`, `MM`, `M`, `DD`, `D`, `dddd`, `ddd`,
		`HH`, `hh`, `h`, `PM`, `pm`, `mm`, `m`, `ss`, `s`, `tz`, `zz`, `z`,
	}
	goTokens := []string{
		`2006`, `06`, `January`, `Jan`, `01`, `1`, `02`, `2`, `Monday`, `Mon`,
		`15`, `03`, `3`, `PM`, `pm`, `04`, `4`, `05`, `5`, `MST`, `-07:00`, `-0700`,
	}

	if tsFormat != "" {
		tsFormat = strings.TrimPrefix(tsFormat, "(")
		tsFormat = strings.TrimSuffix(tsFormat, ")")
	}

	if tsFormat == "" {
		tsFormat = defaultTSFormat
	}

	goFormat := tsFormat
	for i := range tsTokens {
		goFormat = strings.ReplaceAll(goFormat, tsTokens[i], goTokens[i])
	}

	return t.Format(goFormat)
}
