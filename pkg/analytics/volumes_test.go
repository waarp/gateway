package analytics

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The volumes the transfers use: the directories grouped by volume,
// each volume measured once, within a time limit, against the thresholds.
func TestMeasure(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

	t.Run("Given two directories of one volume", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		in, out := filepath.Join(root, "in"), filepath.Join(root, "out")
		require.NoError(t, os.MkdirAll(in, 0o750))
		require.NoError(t, os.MkdirAll(out, 0o750))

		var (
			a     analytics
			calls []string
		)

		a.disks.mounts = func() ([]string, error) { return []string{root, filepath.Dir(root)}, nil }
		a.disks.usage = func(path string) (*disk.UsageStat, error) {
			calls = append(calls, path)

			return &disk.UsageStat{Path: path, Total: 100 << 30, Free: 15 << 30, UsedPercent: 85, InodesTotal: 1000, InodesUsedPercent: 3}, nil
		}

		report := a.Measure(context.Background(), []LocalDir{
			{Path: in, Role: DirIn, Origin: OriginConfigured},
			{Path: out, Role: DirOut, Origin: OriginConfigured},
		}, now)

		require.Len(t, report.Volumes, 1, "one volume holds both directories")
		assert.Len(t, calls, 1, "measured once")
		assert.Equal(t, root, report.Volumes[0].Key, "the longest mount point under the directories")
		assert.Len(t, report.Volumes[0].Dirs, 2)
		assert.InDelta(t, 85, report.Volumes[0].UsedPercent, 0)
		assert.Equal(t, uint64(15<<30), report.Volumes[0].Free)
		assert.True(t, report.Volumes[0].InodesKnown)
		assert.Equal(t, DiskWarning, report.Volumes[0].State, "85 % used: warning")
		assert.Empty(t, report.Unmeasured)
		assert.Equal(t, now, report.MeasuredAt)
		assert.Same(t, report, a.Disks(), "the report is kept")

		usage, ok := a.Disks().Usage(DirOut)
		require.True(t, ok, "the former per-directory reading comes from the volume")
		assert.Equal(t, out, usage.Path)
		assert.InDelta(t, 85, usage.UsedPercent, 0)

		worst, ok := report.Worst()
		require.True(t, ok)
		assert.Equal(t, root, worst.Key)
	})

	t.Run("Given directories that cannot be measured", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		ok := filepath.Join(root, "ok")
		require.NoError(t, os.MkdirAll(ok, 0o750))

		var a analytics

		// The system names one mount point, the "ok" directory; the missing
		// directory lies under none, so its volume is found through the
		// nearest directory above it.
		a.disks.mounts = func() ([]string, error) { return []string{ok}, nil }

		// The volumes are measured each in its own goroutine: the probes come in
		// any order.
		var (
			probed []string
			mu     sync.Mutex
		)

		a.disks.usage = func(path string) (*disk.UsageStat, error) {
			mu.Lock()
			probed = append(probed, path)
			mu.Unlock()

			return &disk.UsageStat{Path: path, Total: 10 << 30, Free: 9 << 30, UsedPercent: 10}, nil
		}

		report := a.Measure(context.Background(), []LocalDir{
			{Path: filepath.Join(root, "absent", "deeper"), Role: RoleServerRoot, Owner: "srv", Origin: OriginConfigured},
			{Path: ok, Role: RoleRuleLocal, Owner: "releves", Origin: OriginConfigured},
			{Path: "S3-X:reception/jour", Role: RoleTask, Owner: "archives", Origin: OriginTask},
		}, now)

		require.Len(t, report.Volumes, 2, "the mount point, and the volume of the missing directory")
		assert.ElementsMatch(t, []string{root, ok}, probed,
			"a missing directory is measured through the nearest directory above it (the gateway creates it at the first transfer); a mount point on itself")

		for _, v := range report.Volumes {
			assert.Equal(t, DiskWarning, v.State, "9 GB free, under 10 GB: warning")
		}

		require.Len(t, report.Unmeasured, 1, "the missing directory is no incident")
		assert.Equal(t, "S3-X:reception/jour", report.Unmeasured[0].Path)
		assert.Equal(t, ReasonCloud, report.Unmeasured[0].Reason)
		assert.False(t, report.Unmeasured[0].Local(), "a cloud storage has no volume: no incident")
	})

	t.Run("Given a directory with nothing above it", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS != "windows" {
			t.Skip("on Unix the root always exists; a gone drive is a Windows case")
		}

		var a analytics

		a.disks.mounts = func() ([]string, error) { return nil, nil }

		report := a.Measure(context.Background(), []LocalDir{{Path: `Q:\flux\in`, Role: RoleRuleLocal, Owner: "flux"}}, now)
		assert.Empty(t, report.Volumes)
		require.Len(t, report.Unmeasured, 1)
		assert.Equal(t, ReasonAbsent, report.Unmeasured[0].Reason, "the drive is gone")
		assert.True(t, report.Unmeasured[0].Local(), "an incident")
	})

	t.Run("Given a volume that does not answer", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		release := make(chan struct{})

		var a analytics

		a.disks.timeout = 50 * time.Millisecond
		a.disks.mounts = func() ([]string, error) { return nil, nil }
		a.disks.usage = func(path string) (*disk.UsageStat, error) {
			<-release // a statfs on a hard mount whose server is gone

			return &disk.UsageStat{Path: path}, nil
		}

		started := time.Now()
		report := a.Measure(context.Background(), []LocalDir{{Path: dir, Role: DirTmp}}, now)
		close(release)

		assert.Less(t, time.Since(started), 2*time.Second, "the measure gives up within the limit")
		assert.Empty(t, report.Volumes)
		require.Len(t, report.Unmeasured, 1)
		assert.Equal(t, ReasonTimeout, report.Unmeasured[0].Reason)
		assert.Equal(t, dir, report.Unmeasured[0].Path)
	})

	t.Run("Given the thresholds", func(t *testing.T) {
		t.Parallel()

		gb := func(n uint64) uint64 { return n << 30 }

		assert.Equal(t, DiskOK, stateOf(&disk.UsageStat{UsedPercent: 79.9, Free: gb(50)}))
		assert.Equal(t, DiskWarning, stateOf(&disk.UsageStat{UsedPercent: 80, Free: gb(50)}), "80 % used")
		assert.Equal(t, DiskWarning, stateOf(&disk.UsageStat{UsedPercent: 10, Free: gb(10) - 1}), "under 10 GB free")
		assert.Equal(t, DiskCritical, stateOf(&disk.UsageStat{UsedPercent: 90, Free: gb(50)}), "90 % used")
		assert.Equal(t, DiskCritical, stateOf(&disk.UsageStat{UsedPercent: 10, Free: gb(2) - 1}), "under 2 GB free")
		assert.Equal(t, DiskWarning, stateOf(&disk.UsageStat{UsedPercent: 10, Free: gb(50), InodesTotal: 100, InodesUsedPercent: 85}),
			"the inodes have the same thresholds")
		assert.Equal(t, DiskCritical, stateOf(&disk.UsageStat{UsedPercent: 10, Free: gb(50), InodesTotal: 100, InodesUsedPercent: 95}))
		assert.Equal(t, DiskOK, stateOf(&disk.UsageStat{UsedPercent: 10, Free: gb(50), InodesUsedPercent: 95}),
			"inodes the system does not count (Windows) do not count")
	})

	t.Run("Given the order of the volumes", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		dirs := map[string]*disk.UsageStat{}

		var list []LocalDir

		for name, usage := range map[string]*disk.UsageStat{
			"a": {UsedPercent: 50, Free: 50 << 30}, "b": {UsedPercent: 95, Free: 5 << 30},
			"c": {UsedPercent: 70, Free: 1 << 30}, "d": {UsedPercent: 60, Free: 50 << 30},
		} {
			path := filepath.Join(root, name)
			require.NoError(t, os.MkdirAll(path, 0o750))
			dirs[path] = usage
			list = append(list, LocalDir{Path: path, Role: RoleTransfer, Origin: OriginSeen})
		}

		var a analytics

		mounts := make([]string, 0, len(dirs))
		for path := range dirs {
			mounts = append(mounts, path)
		}

		a.disks.mounts = func() ([]string, error) { return mounts, nil } // each directory its own volume
		a.disks.usage = func(path string) (*disk.UsageStat, error) { return dirs[path], nil }

		report := a.Measure(context.Background(), list, now)
		require.Len(t, report.Volumes, 4)

		var keys []string
		for _, v := range report.Volumes {
			keys = append(keys, filepath.Base(v.Key))
		}

		assert.Equal(t, []string{"b", "c", "d", "a"}, keys, "the most severe first (b, then c under 2 GB free), then the fullest")
	})
}

// The directories seen by the transfers stay for the window.
func TestSeen(t *testing.T) {
	t.Parallel()

	var a analytics

	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	a.Seen("/data/flux/paie.xml", now.Add(-8*24*time.Hour))
	a.Seen("/data/flux/paie-2.xml", now.Add(-time.Hour))
	a.Seen("/data/flux/paie-3.xml", now.Add(-2*time.Hour))
	a.Seen("/data/archives/a.zip", now)
	a.Seen("S3-X:reception/f.txt", now)
	a.Seen("", now)

	dirs := a.seenDirs(now)
	require.Len(t, dirs, 2, "one entry per directory, the cloud paths and the empty one left out")
	assert.Equal(t, filepath.FromSlash("/data/archives"), dirs[0].Path, "written as the system writes its paths")
	assert.Equal(t, filepath.FromSlash("/data/flux"), dirs[1].Path)
	assert.Equal(t, now.Add(-time.Hour), dirs[1].SeenAt, "the last transfer seen")
	assert.Equal(t, RoleTransfer, dirs[1].Role)
	assert.Equal(t, OriginSeen, dirs[1].Origin)

	dirs = a.seenDirs(now.Add(8 * 24 * time.Hour))
	assert.Empty(t, dirs, "forgotten after the window")
}

// The fixed part of a task destination, before its first variable.
func TestDestinationDir(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		arg   string
		isDir bool
		dir   string
		cloud bool
	}{
		{"/a/b/#DATE#/c", true, "/a/b", false},
		{"/a/b/", true, "/a/b", false},
		{"/a/b", true, "/a/b", false},
		{"/a/b#X#", true, "/a", false},
		{"/FT/02X55F1/ABPQ.r.#REMOTETRANSFERID#", false, "/FT/02X55F1", false},
		{"/archives/#TRUEFILENAME#.zip", false, "/archives", false},
		{"/archives/sortie.zip", false, "/archives", false},
		{"S3-X:dir", true, "S3-X:dir", true},
		{"S3-X:dir/#DATE#", true, "S3-X:dir/", true},
		{"#OUTPATH#/x", true, "", false},
		{"", false, "", false},
		{"/", true, "/", false},
	} {
		dir, cloud := DestinationDir(c.arg, c.isDir)
		want := c.dir
		if !c.cloud && want != "" {
			want = filepath.FromSlash(want) // written as the system writes its paths
		}

		assert.Equalf(t, want, dir, "%q", c.arg)
		assert.Equalf(t, c.cloud, cloud, "%q", c.arg)
	}
}

// abs makes an absolute path out of a slash path, as the product writes them
// (with slashes, under a drive on Windows, where "/gw" is not absolute).
func abs(slash string) string {
	if runtime.GOOS == "windows" {
		return "C:" + slash
	}

	return slash
}

// A relative directory of an object is resolved as the transfers resolve it:
// under its parents in order, then under the home of the gateway.
func TestResolve(t *testing.T) {
	t.Parallel()

	var a analytics

	a.SetHome(abs("/gw"))
	assert.Equal(t, abs("/gw/in"), a.Resolve("in"))
	assert.Equal(t, abs("/srv/in"), a.Resolve("in", abs("/srv")), "under the root of its server")
	assert.Equal(t, abs("/gw/srv/in"), a.Resolve("in", "srv"), "a relative root under the home")
	assert.Equal(t, abs("/data/in"), a.Resolve(abs("/data/in"), abs("/srv")), "an absolute directory as it is")
	assert.Equal(t, abs("/gw/x"), a.Resolve("x", ""), "an empty parent is skipped")
	assert.Empty(t, a.Resolve(""))
}

// The volume of a directory is the longest mount point it lies under; the
// keys are written as the system writes its paths.
func TestVolumeKey(t *testing.T) {
	t.Parallel()

	mounts := []string{abs("/data/flux"), abs("/data"), abs("/")}

	key, mounted := volumeKey(abs("/data/flux/paie"), mounts)
	assert.Equal(t, abs("/data/flux"), key)
	assert.True(t, mounted)

	key, _ = volumeKey(abs("/data/fluxion"), mounts)
	assert.Equal(t, abs("/data"), key, "a name that only starts like a mount point")

	key, _ = volumeKey(abs("/var/log"), mounts)
	assert.Equal(t, abs("/"), key)

	alone := filepath.Clean(abs("/var/log"))
	if drive := filepath.VolumeName(alone); drive != "" {
		alone = drive + string(filepath.Separator) // Windows: the drive of the directory
	}

	key, mounted = volumeKey(abs("/var/log"), nil)
	assert.Equal(t, alone, key, "without mount points, the drive or the directory itself")
	assert.False(t, mounted)
}

// One directory per path, whatever its origins and its spellings: the first
// configured role kept, the last use by a transfer attached.
func TestMergeDirs(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 11, 22, 0, 0, 0, time.UTC)

	merged := mergeDirs([]LocalDir{
		{Path: abs("/gw/in"), Role: DirIn, Origin: OriginConfigured},
		{Path: abs("/gw//in/"), Role: RoleServerReceive, Owner: "sftp", OwnerID: 3, Origin: OriginConfigured},
		{Path: filepath.FromSlash(abs("/gw/in")), Role: RoleTransfer, Origin: OriginSeen, SeenAt: now.Add(-time.Hour)},
		{Path: abs("/gw/in"), Role: RoleTransfer, Origin: OriginSeen, SeenAt: now},
		{Path: abs("/data/flux"), Role: RoleTransfer, Origin: OriginSeen, SeenAt: now},
		{Path: abs("/data/flux"), Role: RoleRuleLocal, Owner: "flux", OwnerID: 5, Origin: OriginConfigured},
		{Path: "S3-X:reception/jour", Role: RoleTask, Owner: "archives", Origin: OriginTask},
		{Path: "", Role: DirTmp, Origin: OriginConfigured},
	})
	require.Len(t, merged, 3, "the spellings of one directory merged, the empty path left out")

	in := merged[0]
	assert.Equal(t, filepath.Clean(abs("/gw/in")), in.Path, "written as the system writes its paths")
	assert.Equal(t, DirIn, in.Role, "the first configured role")
	assert.Equal(t, OriginConfigured, in.Origin)
	assert.Equal(t, now, in.SeenAt, "the last transfer that used it")

	flux := merged[1]
	assert.Equal(t, RoleRuleLocal, flux.Role, "seen by a transfer first: the configuration still wins")
	assert.Equal(t, "flux", flux.Owner)
	assert.Equal(t, int64(5), flux.OwnerID)
	assert.Equal(t, OriginConfigured, flux.Origin)
	assert.Equal(t, now, flux.SeenAt)

	assert.Equal(t, "S3-X:reception/jour", merged[2].Path, "a cloud path as it is")

	if runtime.GOOS == "windows" {
		merged = mergeDirs([]LocalDir{{Path: `C:\GW\IN`, Role: DirIn}, {Path: `c:/gw/in`, Role: DirOut}})
		require.Len(t, merged, 1, "the case does not tell two directories apart")
		assert.Equal(t, `C:\GW\IN`, merged[0].Path, "the first spelling")
	}
}

// Every directory of the report says whether it exists, and a directory given
// twice is measured once.
func TestPresence(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 11, 22, 0, 0, 0, time.UTC)
	root := t.TempDir()
	present := filepath.Join(root, "in")
	require.NoError(t, os.MkdirAll(present, 0o750))

	var a analytics

	a.disks.mounts = func() ([]string, error) { return []string{root}, nil }
	a.disks.usage = func(path string) (*disk.UsageStat, error) {
		return &disk.UsageStat{Path: path, Total: 100 << 30, Free: 50 << 30, UsedPercent: 50}, nil
	}

	report := a.Measure(context.Background(), []LocalDir{
		{Path: present, Role: DirIn, Origin: OriginConfigured},
		{Path: present + "/", Role: RoleServerReceive, Owner: "sftp", OwnerID: 3, Origin: OriginConfigured},
		{Path: filepath.Join(present, "later"), Role: RoleRuleLocal, Owner: "flux", OwnerID: 5, Origin: OriginConfigured},
		{Path: "S3-X:reception", Role: RoleTask, Owner: "archives", Origin: OriginTask},
	}, now)

	require.Len(t, report.Volumes, 1)
	require.Len(t, report.Volumes[0].Dirs, 2, "the two spellings of the present directory make one")
	assert.Equal(t, PresencePresent, report.Volumes[0].Dirs[0].Presence)
	assert.Equal(t, DirIn, report.Volumes[0].Dirs[0].Role)
	assert.Equal(t, PresenceToCreate, report.Volumes[0].Dirs[1].Presence, "measured by the volume that will hold it")
	require.Len(t, report.Unmeasured, 1)
	assert.Equal(t, PresenceCloud, report.Unmeasured[0].Presence)

	t.Run("Given a volume that does not answer", func(t *testing.T) {
		t.Parallel()

		release := make(chan struct{})

		var b analytics

		b.disks.timeout = 50 * time.Millisecond
		b.disks.mounts = func() ([]string, error) { return nil, nil }
		b.disks.usage = func(path string) (*disk.UsageStat, error) {
			<-release

			return &disk.UsageStat{Path: path}, nil
		}

		report := b.Measure(context.Background(), []LocalDir{{Path: present, Role: DirIn}}, now)
		close(release)

		require.Len(t, report.Unmeasured, 1)
		assert.Equal(t, PresenceUnresponsive, report.Unmeasured[0].Presence)
	})

	t.Run("Given a directory with nothing above it", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS != "windows" {
			t.Skip("on Unix the root always exists; a gone drive is a Windows case")
		}

		var c analytics

		c.disks.mounts = func() ([]string, error) { return nil, nil }

		report := c.Measure(context.Background(), []LocalDir{{Path: `Q:\flux\in`, Role: RoleRuleLocal}}, now)
		require.Len(t, report.Unmeasured, 1)
		assert.Equal(t, PresenceMissing, report.Unmeasured[0].Presence, "the drive is gone")
	})

	t.Run("Given a directory the process may not look up", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("the permissions of a directory stop neither Windows nor root")
		}

		locked := filepath.Join(root, "locked")
		require.NoError(t, os.MkdirAll(filepath.Join(locked, "inner"), 0o750))
		require.NoError(t, os.Chmod(locked, 0o000))
		t.Cleanup(func() { _ = os.Chmod(locked, 0o750) })

		report := a.Measure(context.Background(), []LocalDir{{Path: filepath.Join(locked, "inner"), Role: DirOut}}, now)
		require.Len(t, report.Unmeasured, 1)
		assert.Equal(t, PresenceDenied, report.Unmeasured[0].Presence)
		assert.Equal(t, ReasonDenied, report.Unmeasured[0].Reason)
	})
}
