package utils

import (
	"errors"
	"fmt"
	"sort"

	"golang.org/x/exp/constraints"
)

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

func OrderedIterate[V any](m map[string]V, f func(key string, val V)) {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		f(key, m[key])
	}
}

var (
	// ErrKeyNotFound is the error returned by GetAs if the requested
	// key does not exist in the given map.
	ErrKeyNotFound = errors.New("key not found")
	// ErrIncorrectValueType is the error returned by GetAs if the value
	// associated with the requested key is not of the expected type.
	ErrIncorrectValueType = errors.New("incorrect value type")
)

// GetAs checks whether the given key exists in the given map, and whether
// the value associated with it is of the same type as the function's type parameter.
// If the key does exist, and the value is of the correct type, then the value
// is returned with that type. Otherwise, an error is returned.
func GetAs[T any](m map[string]any, key string) (t T, _ error) {
	if asAny, hasProperty := m[key]; hasProperty {
		if asT, isT := asAny.(T); isT {
			return asT, nil
		}

		return t, fmt.Errorf("key %q: %w: expected %T, got %T", key,
			ErrIncorrectValueType, t, asAny)
	}

	return t, fmt.Errorf("key %q: %w", key, ErrKeyNotFound)
}
