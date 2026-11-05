package rules

import (
	"net/http"
	"net/url"
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
			`data-case="ascii-ebcdic ascii-ascii" data-of="transcode-direction"`, `name="transcode.fromCharset_ascii"`,
			`<optgroup label="Unicode">`, `<option value="UTF-8">UTF-8 — Unicode sur 8 bits</option>`,
			`data-case="ebcdic-ascii ebcdic-ebcdic" data-of="transcode-direction"`, `name="transcode.fromCharset_ebcdic"`,
			`<optgroup label="Alphabet latin">`, `<option value="IBM Code Page 297">IBM Code Page 297 — France</option>`,
			`<optgroup label="Autres alphabets">`, `name="transcode.toCharset_ascii"`, `name="transcode.toCharset_ebcdic"`)
		assert.NotContains(t, body, `<option value="X-User-Defined">`, "a deprecated character set is not offered")
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
			assert.Contains(t, body, `<option value="`+stored["fromCharset"]+`" selected>`, "the source selected")
			assert.Contains(t, body, `<option value="`+stored["toCharset"]+`" selected>`, "the target selected")

			action, values := postedForm(t, body)
			require.Equal(t, http.StatusSeeOther, s.post(t, permWrite, action, values).Code)
			assert.Equal(t, stored, map[string]string(s.chain(t, rule, model.ChainPre)[0].Args), "saved as it was")
		}
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
