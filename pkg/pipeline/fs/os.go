package fs

import (
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/os"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs/flags"
)

type LocalFS struct {
	*os.FS
}

//nolint:wrapcheck //wrapping the error adds nothing
func (l *LocalFS) Create(name string) (hackpadfs.File, error) {
	//nolint:gomnd //magic number is needed here to mimic the behavior of os.Create
	return l.OpenFile(name, flags.ReadWrite|flags.Create|flags.Truncate, 0o666)
}
