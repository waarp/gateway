package features

import (
	"iter"
	"slices"
)

type Feature byte

const (
	Listing Feature = iota
	Deletion
	RecursiveDeletion

	endOfFeatures // always keep last
)

//nolint:gochecknoglobals //global var is needed here
var list = map[string][]Feature{}

func Register(protocol string, features ...Feature) {
	list[protocol] = append(list[protocol], features...)
}

func AllFeatures() iter.Seq[Feature] {
	return func(yield func(Feature) bool) {
		for f := range endOfFeatures {
			if !yield(f) {
				return
			}
		}
	}
}

func Supports(protocol string, feature Feature) bool {
	features, ok := list[protocol]
	if !ok {
		return false
	}

	return slices.Contains(features, feature)
}
