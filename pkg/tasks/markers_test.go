package tasks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestMarkers(t *testing.T) {
	t.Parallel()

	fixed := Markers()
	assert.Contains(t, fixed, "TRUEFILENAME")
	assert.Contains(t, fixed, "ERRORMSG")
	assert.Contains(t, fixed, "FULLTRANSFERID", "a deprecated marker is still implemented")
	assert.IsIncreasing(t, fixed)

	for _, word := range unimplementedMarkers {
		assert.NotContains(t, fixed, word[1:len(word)-1], "a marker the tasks do not implement is not listed")
	}

	assert.Equal(t, []string{"STARTTIMESTAMP", "TIMESTAMP"}, FormatMarkers())
	assert.NotContains(t, fixed, "TIMESTAMP", "a marker with a format is listed apart")
}

func TestUnknownMarkers(t *testing.T) {
	t.Parallel()

	text := "#TRUEFILENAME# #FOO# #DATE# #ARCHPATH# #TIMESTAMP(DD/MM/YYYY à HH:mm)# #TIMESTAMP# #TI_clé# #TI_lot# " +
		"#date# #FOO# #DATE(x)# &#233; color:#FFF; #1#"

	assert.Equal(t, []string{"#FOO#", "#ARCHPATH#", "#date#", "#DATE(x)#"}, UnknownMarkers(text))
	assert.Empty(t, UnknownMarkers("Transfer #TRANSFERID# done: #ERRORMSG#"))
}

// A marker the tasks do not implement still makes them fail.
func TestUnimplementedMarkersFail(t *testing.T) {
	t.Parallel()

	ctx := &model.TransferContext{Transfer: &model.Transfer{}, Rule: &model.Rule{}}

	for _, word := range unimplementedMarkers {
		_, err := replaceVars("x "+word, ctx)
		assert.ErrorIs(t, err, errNotImplemented, word)
	}
}

func TestTimestampTokens(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 10, 21, 9, 12, 5, 0, time.Local)

	assert.Equal(t, "YYYY", TimestampTokens()[0])
	assert.Len(t, TimestampTokens(), len(timestampTokens))
	assert.Equal(t, "21/10/2026 09:12", FormatTimestamp("(DD/MM/YYYY HH:mm)", date))
	assert.Equal(t, "2026-10-21_091205", FormatTimestamp("", date))
}

func TestRenderEmail(t *testing.T) {
	t.Parallel()

	template := &model.EmailTemplate{
		Name:        "fin-de-transfert",
		Subject:     "Transfer #TRANSFERID# of #TRUEFILENAME#",
		Body:        "<p>Rule #RULE#, started #STARTTIMESTAMP(DD/MM/YYYY)#, #UNKNOWN#</p>",
		Attachments: []string{"#TRUEFULLPATH#", "/fixed/report.pdf"},
	}
	ctx := &model.TransferContext{
		Transfer: &model.Transfer{
			ID: 1234, LocalPath: "/data/in/invoices.csv", Start: time.Date(2026, 10, 21, 9, 12, 5, 0, time.Local),
		},
		Rule:  &model.Rule{Name: "paie-mensuelle"},
		Paths: &conf.PathsConfig{},
	}

	msg, err := RenderEmail(template, ctx)
	require.NoError(t, err)

	assert.Equal(t, "Transfer 1234 of invoices.csv", msg.Subject)
	assert.Equal(t, "<p>Rule paie-mensuelle, started 21/10/2026, #UNKNOWN#</p>", msg.Body,
		"an unknown marker is left as it is, as the task sends it")
	assert.Equal(t, []string{"/data/in/invoices.csv", "/fixed/report.pdf"}, msg.Attachments)
	assert.Equal(t, "#TRUEFULLPATH#", template.Attachments[0], "the template is left unchanged")

	_, err = RenderEmail(&model.EmailTemplate{Subject: "#ARCHPATH#"}, ctx)
	assert.ErrorIs(t, err, errNotImplemented)
}
