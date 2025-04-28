package fs

import (
	"fmt"
	"path/filepath"

	"github.com/hack-pad/hackpadfs/mem"
	"github.com/hack-pad/hackpadfs/os"
)

func NewLocalFS(ospath string) (FS, error) {
	subVolume := filepath.VolumeName(ospath)
	osfs := os.NewFS()

	if subVolume != "" {
		if subfs, err := osfs.SubVolume(subVolume); err != nil {
			return nil, fmt.Errorf("failed to change the filesystem volume name: %w", err)
		} else {
			//nolint:forcetypeassert,errcheck //assertion always succeeds
			osfs = subfs.(*os.FS)
		}
	}

	return osfs, nil
}

func NewMemFS() (FS, error) {
	fs, err := mem.NewFS()
	if err != nil {
		return nil, fmt.Errorf("failed to create memfs: %w", err)
	}

	return fs, nil
}
