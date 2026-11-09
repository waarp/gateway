package tasks

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

// unimplementedMarkers are known to the tasks but not implemented: a task
// whose arguments, or whose e-mail template, hold one fails.
//
//nolint:gochecknoglobals // reference data of the substitutions
var unimplementedMarkers = []string{
	"#ARCHPATH#", "#REMOTEHOSTIP#", "#LOCALHOSTIP#", "#RANKTRANSFER#", "#BLOCKSIZE#", "#NOWAIT#", "#LOCALEXEC#",
}

// markerPattern finds what reads as a marker in a text: a name between two
// "#", with an optional parameter between parentheses. The markers of the
// transfer info, whose key is free, are recognised by their prefix.
var markerPattern = regexp.MustCompile(`#([A-Za-z][A-Za-z0-9_]*(?:\([^)#\n]*\))?)#`)

// transferInfoPrefix starts the name of the marker of a transfer info
// (#TI_<key>#).
const transferInfoPrefix = "TI_"

// markerNames splits the markers the tasks implement into the fixed ones and
// those that take a date format between parentheses, by name, sorted.
func markerNames() (fixed, format []string) {
	for key := range getReplacers() {
		if slices.Contains(unimplementedMarkers, key) {
			continue
		}

		name := strings.Trim(key, "#")
		if base, _, withFormat := strings.Cut(name, "("); withFormat {
			format = append(format, base)

			continue
		}

		fixed = append(fixed, name)
	}

	slices.Sort(fixed)
	slices.Sort(format)

	return fixed, format
}

// Markers are the names of the fixed substitution markers the tasks
// implement, as written between the two "#" ("TRUEFILENAME"), sorted. The
// markers that take a date format are listed by FormatMarkers; the markers
// of the transfer info (#TI_<key>#) are not listed, their key being free.
func Markers() []string {
	fixed, _ := markerNames()

	return fixed
}

// FormatMarkers are the names of the substitution markers that take a date
// format between parentheses ("TIMESTAMP" for #TIMESTAMP(format)#), sorted;
// without a format, they use YYYY-MM-DD_HHmmss.
func FormatMarkers() []string {
	_, format := markerNames()

	return format
}

// UnknownMarkers lists, in the order they appear and once each, the markers
// of a text that the tasks do not substitute: a marker they do not know is
// left as it is in what they produce, one they do not implement makes them
// fail. A marker of the transfer info is always known.
func UnknownMarkers(text string) []string {
	fixed, format := markerNames()

	var unknown []string

	for _, match := range markerPattern.FindAllStringSubmatch(text, -1) {
		name := match[1]
		base, _, _ := strings.Cut(name, "(")

		known := strings.HasPrefix(name, transferInfoPrefix) || slices.Contains(format, base) ||
			(base == name && slices.Contains(fixed, name))

		if !known && !slices.Contains(unknown, match[0]) {
			unknown = append(unknown, match[0])
		}
	}

	return unknown
}

// TimestampTokens are the elements of the date format of a timestamp marker
// ("YYYY", "MM", "DD"...), in the order the tasks read them.
func TimestampTokens() []string {
	tokens := make([]string, 0, len(timestampTokens))
	for _, t := range timestampTokens {
		tokens = append(tokens, t[0])
	}

	return tokens
}

// FormatTimestamp formats a date as a timestamp marker does with the given
// format ("DD/MM/YYYY", or "" for the default one).
func FormatTimestamp(format string, t time.Time) string { return formatTime(format, t) }
