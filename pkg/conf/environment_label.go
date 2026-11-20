package conf

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxEnvironmentLabelLength is the number of characters an environment label
// may hold.
const MaxEnvironmentLabelLength = 32

// EnvironmentLabel is the short text that tells an instance from the others
// ("Production", "Acceptance", "Test"...), shown next to the name of the
// instance in the header of the web interface and given by the REST API
// (AdminConfig.EnvironmentLabel). It is free, on one line.
type EnvironmentLabel string

var errInvalidEnvironmentLabel = errors.New("invalid environment label")

// UnmarshalFlag reads a label from the configuration file, the spaces around
// it left out. A label that is too long, or that holds a character that
// cannot be printed, is refused: the gateway does not start with it.
func (l *EnvironmentLabel) UnmarshalFlag(value string) error {
	label := strings.TrimSpace(value)

	if length := utf8.RuneCountInString(label); length > MaxEnvironmentLabelLength {
		return fmt.Errorf("%w: %d characters, %d at most",
			errInvalidEnvironmentLabel, length, MaxEnvironmentLabelLength)
	}

	if strings.ContainsFunc(label, func(r rune) bool { return !unicode.IsPrint(r) }) {
		return fmt.Errorf("%w: it holds a character that cannot be printed", errInvalidEnvironmentLabel)
	}

	*l = EnvironmentLabel(label)

	return nil
}
