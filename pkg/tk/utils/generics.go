package utils

import "golang.org/x/exp/constraints"

// If simulates a ternary operator.
func If[T any](cond bool, valTrue, valFalse T) T {
	if cond {
		return valTrue
	} else {
		return valFalse
	}
}

func Max[T constraints.Ordered](args ...T) T {
	if len(args) == 0 {
		var zeroVal T

		return zeroVal
	}

	max := args[0]

	for _, candidate := range args {
		if candidate > max {
			max = candidate
		}
	}

	return max
}

func Min[T constraints.Ordered](args ...T) T {
	if len(args) == 0 {
		var zeroVal T

		return zeroVal
	}

	min := args[0]

	for _, candidate := range args {
		if candidate < min {
			min = candidate
		}
	}

	return min
}
