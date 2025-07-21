package utils

import (
	"errors"
	"fmt"
	"slices"
)

// If simulates a ternary operator.
func If[T any](cond bool, valTrue, valFalse T) T {
	if cond {
		return valTrue
	}

	return valFalse
}

// ContainsOneOf returns whether the given slice contains at least one of the
// given elements or not.
func ContainsOneOf[T comparable](slice []T, elems ...T) bool {
	for _, elem := range elems {
		if slices.Contains(slice, elem) {
			return true
		}
	}

	return false
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

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}

	return val
}
