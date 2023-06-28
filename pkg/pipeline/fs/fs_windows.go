//go:build windows

package fs

import (
	"fmt"
	"path/filepath"

	"github.com/hack-pad/hackpadfs/os"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func NewLocalFS(path *types.URL) (FS, error) {
	subVolume := filepath.VolumeName(TrimPath(path))
	cfs := os.NewFS()

	fs, err := cfs.SubVolume(subVolume)
	if err != nil {
		return nil, fmt.Errorf("failed to change the filesystem volume name: %w", err)
	}

	return &LocalFS{FS: fs.(*os.FS)}, nil
}
