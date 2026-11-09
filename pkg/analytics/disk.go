package analytics

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/shirou/gopsutil/v4/disk"
)

// The names of the directories of the gateway the service measures.
const (
	DirIn       = "in"
	DirOut      = "out"
	DirTmp      = "tmp"
	DirDatabase = "database"
)

var ErrUnknownDir = errors.New("unknown directory")

// Dir is a directory of the gateway whose file system the service measures.
type Dir struct {
	Name, Path string
}

// DiskUsage is the usage of the file system of a directory: the share used
// and the bytes free.
type DiskUsage struct {
	Dir
	UsedPercent float64
	Free        uint64
}

// dirs holds the directories to measure, given by the daemon once they
// exist.
type dirs struct {
	mu   sync.RWMutex
	list []Dir
}

// SetDirs gives the service the directories to measure.
func (a *analytics) SetDirs(list []Dir) {
	a.dirs.mu.Lock()
	defer a.dirs.mu.Unlock()

	a.dirs.list = slices.Clone(list)
}

// Dirs is the list of the directories measured.
func (a *analytics) Dirs() []Dir {
	a.dirs.mu.RLock()
	defer a.dirs.mu.RUnlock()

	return slices.Clone(a.dirs.list)
}

// DiskUsageOf is the usage of the file system of the directory of that name:
// the last measure of its volume when there is one, a measure taken
// now otherwise.
func (a *analytics) DiskUsageOf(name string) (DiskUsage, error) {
	if usage, ok := a.Disks().Usage(name); ok {
		return usage, nil
	}

	for _, dir := range a.Dirs() {
		if dir.Name != name {
			continue
		}

		usage, err := disk.Usage(dir.Path)
		if err != nil {
			return DiskUsage{}, fmt.Errorf("failed to measure the file system of %q: %w", dir.Path, err)
		}

		return DiskUsage{Dir: dir, UsedPercent: usage.UsedPercent, Free: usage.Free}, nil
	}

	return DiskUsage{}, fmt.Errorf("%w: %q", ErrUnknownDir, name)
}
