package protoform

import (
	"math"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"
)

// Number describes a number field as the browser checks it before the server:
// a whole number, or a decimal one with Decimal; its bounds (Max 0:
// none) and the step of its arrows (0: 1 for a whole number, any value for a
// decimal). The server reads the value with the same bounds (Read); the step
// only helps the typing.
type Number struct {
	Decimal  bool
	Min, Max float64
	Step     float64
}

// InputMode is the keyboard a phone opens on the field.
func (n *Number) InputMode() string {
	if n.Decimal {
		return "decimal"
	}

	return "numeric"
}

// StepAttr is the step attribute of the input.
func (n *Number) StepAttr() string {
	switch {
	case n.Step > 0:
		return NumberText(n.Step)
	case n.Decimal:
		return "any"
	default:
		return "1"
	}
}

// MinAttr is the min attribute of the input.
func (n *Number) MinAttr() string { return NumberText(n.Min) }

// MaxAttr is the max attribute of the input, "" without a maximum.
func (n *Number) MaxAttr() string {
	if n.Max == 0 {
		return ""
	}

	return NumberText(n.Max)
}

// Read reads a typed number within the bounds, a decimal comma accepted; a
// refusal answers the message of what the field expects, its bounds written as
// the language reads them.
func (n *Number) Read(translate gui.Translator, raw string) (value float64, msg string) {
	raw = strings.Replace(strings.TrimSpace(raw), ",", ".", 1)

	var err error
	if n.Decimal {
		value, err = strconv.ParseFloat(raw, 64)
	} else {
		var whole int
		whole, err = strconv.Atoi(raw)
		value = float64(whole)
	}

	switch {
	case err != nil, math.IsInf(value, 0), math.IsNaN(value), value < n.Min, n.Max != 0 && value > n.Max:
		return 0, translate(n.messageKey(), "min", n.word(translate, n.Min), "max", n.word(translate, n.Max))
	}

	return value, ""
}

// messageKey is the catalogue key of the refusal: the field has a lower bound
// at least (the input carries it), an upper one when Max is set.
func (n *Number) messageKey() string {
	switch {
	case n.Decimal && n.Max != 0:
		return "proto.error_decimal_range"
	case n.Decimal:
		return "proto.error_decimal_min"
	case n.Max != 0:
		return "proto.error_number_range"
	default:
		return "proto.error_number_min"
	}
}

// word writes a bound as the language reads it ("0,5" in French).
func (n *Number) word(translate gui.Translator, bound float64) string {
	return strings.Replace(NumberText(bound), ".", translate("format.decimal"), 1)
}

// NumberText writes a number as the product and the browser read it: "7",
// "1.5", without a trailing zero.
func NumberText(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
