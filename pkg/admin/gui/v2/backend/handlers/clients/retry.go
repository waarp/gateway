package clients

import (
	"encoding/json"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// How long the automatic retries of a client take (decision of 2026-10-20):
// a sentence and a table rendered by the server from the model's schedule,
// drawn as a timeline and recomputed at each keystroke by clients.js.
const (
	secondsPerDay = 24 * secondsPerHour
	daysPerYear   = 365
)

// retryRow is a line of the table of the retries: the number of the
// attempt, the wait before it and the moment since the first failure.
type retryRow struct{ N, Wait, At string }

// retrySentence says in one sentence how long the retries take: none, one,
// the same wait every time, or growing waits; an empty delay says that the
// gateway waits one second.
func retrySentence(translate gui.Translator, attempts int8, delay int32, factor float32) string {
	waits := model.RetryWaits(attempts, delay, factor)
	if len(waits) == 0 {
		return translate("clients.retries.sentence_none")
	}

	var total int64
	for _, w := range waits {
		total += int64(w)
	}

	var out string

	switch {
	case len(waits) == 1:
		out = translate("clients.retries.sentence_one", "delay", formatSpan(translate, int64(waits[0])))
	case factor == 0 || factor == 1:
		out = translate("clients.retries.sentence_constant", "n", len(waits),
			"delay", formatSpan(translate, int64(waits[0])), "total", formatSpan(translate, total))
	default:
		out = translate("clients.retries.sentence_growing", "n", len(waits), "total", formatSpan(translate, total),
			"first", formatSpan(translate, int64(waits[0])), "last", formatSpan(translate, int64(waits[len(waits)-1])))
	}

	if delay == 0 {
		out += " " + translate("clients.retries.sentence_empty_delay")
	}

	return out
}

// retryTable lists the retries: their number, the wait before each one and
// the moment since the first failure.
func retryTable(translate gui.Translator, attempts int8, delay int32, factor float32) []retryRow {
	waits := model.RetryWaits(attempts, delay, factor)
	rows := make([]retryRow, 0, len(waits))

	var at int64

	for i, w := range waits {
		at += int64(w)
		rows = append(rows, retryRow{
			N: strconv.Itoa(i + 1), Wait: formatSpan(translate, int64(w)), At: momentOf(translate, at),
		})
	}

	return rows
}

// momentOf writes a moment since the first failure ("+1 h 30 min"); beyond
// a year the order of magnitude stands alone.
func momentOf(translate gui.Translator, seconds int64) string {
	if seconds >= daysPerYear*secondsPerDay {
		return formatSpan(translate, seconds)
	}

	return "+" + formatSpan(translate, seconds)
}

// formatSpan writes a span of seconds as days, hours, minutes and seconds,
// the zero parts left out ("1 j 2 h 30 min", "45 s"); beyond a year, the
// order of magnitude only ("plus de 68 ans"): such a setting makes no sense
// and the figure says so.
func formatSpan(translate gui.Translator, seconds int64) string {
	if seconds <= 0 {
		return translate("clients.retries.delay_seconds", "n", 0)
	}

	days := seconds / secondsPerDay
	if days >= daysPerYear {
		return translate("clients.retries.more_than_years", "n", days/daysPerYear)
	}

	var parts []string
	if days > 0 {
		parts = append(parts, translate("clients.retries.delay_days", "n", days))
	}

	hours, minutes, secs := splitDelay(int32(seconds % secondsPerDay))
	if hours > 0 {
		parts = append(parts, translate("clients.retries.delay_hours", "n", hours))
	}

	if minutes > 0 {
		parts = append(parts, translate("clients.retries.delay_minutes", "n", minutes))
	}

	if secs > 0 {
		parts = append(parts, translate("clients.retries.delay_seconds", "n", secs))
	}

	return strings.Join(parts, " ")
}

// retryTexts is the JSON of the texts clients.js needs to write the sentence,
// the timeline and the table live, placeholders included ({n}, {delay},
// {total}, {first}, {last}).
func retryTexts(translate gui.Translator) string {
	keys := map[string]string{
		"none": "sentence_none", "one": "sentence_one", "constant": "sentence_constant", "growing": "sentence_growing",
		"empty": "sentence_empty_delay", "days": "delay_days", "hours": "delay_hours", "minutes": "delay_minutes",
		"seconds": "delay_seconds", "years": "more_than_years", "failure": "failure",
	}

	texts := make(map[string]string, len(keys))
	for name, key := range keys {
		texts[name] = translate("clients.retries." + key)
	}

	out, err := json.Marshal(texts)
	if err != nil {
		return "{}"
	}

	return string(out)
}
