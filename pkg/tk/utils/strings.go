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

func FormatInt[T constraints.Integer](i T) string {
	return strconv.FormatInt(int64(i), 10)
}

func FormatUint[T constraints.Unsigned](i T) string {
	return strconv.FormatUint(uint64(i), 10)
}

func FormatFloat[T constraints.Float](f T) string {
	return strconv.FormatFloat(float64(f), 'f', -1, 64)
}
