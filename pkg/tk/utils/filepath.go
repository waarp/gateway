package utils

import (
	"path"
)

// GetPath return the path given in the first non empty root provided
func GetPath(tail string, root ...string) string {
	if path.IsAbs(tail) {
		return tail
	}
	return path.Join(firstNonEmpty(root...), tail)
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}

	return ""
}
