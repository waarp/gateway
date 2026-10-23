package clients

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// spanTranslator writes a key with its arguments, so that a test reads the
// shape of a sentence without the catalogue.
func spanTranslator(key string, args ...any) string {
	key = strings.TrimPrefix(key, "clients.retries.")

	parts := []string{key}
	for i := 0; i+1 < len(args); i += 2 {
		parts = append(parts, fmt.Sprintf("%v=%v", args[i], args[i+1]))
	}

	return strings.Join(parts, " ")
}

func TestFormatSpan(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "delay_seconds n=0", formatSpan(spanTranslator, 0))
	assert.Equal(t, "delay_seconds n=45", formatSpan(spanTranslator, 45))
	assert.Equal(t, "delay_hours n=1 delay_minutes n=1 delay_seconds n=30", formatSpan(spanTranslator, 3690))
	assert.Equal(t, "delay_days n=1 delay_hours n=2", formatSpan(spanTranslator, 86400+7200), "days beyond a day")
	assert.Equal(t, "more_than_years n=68", formatSpan(spanTranslator, 2147483647), "beyond a year, the order of magnitude")
	assert.Equal(t, "+delay_seconds n=30", momentOf(spanTranslator, 30))
	assert.Equal(t, "more_than_years n=1", momentOf(spanTranslator, 366*86400), "no plus sign on an order of magnitude")
}

func TestRetrySentence(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "sentence_none", retrySentence(spanTranslator, 0, 30, 2))
	assert.Equal(t, "sentence_one delay=delay_seconds n=30", retrySentence(spanTranslator, 1, 30, 2))
	assert.Equal(t, "sentence_constant n=3 delay=delay_seconds n=30 total=delay_minutes n=1 delay_seconds n=30",
		retrySentence(spanTranslator, 3, 30, 1), "a constant delay")
	assert.Equal(t, "sentence_constant n=3 delay=delay_seconds n=30 total=delay_minutes n=1 delay_seconds n=30",
		retrySentence(spanTranslator, 3, 30, 0), "no factor stored reads as a constant delay")
	assert.Equal(t, "sentence_growing n=5 total=delay_minutes n=15 delay_seconds n=30 first=delay_seconds n=30 last=delay_minutes n=8",
		retrySentence(spanTranslator, 5, 30, 2), "growing waits")
	assert.Equal(t, "sentence_constant n=3 delay=delay_seconds n=1 total=delay_seconds n=3 sentence_empty_delay",
		retrySentence(spanTranslator, 3, 0, 1), "an empty delay waits one second and says so")

	rows := retryTable(spanTranslator, 3, 30, 2)
	assert.Equal(t, []retryRow{
		{Wait: "delay_seconds n=30", At: "+delay_seconds n=30"},
		{Wait: "delay_minutes n=1", At: "+delay_minutes n=1 delay_seconds n=30"},
		{Wait: "delay_minutes n=2", At: "+delay_minutes n=3 delay_seconds n=30"},
	}, rows)
	assert.Empty(t, retryTable(spanTranslator, 0, 30, 2))

	texts := retryTexts(spanTranslator)
	assert.Contains(t, texts, `"growing":"sentence_growing"`)
	assert.Contains(t, texts, `"failure":"failure"`)
}
