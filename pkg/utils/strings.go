package utils

import (
	"sort"
	"strconv"

	"golang.org/x/exp/constraints"
)

// ContainsStrings returns whether the given slice contains one of the given
// strings or not.
func ContainsStrings(slice []string, strings ...string) bool {
	n := len(slice)
	foundIndex := sort.Search(n, func(i int) bool {
		for _, s := range strings {
			if slice[i] == s {
				return true
			}
		}

		return false
	})

	return foundIndex != n
}

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
