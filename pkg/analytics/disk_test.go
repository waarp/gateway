package analytics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskUsageOf(t *testing.T) {
	var a analytics

	_, err := a.DiskUsageOf(DirIn)
	require.ErrorIs(t, err, ErrUnknownDir, "no directory before the daemon sets them")

	dir := t.TempDir()
	a.SetDirs([]Dir{{Name: DirIn, Path: dir}})

	usage, err := a.DiskUsageOf(DirIn)
	require.NoError(t, err)
	assert.Equal(t, dir, usage.Path)
	assert.GreaterOrEqual(t, usage.UsedPercent, 0.0)
	assert.LessOrEqual(t, usage.UsedPercent, 100.0)
	assert.Positive(t, usage.Free)

	_, err = a.DiskUsageOf(DirOut)
	require.ErrorIs(t, err, ErrUnknownDir)

	a.SetDirs([]Dir{{Name: DirOut, Path: "/nowhere/such/dir"}})
	_, err = a.DiskUsageOf(DirOut)
	require.Error(t, err, "a directory that cannot be measured")
	assert.NotErrorIs(t, err, ErrUnknownDir)
}
