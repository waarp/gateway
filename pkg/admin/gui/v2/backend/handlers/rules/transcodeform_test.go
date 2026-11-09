package rules

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

func TestCharsetKey(t *testing.T) {
	t.Parallel()

	for name, key := range map[string]string{
		"IBM Code Page 297": "ibm_code_page_297",
		"UTF-16BE BOM":      "utf_16be_bom",
		"ISO-8859-6E":       "iso_8859_6e",
		"HZ-GB2312":         "hz_gb2312",
		"X-User-Defined":    "x_user_defined",
	} {
		assert.Equal(t, key, charsetKey(name), name)
	}
}

// Every character set of the task is on one side, in a group; the EBCDIC
// pages are split by their letters.
func TestCharsetSides(t *testing.T) {
	t.Parallel()

	sides := charsetSides()
	groupOf := map[string]string{}

	for side, items := range sides {
		for _, item := range items {
			_, seen := groupOf[item.name]
			assert.False(t, seen, "%s is on one side only", item.name)
			assert.Contains(t, charsetGroupOrder(), item.group, item.name)

			groupOf[item.name] = side + "/" + item.group
		}
	}

	assert.Len(t, groupOf, len(tasks.TranscodeRegistry().List()), "every character set is offered")

	for name, group := range map[string]string{
		"UTF-8":              "ascii/unicode",
		"ISO 8859-15":        "ascii/iso8859",
		"Windows 1252":       "ascii/windows",
		"IBM Code Page 850":  "ascii/pc",
		"Shift JIS":          "ascii/asian",
		"IBM Code Page 297":  "ebcdic/ebcdic_latin",
		"IBM Code Page 1047": "ebcdic/ebcdic_latin",
		"IBM Code Page 290":  "ebcdic/ebcdic_other",
		"IBM Code Page 875":  "ebcdic/ebcdic_other",
		"IBM Code Page 1025": "ebcdic/ebcdic_other",
	} {
		assert.Equal(t, group, groupOf[name], name)
	}
}

func TestTranscodeForm(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given a new task", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type="+tasks.Transcode).Body.String()
		indexes(t, body, "<h2 class=\"w-card__title\">Encodage</h2>",
			`name="transcode.direction" value="ascii-ebcdic" required data-switch="transcode-direction"`,
			`name="transcode.direction" value="ebcdic-ebcdic" required data-switch="transcode-direction"`,
			`data-case="ascii-ebcdic ascii-ascii" data-of="transcode-direction"`,
			`name="transcode.fromCharset_ascii" required data-combobox=`, `<optgroup label="Unicode">`,
			`<option value="UTF-8" data-label="UTF-8 — Unicode 8 bits">UTF-8 — Unicode 8 bits</option>`,
			`data-case="ebcdic-ascii ebcdic-ebcdic" data-of="transcode-direction"`,
			`name="transcode.fromCharset_ebcdic" required data-combobox=`, `<optgroup label="Alphabet latin">`,
			`<option value="IBM Code Page 297" data-label="IBM Code Page 297 — France">IBM Code Page 297 — France</option>`,
			`<optgroup label="Autres alphabets">`, `name="transcode.toCharset_ascii"`, `name="transcode.toCharset_ebcdic"`)
		assert.NotContains(t, body, `<option value="X-User-Defined"`, "a deprecated character set is not offered")
		assert.NotContains(t, body, `name="transcode.records"`, "TRANSCODE converts the whole file")
	})

	t.Run("Given a stored task", func(t *testing.T) {
		t.Parallel()

		for _, stored := range []map[string]string{
			{"fromCharset": "UTF-8", "toCharset": "IBM Code Page 297"},
			{"fromCharset": "IBM Code Page 1147", "toCharset": "IBM Code Page 037"},
			{"fromCharset": "X-User-Defined", "toCharset": "Windows 1252"},
		} {
			rule := &model.Rule{Name: "encodage-" + charsetKey(stored["fromCharset"]), IsSend: true, Path: "encodage-" + charsetKey(stored["fromCharset"])}
			require.NoError(t, s.db.Insert(rule).Run())
			require.NoError(t, s.db.Insert(&model.Task{RuleID: rule.ID, Chain: model.ChainPre, Type: tasks.Transcode, Args: stored}).Run())

			body := s.get(t, permWrite, path(rule, "/tasks/pre/0/edit")).Body.String()
			direction := charsetSide(stored["fromCharset"]) + "-" + charsetSide(stored["toCharset"])
			assert.Contains(t, body, `name="transcode.direction" value="`+direction+`" checked`, "the direction of the character sets")
			assert.Contains(t, body, `<option value="`+stored["fromCharset"]+`" selected data-label=`, "the source selected")
			assert.Contains(t, body, `<option value="`+stored["toCharset"]+`" selected data-label=`, "the target selected")

			action, values := postedForm(t, body)
			require.Equal(t, http.StatusSeeOther, s.post(t, permWrite, action, values).Code)
			assert.Equal(t, stored, map[string]string(s.chain(t, rule, model.ChainPre)[0].Args), "saved as it was")
		}
	})

	t.Run("Given the cards of the records", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type="+url.QueryEscape(tasks.TranscodeAndReformat)).Body.String()
		indexes(t, body, `name="transcode_reformat.substitute"`, "<h2 class=\"w-card__title\">Fichier source</h2>",
			`id="transcode_reformat-records" name="transcode_reformat.records"`, `data-switch="transcode_reformat-records"`,
			`<option value="">Fichier entier</option>`, `<option value="lines">Lignes</option>`,
			`data-case="fixed" data-of="transcode_reformat-records"`, `name="transcode_reformat.recordLength"`, `name="transcode_reformat.trimPadding"`,
			`data-case="lines fixed rdw" data-of="transcode_reformat-records"`, "<h2 class=\"w-card__title\">Fichier converti</h2>",
			`<option value="">Comme le fichier source</option>`, `<option value="none">Sans séparateur</option>`,
			`data-case="lines" data-of="transcode_reformat-outputRecords"`, `name="transcode_reformat.outputSeparator" value=""`,
			`name="transcode_reformat.outputSeparator" value="NEL"`, "NL EBCDIC",
			`data-case="fixed" data-of="transcode_reformat-outputRecords"`, `name="transcode_reformat.outputLength"`,
			`data-case="fixed rdw" data-of="transcode_reformat-outputRecords"`, `name="transcode_reformat.truncate"`)
	})

	t.Run("Given a task in lines", func(t *testing.T) {
		t.Parallel()

		rule := &model.Rule{Name: "lignes", IsSend: true, Path: "lignes"}
		require.NoError(t, s.db.Insert(rule).Run())

		rec := s.post(t, permWrite, path(rule, "/tasks/pre/new"), url.Values{
			"type": {tasks.TranscodeAndReformat}, "transcode_reformat.direction": {"ebcdic-ascii"},
			"transcode_reformat.fromCharset_ebcdic": {"IBM Code Page 297"}, "transcode_reformat.toCharset_ascii": {"ISO 8859-1"},
			"transcode_reformat.records": {"lines"}, "transcode_reformat.outputRecords": {"lines"}, "transcode_reformat.outputSeparator": {"CRLF"},
			"transcode_reformat.recordLength": {"80"}, "transcode_reformat.outputLength": {"80"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, map[string]string{
			"fromCharset": "IBM Code Page 297", "toCharset": "ISO 8859-1",
			"records": "lines", "outputRecords": "lines", "outputSeparator": "CRLF",
		}, map[string]string(s.chain(t, rule, model.ChainPre)[0].Args), "the lengths that do not apply are left out")
	})

	t.Run("Given a task back to the whole file", func(t *testing.T) {
		t.Parallel()

		rule := &model.Rule{Name: "entier", IsSend: true, Path: "entier"}
		require.NoError(t, s.db.Insert(rule).Run())
		require.NoError(t, s.db.Insert(&model.Task{RuleID: rule.ID, Chain: model.ChainPre, Type: tasks.TranscodeAndReformat, Args: map[string]string{
			"fromCharset": "IBM Code Page 297", "toCharset": "ISO 8859-1", "records": "fixed", "recordLength": "80",
			"trimPadding": "true", "outputRecords": "lines", "outputSeparator": "CRLF",
		}}).Run())

		// Without JavaScript, every card is sent: the server reads none of those
		// that the choice of the whole file hides.
		action, values := postedForm(t, s.get(t, permWrite, path(rule, "/tasks/pre/0/edit")).Body.String())
		values.Set("transcode_reformat.records", "")
		require.Equal(t, http.StatusSeeOther, s.post(t, permWrite, action, values).Code)
		assert.Equal(t, map[string]string{"fromCharset": "IBM Code Page 297", "toCharset": "ISO 8859-1"},
			map[string]string(s.chain(t, rule, model.ChainPre)[0].Args))
	})

	t.Run("Given a new REFORMAT task", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type="+tasks.Reformat).Body.String()
		indexes(t, body, "<h2 class=\"w-card__title\">Fichier source</h2>",
			`name="reformat.charset" required data-combobox=`, `<optgroup label="Unicode">`,
			`<option value="UTF-8" data-label="UTF-8 — Unicode 8 bits">`, `<optgroup label="EBCDIC — Alphabet latin">`,
			`<option value="IBM Code Page 297" data-label="IBM Code Page 297 — France">`,
			`<optgroup label="EBCDIC — Autres alphabets">`,
			`name="reformat.records" value="lines"`, `name="reformat.records" value="rdw"`,
			"<h2 class=\"w-card__title\">Fichier produit</h2>", `name="reformat.outputRecords"`,
			`<option value="none">Sans séparateur</option>`)
		assert.NotContains(t, body, `name="reformat.records" value=""`, "the records are required")
		_, output, _ := strings.Cut(body, `id="reformat-outputRecords"`)
		output, _, _ = strings.Cut(output, "</select>")
		assert.NotContains(t, output, "Comme le fichier source", "the result is laid out")
		assert.Contains(t, output, `<option value="">Choisissez</option>`)
	})

	t.Run("Given a REFORMAT task saved", func(t *testing.T) {
		t.Parallel()

		rule := &model.Rule{Name: "mise-en-forme", IsSend: false, Path: "mise-en-forme"}
		require.NoError(t, s.db.Insert(rule).Run())

		rec := s.post(t, permWrite, path(rule, "/tasks/post/new"), url.Values{
			"type": {tasks.Reformat}, "reformat.charset": {"IBM Code Page 297"},
			"reformat.records": {"rdw"}, "reformat.outputRecords": {"lines"}, "reformat.outputSeparator": {""},
			"reformat.recordLength": {"80"}, "reformat.outputLength": {"80"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, map[string]string{"charset": "IBM Code Page 297", "records": "rdw", "outputRecords": "lines"},
			map[string]string(s.chain(t, rule, model.ChainPost)[0].Args), "the lengths that do not apply are left out")

		// A character set stored by an alias is shown as the one it names.
		stored := &model.Rule{Name: "alias", IsSend: true, Path: "alias"}
		require.NoError(t, s.db.Insert(stored).Run())
		require.NoError(t, s.db.Insert(&model.Task{RuleID: stored.ID, Chain: model.ChainPre, Type: tasks.Reformat, Args: map[string]string{
			"charset": "IBM1147", "records": "lines", "outputRecords": "none",
		}}).Run())

		body := s.get(t, permWrite, path(stored, "/tasks/pre/0/edit")).Body.String()
		assert.Contains(t, body, `<option value="IBM Code Page 1147" selected data-label=`)
	})

	t.Run("Given a task without its direction", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), url.Values{
			"type": {tasks.Transcode}, "transcode.fromCharset_ascii": {"UTF-8"}, "transcode.toCharset_ebcdic": {"IBM Code Page 297"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Choisissez un sens.")
		assert.Empty(t, s.chain(t, s.orders, model.ChainPre), "nothing saved")
	})
}
