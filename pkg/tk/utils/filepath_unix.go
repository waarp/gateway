//go:build !windows
// +build !windows

package utils

import "path/filepath"

// NormalizePath transforms a Unix path into a valid "file" URI according to
// RFC 8089.
// Deprecated: file URIs are no longer used.
func NormalizePath(path string) string {
	return filepath.Clean(path)
}

// DenormalizePath transforms a "file" URI into a valid Unix path.
// Deprecated: file URIs are no longer used.
func DenormalizePath(path string) string {
	return filepath.Clean(path)
}
