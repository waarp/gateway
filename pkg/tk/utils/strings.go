package utils

import "sort"

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
