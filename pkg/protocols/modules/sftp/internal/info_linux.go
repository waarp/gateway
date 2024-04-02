package internal

import "syscall"

// Sys returns additional info on the directory (there is never any).
func (d *DirInfo) Sys() interface{} {
	return &syscall.Stat_t{Uid: 65534, Gid: 65534} //nolint:gomnd //too specific
}
