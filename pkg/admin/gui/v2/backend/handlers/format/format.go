// Package format writes numbers and sizes the way the language reads them:
// digits grouped by thousands, sizes in bytes and in their readable unit.
// One source for every screen (the transfer monitoring, the sheets of the
// protocol options), with the keys of the "format" section of the catalogue.
package format

import (
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"
)

const (
	kibi        = 1024
	tenUnits    = 10 // one decimal under ten
	singularMax = 2  // 0 and 1 are singular in French
)

// units are the catalogue keys of the sizes above the byte, in order.
//
//nolint:gochecknoglobals // reference data
var units = []string{"kb", "mb", "gb", "tb"}

// GroupDigits writes n with its digits grouped by thousands, the separator
// of the language between the groups.
func GroupDigits(translate gui.Translator, n uint64) string {
	digits := strconv.FormatUint(n, 10)
	sep := translate("format.thousands")

	var out strings.Builder

	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out.WriteString(sep)
		}

		out.WriteRune(c)
	}

	return out.String()
}

// ExactBytes is a size to the byte, its digits grouped ("1 228 800 octets").
func ExactBytes(translate gui.Translator, n int64) string {
	if n < singularMax {
		return translate("format.size.byte", "n", n)
	}

	return translate("format.size.bytes", "n", GroupDigits(translate, uint64(n)))
}

// Bytes writes a size in bytes, then in the largest unit that keeps it above
// one, with one decimal under ten ("1,2 Mo").
func Bytes(translate gui.Translator, n int64) string {
	switch {
	case n < singularMax:
		return translate("format.size.byte", "n", n)
	case n < kibi:
		return translate("format.size.bytes", "n", n)
	}

	value := float64(n)
	unit := ""

	for _, u := range units {
		value /= kibi
		unit = u

		if value < kibi {
			break
		}
	}

	text := strconv.FormatFloat(value, 'f', 0, 64)
	if value < tenUnits {
		text = strings.Replace(strconv.FormatFloat(value, 'f', 1, 64), ".", translate("format.decimal"), 1)
	}

	return translate("format.size."+unit, "n", text)
}

// WholeBytes writes a size for a sheet: the exact size, preceded by its
// readable unit when the size is a whole number of that unit
// ("10 Mo (10 485 760 octets)", "32 700 octets").
func WholeBytes(translate gui.Translator, n int64) string {
	exact := ExactBytes(translate, n)
	if n < kibi {
		return exact
	}

	value, unit := n, ""

	for _, u := range units {
		if value%kibi != 0 {
			break
		}

		value /= kibi
		unit = u
	}

	if unit == "" {
		return exact
	}

	return translate("format.size."+unit, "n", GroupDigits(translate, uint64(value))) + " (" + exact + ")"
}
