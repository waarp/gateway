package common

import (
	"errors"
	"fmt"
	"html/template"
	"path"
	"strings"
	"time"

	"github.com/karrick/tparse/v2"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func ParseTemplate(file string, others ...string) *template.Template {
	return NamedParseTemplate(path.Base(file), file, others...)
}

func NamedParseTemplate(name, file string, others ...string) *template.Template {
	return template.Must(
		template.New(name).
			Funcs(templateFuncs()).
			ParseFS(webfs.WebFS, append([]string{file}, others...)...),
	)
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"print":                 fmt.Print,
		"add":                   add,
		"mkDur":                 mkDur,
		"dict":                  dict,
		"list":                  list,
		"timePart":              getDurationPart,
		"archiveBase":           archiveBase,
		"archiveExt":            archiveExt,
		"date":                  date,
		"time":                  hour,
		"tz":                    tz,
		"listCharsets":          tasks.TranscodeFormats.Keys,
		"listArchiveExtensions": tasks.ArchiveExtensions.Keys,
		"listExtractExtensions": tasks.ExtractExtensions.Keys,
		"encryptMethods":        tasks.EncryptMethods.Keys,
		"decryptMethods":        tasks.DecryptMethods.Keys,
		"signMethods":           tasks.SignMethods.Keys,
		"verifyMethods":         tasks.VerifyMethods.Keys,
		"encryptSignMethods":    tasks.EncryptSignMethods.Keys,
		"decryptVerifyMethods":  tasks.DecryptVerifyMethods.Keys,
	}
}

func add(a, b int, c ...int) int {
	sum := a + b
	for _, v := range c {
		sum += v
	}

	return sum
}

func mkDur(val any, unit string) any {
	if val == nil {
		return nil
	}

	return fmt.Sprint(val, unit)
}

func getDurationPart(value any, toUnit string) (string, error) {
	if value == nil {
		return "", nil
	}

	dur, pErr := tparse.AbsoluteDuration(time.Now(), fmt.Sprint(value))
	if pErr != nil {
		return "", fmt.Errorf("failed to parse time value: %w", pErr)
	}

	ns := dur.Nanoseconds()

	for _, unit := range []struct {
		name string
		dur  int64
	}{
		{"d", int64(time.Hour * 24)}, //nolint:mnd //too specific
		{"h", int64(time.Hour)},
		{"m", int64(time.Minute)},
		{"s", int64(time.Second)},
		{"ms", int64(time.Millisecond)},
		{"us", int64(time.Microsecond)},
		{"ns", int64(time.Nanosecond)},
	} {
		stepValue := ns / unit.dur
		if unit.name == toUnit {
			return utils.FormatInt(stepValue), nil
		}

		ns -= stepValue * unit.dur
	}

	//nolint:err113 //this is a base error
	return "", fmt.Errorf("unknown targer unit %q", toUnit)
}

func list(s ...string) []string { return s }

func archiveExt(p any) string {
	str, ok := p.(string)
	if !ok {
		return ""
	}

	for _, ext := range tasks.ArchiveExtensions.Keys() {
		if strings.HasSuffix(str, ext) {
			return ext
		}
	}

	return ""
}

func archiveBase(p any) string {
	str, ok := p.(string)
	if !ok {
		return ""
	}

	return strings.TrimSuffix(str, archiveExt(str))
}

//nolint:err113 //these are base errors
func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, errors.New(`invalid number of argument for "dict"`)
	}

	m := map[string]any{}
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("keys must be strings (got %T)", values[i])
		}
		m[key] = values[i+1]
	}

	return m, nil
}

func date(value any) (string, error) {
	if value == nil {
		return "", nil
	}

	t, err := time.Parse(time.RFC3339, fmt.Sprint(value))
	if err != nil {
		return "", fmt.Errorf("failed to parse time value: %w", err)
	}

	return t.Format(time.DateOnly), nil
}

func hour(value any) (string, error) {
	if value == nil {
		return "", nil
	}

	t, err := time.Parse(time.RFC3339, fmt.Sprint(value))
	if err != nil {
		return "", fmt.Errorf("failed to parse time value: %w", err)
	}

	return t.Format(time.DateOnly), nil
}

func tz() string { return time.Now().Format("Z07:00") }
