package liststate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchMatches(t *testing.T) {
	t.Parallel()

	names := []string{"srv-sftp", "SRV-R66", "Banque-prod", "relevé-été", "a?b"}
	cases := []struct {
		text string
		want []string
	}{
		{"", names},
		{"  ", names},
		{"srv", []string{"srv-sftp", "SRV-R66"}},
		{"SFTP", []string{"srv-sftp"}},
		{"prod", []string{"Banque-prod"}},
		{"releve-ete", []string{"relevé-été"}},
		{"RELEVÉ", []string{"relevé-été"}},
		{"srv*66", []string{"SRV-R66"}},
		{"s*p", []string{"srv-sftp"}},
		{"*", names},
		{"srv-r?6", []string{"SRV-R66"}},
		{"a?b", []string{"a?b"}},
		{"^srv", []string{"srv-sftp", "SRV-R66"}},
		{"^sftp", nil},
		{"sftp$", []string{"srv-sftp"}},
		{"^srv-sftp$", []string{"srv-sftp"}},
		{"^srv-sft$", nil},
		{"^", names},
		{"$", names},
		{"^$", nil},
		{"a.b", nil},
		{"(prod)", nil},
	}

	for _, c := range cases {
		s := NewSearch(c.text)

		var got []string

		for _, name := range names {
			if s.Matches(name) {
				got = append(got, name)
			}
		}

		assert.Equal(t, c.want, got, "search %q", c.text)
		assert.Equal(t, strings.TrimSpace(c.text) == "", s.Empty(), "emptiness of %q", c.text)
	}
}

func TestSearchLike(t *testing.T) {
	t.Parallel()

	const esc = "\033"

	cases := map[string]string{
		"":              "%",
		"releve":        "%releve%",
		"releve*pdf":    "%releve%pdf%",
		"releve-?.pdf":  "%releve-_.pdf%",
		"^in/":          "in/%",
		".pdf$":         "%.pdf",
		"^in/paie.xml$": "in/paie.xml",
		"100%":          "%100" + esc + "%%",
		"a_b":           "%a" + esc + "_b%",
		"a" + esc + "b": "%a" + esc + esc + "b%",
	}

	for text, want := range cases {
		assert.Equal(t, want, NewSearch(text).Like(esc), "search %q", text)
	}
}
