package utils

import "iter"

func Count[T any](seq iter.Seq[T]) int {
	n := 0
	for range seq {
		n++
	}

	return n
}

func Count2[K, V any](seq iter.Seq2[K, V]) int {
	n := 0
	for range seq {
		n++
	}

	return n
}
