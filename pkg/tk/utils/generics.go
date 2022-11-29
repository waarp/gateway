package utils

// If simulates a ternary operator.
func If[T any](cond bool, valTrue, valFalse T) T {
	if cond {
		return valTrue
	} else {
		return valFalse
	}
}
