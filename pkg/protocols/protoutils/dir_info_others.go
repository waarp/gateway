//go:build !linux

package protoutils

// Sys returns additional info on the directory (there is never any).
func (FakeDirInfo) Sys() any { return nil }
