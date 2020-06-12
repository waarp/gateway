package utils

import (
	"path"
	"path/filepath"
)

// Deprecated: SlashJoin joins all the given elements into a path using the slash '/'
// character as separator.
func SlashJoin(elem ...string) string {
	return filepath.ToSlash(filepath.Join(elem...))
}

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
