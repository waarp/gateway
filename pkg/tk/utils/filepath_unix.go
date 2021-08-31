// +build !windows,!plan9

package utils

import "path/filepath"

// NormalizePath transforms a Unix path into a valid "file" URI according to
// RFC 8089.
// Deprecated
func NormalizePath(path string) string {
	return filepath.Clean(path)
}
