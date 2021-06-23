//+build !windows

package conf

// On Unix systems, mkWin returns the path as is.
func mkWin(path string) string {
	return path
}
