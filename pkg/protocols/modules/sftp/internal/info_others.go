//go:build !linux
// +build !linux

package internal

// Sys returns additional info on the directory (there is never any).
func (d *DirInfo) Sys() interface{} {
	return nil
}
