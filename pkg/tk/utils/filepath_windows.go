//go:build windows
// +build windows

package utils

import (
	"path/filepath"
)

// NormalizePath transforms a Windows path into a valid "file" URI according to
// RFC 8089.
// Deprecated
func NormalizePath(path string) string {
	norm := filepath.ToSlash(filepath.Clean(path))
	if filepath.IsAbs(path) {
		return "/" + norm
	}
	return norm
}
