package varlist

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The variables the list offers are those the tasks implement (pkg/tasks),
// each with a label in French and in English.
func TestVariablesFollowTheTasks(t *testing.T) {
	t.Parallel()

	// Implemented but not offered: a duplicate of #ERRORMSG# and a deprecated marker.
	notOffered := []string{"ERRORSTRCODE", "FULLTRANSFERID"}

	var grouped []string
	for _, g := range varGroups {
		grouped = append(grouped, g.names...)
	}

	implemented := append(tasks.Markers(), tasks.FormatMarkers()...)
	for _, name := range implemented {
		assert.Truef(t, slices.Contains(grouped, name) || slices.Contains(notOffered, name),
			"the marker %q the tasks implement must be in a group, or set aside", name)
	}

	for _, name := range append(slices.Clone(grouped), notOffered...) {
		assert.Truef(t, name == transferInfoName || slices.Contains(implemented, name),
			"%q is not a marker the tasks implement", name)
	}

	catalog := common.NewShell().Catalog()
	now := time.Date(2026, 10, 21, 9, 12, 5, 0, time.Local)

	for _, lang := range []string{"fr", "en"} {
		translate := catalog.Translator(lang)
		list := List(translate, now)

		for _, g := range list.Groups {
			assert.NotContains(t, g.Label, "¿")

			for _, v := range g.Vars {
				assert.NotContainsf(t, v.Label, "¿", "%s: the label of %s", lang, v.Name)

				for _, tok := range v.Tokens {
					assert.NotContainsf(t, tok.Label, "¿", "%s: the label of the element %s", lang, tok.Code)
					assert.NotEmpty(t, tok.Example)
				}
			}
		}
	}

	for _, code := range tasks.TimestampTokens() {
		assert.Containsf(t, tokenKeys, code, "the element %q of a date format needs a label", code)
	}

	assert.NotEqual(t, "AM", strings.ToUpper(tasks.FormatTimestamp("(pm)", now)),
		"pm is set aside while the tasks render it wrong: offer it again once they do")
}

// The reserved keys of the transfer information (pkg/model) are offered, each
// with a label in French and in English, by family (#816).
func TestReservedKeysAreOffered(t *testing.T) {
	t.Parallel()

	catalog := common.NewShell().Catalog()
	keys := model.ReservedInfoKeys()

	for _, lang := range []string{"fr", "en"} {
		list := List(catalog.Translator(lang), time.Now())
		offered := map[string]string{}

		for _, g := range list.Groups {
			assert.NotContains(t, g.Label, "¿")

			for _, v := range g.Vars {
				offered[v.Name] = v.Label
			}
		}

		for _, k := range keys {
			label, ok := offered[transferInfoName+k.Name]
			assert.Truef(t, ok, "%s: %s is not offered", lang, k.Name)
			assert.NotEmptyf(t, label, "%s: the label of %s", lang, k.Name)
			assert.NotContainsf(t, label, "¿", "%s: the label of %s", lang, k.Name)
		}

		assert.Len(t, list.Groups, len(varGroups)+2, "a group for the PeSIT keys, one for the acknowledgements")
	}
}

func TestInfoKeyLabel(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]string{
		"__followID__":               "vars.info.follow_id",
		"__clientTransFreetextRaw__": "vars.info.client_trans_freetext_raw",
		"__ackReceivedOn__":          "vars.info.ack_received_on",
		"__syncTransferID__":         "vars.info.sync_transfer_id",
		"__bankID__":                 "vars.info.bank_id",
	} {
		assert.Equal(t, want, InfoKeyLabel(name))
	}
}
