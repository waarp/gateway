// Package utils contains utility function for a number of common tasks needed
// throughout the project. As such, care should be taken to ensure that this
// package does NOT import any other packages from the project (to avoid
// import cycles).
package utils

import (
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

// FormatInt formats an integer as a string. It is a shortcut for strconv.FormatInt
// but without the need to cast the integer into an int64, and without the need
// to specify the base.
func FormatInt[T constraints.Signed](i T) string {
	return strconv.FormatInt(int64(i), 10)
}

// FormatUint formats an unsigned integer as a string. It is a shortcut for
// strconv.FormatUint but without the need to cast the integer into a uint64,
// and without the need to specify the base.
func FormatUint[T constraints.Unsigned](i T) string {
	return strconv.FormatUint(uint64(i), 10)
}

// FormatFloat formats a float as a string. It is a shortcut for strconv.FormatFloat
// but without the need to cast the float into a float64, and without the need
// to specify the format, precision and bit size.
func FormatFloat[T constraints.Float](f T) string {
	return strconv.FormatFloat(float64(f), 'f', -1, 64)
}

// ParseInt parses an integer from a string. It is a shortcut for strconv.ParseInt
// but without the need to specify the base and without the need to cast the result
// into the desired type.
func ParseInt[T constraints.Signed](s string) (T, error) {
	i, err := strconv.ParseInt(s, 10, reflect.TypeFor[T]().Bits())

	return T(i), err
}

// ParseUint parses an integer from a string. It is a shortcut for strconv.ParseUint
// but without the need to specify the base and without the need to cast the result
// into the desired type.
func ParseUint[T constraints.Unsigned](s string) (T, error) {
	i, err := strconv.ParseUint(s, 10, reflect.TypeFor[T]().Bits())

	return T(i), err
}

// TrimSplit splits the string by the given separator and trims the resulting
// slice of strings of leading and trailing whitespaces.
func TrimSplit(str, sep string) []string {
	s := strings.Split(str, sep)

	for i := range s {
		s[i] = strings.TrimSpace(s[i])
	}

	return s
}
