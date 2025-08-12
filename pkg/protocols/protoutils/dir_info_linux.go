package protoutils

import "syscall"

// Sys returns additional info on the directory (there is never any).
func (FakeDirInfo) Sys() any {
	return &syscall.Stat_t{Uid: 65534, Gid: 65534} //nolint:mnd //too specific
}
