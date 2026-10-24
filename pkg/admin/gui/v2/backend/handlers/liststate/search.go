package liststate

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// The wildcards of a search: any text, one character, and the anchors at
// the beginning and at the end of the name.
const (
	wildAny   = '*'
	wildOne   = '?'
	wildBegin = '^'
	wildEnd   = '$'
)

// Search is the search of a list, read once from its text: the text is
// looked for anywhere in a name, case and accents aside; "*" stands for any
// text, "?" for one character, "^" at the beginning and "$" at the end
// anchor the search there. The empty search matches every name.
type Search struct {
	body       string // the text without its anchors
	begin, end bool   // anchored at the beginning, at the end
	re         *regexp.Regexp
}

// NewSearch reads the text of a search.
func NewSearch(text string) Search {
	text = strings.TrimSpace(text)
	if text == "" {
		return Search{}
	}

	search := Search{body: text}
	if search.begin = strings.HasPrefix(search.body, string(wildBegin)); search.begin {
		search.body = search.body[1:]
	}

	if search.end = strings.HasSuffix(search.body, string(wildEnd)); search.end {
		search.body = search.body[:len(search.body)-1]
	}

	var expr strings.Builder
	if search.begin {
		expr.WriteString("^")
	}

	for _, r := range fold(search.body) {
		switch r {
		case wildAny:
			expr.WriteString(".*")
		case wildOne:
			expr.WriteString(".")
		default:
			expr.WriteString(regexp.QuoteMeta(string(r)))
		}
	}

	if search.end {
		expr.WriteString("$")
	}

	search.re = regexp.MustCompile(expr.String())

	return search
}

// Empty says whether the search has no text.
func (s Search) Empty() bool { return s.re == nil }

// Matches says whether the name matches the search.
func (s Search) Matches(name string) bool {
	return s.re == nil || s.re.MatchString(fold(name))
}

// Like is the search as a LIKE pattern for the database: "%" for any text,
// "_" for one character, the pattern bounded by "%" where the search is not
// anchored; the "%", "_" and escape characters of the text are escaped with
// the given one. The case and the accents are the database's: the caller
// lowers both sides when it wants the case aside.
func (s Search) Like(escape string) string {
	if s.re == nil {
		return "%"
	}

	var pattern strings.Builder
	if !s.begin {
		pattern.WriteString("%")
	}

	for _, r := range s.body {
		switch r {
		case wildAny:
			pattern.WriteString("%")
		case wildOne:
			pattern.WriteString("_")
		case '%', '_':
			pattern.WriteString(escape)
			pattern.WriteRune(r)
		default:
			if string(r) == escape {
				pattern.WriteString(escape)
			}

			pattern.WriteRune(r)
		}
	}

	if !s.end {
		pattern.WriteString("%")
	}

	return pattern.String()
}

// fold is the text without its accents, in lower case: the form the names
// and the searches are compared in.
func fold(text string) string {
	unaccented, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), text)
	if err != nil {
		unaccented = text
	}

	return strings.ToLower(unaccented)
}
