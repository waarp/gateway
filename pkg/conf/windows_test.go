//+build windows

package conf

import (
	"path"
	"path/filepath"
)

// On Windows systems, mkWin transforms the given Unix path into a Windows path
// (only use for test purposes).
func mkWin(dir string) string {
	if path.IsAbs(dir) {
		return "C:" + filepath.FromSlash(dir)
	}
	return filepath.FromSlash(dir)
}
