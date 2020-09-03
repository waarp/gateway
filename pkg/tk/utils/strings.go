package utils

import "sort"

// ContainsStrings returns whether the given slice contains one of the given
// strings or not.
func ContainsStrings(slice []string, strings ...string) bool {
	n := len(slice)
	return sort.Search(n, func(i int) bool {
		for _, s := range strings {
			if slice[i] == s {
				return true
			}
		}
		return false
	}) != n
}

// StringPtr takes a string and returns a pointer to that string. Useful for
// putting a string literal where a string pointer is required.
func StringPtr(s string) *string {
	return &s
}

// String takes a string pointer and returns the string it is pointing to. If
// the pointer is nil, returns an empty string. Useful to avoid panics bacause
// of nil pointer dereference.
func String(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
