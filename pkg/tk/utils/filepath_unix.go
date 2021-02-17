// +build !windows,!plan9

package utils

import "path/filepath"

// Deprecated:
// NormalizePath transforms a Unix path into a valid "file" URI according to
// RFC 8089.
func NormalizePath(path string) string {
	return filepath.Clean(path)
}

// Deprecated:
// DenormalizePath transforms a "file" URI into a valid Unix path.
func DenormalizePath(path string) string {
	return filepath.Clean(path)
}
