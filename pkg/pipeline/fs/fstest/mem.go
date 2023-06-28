// Package fstest provides a fs.FS implementation of an in-memory file system
// which can be used for testing (and should only be used for testing).
package fstest

import (
	"fmt"
	"sort"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs/flags"
)

//nolint:gochecknoglobals //this is only used in tests
var persistentFS *testFS

func newMemFS(*types.URL) (fs.FS, error) {
	if persistentFS != nil {
		return persistentFS, nil
	}

	memFS, err := mem.NewFS()
	if err != nil {
		return nil, fmt.Errorf("failed to create the in-memory filesystem: %w", err)
	}

	persistentFS = &testFS{memFS}

	return persistentFS, nil
}

func InitMemFS(c convey.C) {
	fs.FileSystems["mem"] = newMemFS

	c.Reset(func() { persistentFS = nil })
}

type testFS struct {
	*mem.FS
}

func (m *testFS) ReadDir(name string) ([]hackpadfs.DirEntry, error) {
	file, openErr := m.OpenFile(name, flags.ReadOnly, 0)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open directory: %w", openErr)
	}

	dir, canReadDir := file.(hackpadfs.DirReaderFile)
	if !canReadDir {
		return nil, fmt.Errorf(`%w: "ReadDir"`, fs.ErrNotImplemented)
	}

	dirs, readErr := dir.ReadDir(-1)
	if readErr != nil {
		return nil, fmt.Errorf("failed to retrieve the directory entries: %w", readErr)
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	return dirs, nil
}
