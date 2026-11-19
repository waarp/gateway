package utils

import "slices"

// Merge merges two slices of comparable elements, removing duplicates.
func Merge[T comparable](s1, s2 []T) []T {
	for _, elem := range s2 {
		if !slices.Contains(s1, elem) {
			s1 = append(s1, elem)
		}
	}

	return s1
}
