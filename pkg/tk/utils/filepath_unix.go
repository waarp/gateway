//go:build !windows || ignore || !plan9
// +build !windows ignore !plan9

package utils

import "path/filepath"

// NormalizePath transforms a Unix path into a valid "file" URI according to
// RFC 8089.
func NormalizePath(path string) string {
	return filepath.Clean(path)
}

// DenormalizePath transforms a "file" URI into a valid Unix path.
func DenormalizePath(path string) string {
	return filepath.Clean(path)
}
