//go:build windows
// +build windows

package utils

import (
	"path"
	"path/filepath"
)

// NormalizePath transforms a Windows path into a valid "file" URI according to
// RFC 8089.
func NormalizePath(path string) string {
	norm := filepath.ToSlash(filepath.Clean(path))
	if filepath.IsAbs(path) {
		return "/" + norm
	}
	return norm
}

// DenormalizePath transforms a "file" URI into a valid Windows path.
func DenormalizePath(norm string) string {
	if path.IsAbs(norm) {
		norm = norm[1:]
	}
	return filepath.FromSlash(norm)
}
