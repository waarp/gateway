package utils

import "path/filepath"

// SlashJoin joins all the given elements into a path using the slash '/'
// character as separator.
func SlashJoin(elem ...string) string {
	return filepath.ToSlash(filepath.Join(elem...))
}

// CleanSlash replaces all separators of the path with slashes '/', then cleans the
// resulting path.
func CleanSlash(path string) string {
	return filepath.Clean(filepath.ToSlash(path))
}

// GetPath return the path given in the first non empty root provided
func GetPath(path string, root ...string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return SlashJoin(firstNonEmpty(root...), path)
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}

	return ""
}
