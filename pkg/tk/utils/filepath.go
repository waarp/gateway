package utils

import "path/filepath"

// SlashJoin joins all the given elements into a path using the slash '/'
// character as separator.
func SlashJoin(elem ...string) string {
	return filepath.ToSlash(filepath.Join(elem...))
}
