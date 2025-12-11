package internal

import (
	"time"

	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/vfs/vfscommon"
)

func VFSOpts() *vfscommon.Options {
	const (
		readWaitDuration = rfs.Duration(100 * time.Millisecond)
	)

	vfsOpts := &vfscommon.Options{
		FastFingerprint: true,
		CacheMode:       vfscommon.CacheModeOff,
		ReadWait:        readWaitDuration,
	}

	return vfsOpts
}
