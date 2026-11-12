package analytics

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// A tree for the discovery, on one volume: the home of the gateway with its
// directories, a server and a rule under it, a server and a rule outside it;
// and, next to them, directories no object refers to.
type tree struct {
	base, home, other string
	known             []LocalDir
}

// under is a path under the home of the gateway.
func (tr *tree) under(parts ...string) string {
	return filepath.Join(append([]string{tr.home}, parts...)...)
}

// outside is a path outside the home of the gateway.
func (tr *tree) outside(parts ...string) string {
	return filepath.Join(append([]string{tr.other}, parts...)...)
}

func makeTree(t *testing.T) *tree {
	t.Helper()

	base := t.TempDir()
	tr := &tree{base: base, home: filepath.Join(base, "home"), other: filepath.Join(base, "other")}

	tr.known = []LocalDir{
		{Path: tr.under("in"), Role: DirIn, Origin: OriginConfigured},
		{Path: tr.under("out"), Role: DirOut, Origin: OriginConfigured},
		{Path: tr.under("data", "srv1"), Role: RoleServerRoot, Owner: "srv1", OwnerID: 1, Origin: OriginConfigured},
		{Path: tr.under("data", "srv1", "in"), Role: RoleServerReceive, Owner: "srv1", OwnerID: 1, Origin: OriginConfigured},
		{Path: tr.under("flux", "a", "in"), Role: RoleRuleLocal, Owner: "a", OwnerID: 7, Origin: OriginConfigured},
		{Path: tr.under("later"), Role: RoleRuleLocal, Owner: "later", OwnerID: 8, Origin: OriginConfigured}, // to create
		{Path: tr.outside("srv2"), Role: RoleServerRoot, Owner: "srv2", OwnerID: 2, Origin: OriginConfigured},
		{Path: tr.outside("rule", "in"), Role: RoleRuleLocal, Owner: "r", OwnerID: 9, Origin: OriginConfigured},
	}

	for _, d := range tr.known {
		if filepath.Base(d.Path) != "later" {
			require.NoError(t, os.MkdirAll(d.Path, 0o750))
		}
	}

	// No object refers to these.
	writeFiles(t, tr.under("data", "old"), 10)               // the root of a deleted server: a file, and "sub"
	writeFiles(t, tr.under("data", "old", "sub"), 20)        // a file at depth 1, and "sub/sub"
	writeFiles(t, tr.under("data", "old", "sub", "sub"), 30) // a file at depth 2

	for _, dir := range []string{
		tr.under("data", "srv1", "leftover"), // under the root of a server
		tr.under("flux", "stray"),            // under a directory between the home and a rule
		tr.outside("srv2", "junk"),           // under the root of a server outside the home
		tr.outside("elsewhere"),              // next to a root outside the home: never listed
		tr.outside("rule", "x"),              // next to a rule outside the home: never listed
	} {
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}

	require.NoError(t, os.WriteFile(tr.under("data", "notes.txt"), []byte("x"), 0o600)) // a file is not a directory

	return tr
}

// byPath is every directory of a report, by its path.
func byPath(report *DiskReport) map[string]LocalDir {
	out := map[string]LocalDir{}

	for _, v := range report.Volumes {
		for _, d := range v.Dirs {
			out[d.Path] = d
		}
	}

	for i := range report.Unmeasured {
		out[report.Unmeasured[i].Path] = report.Unmeasured[i].LocalDir
	}

	return out
}

// paths lists the paths of directories.
func paths(dirs []LocalDir) []string {
	out := make([]string, len(dirs))
	for i := range dirs {
		out[i] = dirs[i].Path
	}

	return out
}

// The discovery finds, under the home, between the home and a known directory
// and under the root of each server, the directories no object refers to;
// never a known directory, one above a known one, nor anything outside the
// home but under the roots of the servers. They join the report at the next
// measure, surveyed at any depth right after; one leaves it when it is gone
// from the disk, and takes the role of an object that refers to it again.
func TestDiscovery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	tr := makeTree(t)
	old, leftover := tr.under("data", "old"), tr.under("data", "srv1", "leftover")
	stray, junk := tr.under("flux", "stray"), tr.outside("srv2", "junk")

	var a analytics

	a.SetHome(tr.home)

	report := measureRoot(&a, tr.base, tr.known, now)
	assert.Empty(t, a.foundDirs(), "nothing before the first cycle")

	a.Survey(ctx, now)

	found := a.foundDirs()
	assert.Equal(t, []string{old, leftover, stray, junk}, paths(found))

	for _, d := range found {
		assert.Equal(t, RoleUnreferenced, d.Role, d.Path)
		assert.Equal(t, OriginFound, d.Origin, d.Path)
		assert.Empty(t, d.Owner, d.Path)
	}

	_, listed := byPath(report)[old]
	assert.False(t, listed, "not in the report before the next measure")

	// The next measure: in the report, present, on their volume; surveyed at
	// any depth right after, as new directories.
	later := now.Add(DiskPeriod)
	report = measureRoot(&a, tr.base, tr.known, later)
	require.Len(t, report.Volumes, 1)
	assert.Equal(t, DiskOK, report.Volumes[0].State, "the state of the volume does not change")

	a.SurveyNew(ctx, later)

	dirs := byPath(a.Disks())
	assert.Equal(t, LocalDir{
		Path: old, Role: RoleUnreferenced, Origin: OriginFound, Presence: PresencePresent,
		Survey: &DirSurvey{Files: 3, Size: 60, Deep: true, At: later},
	}, dirs[old], "the files counted at any depth")
	assert.Equal(t, &DirSurvey{Deep: true, At: later}, dirs[leftover].Survey, "empty, at any depth")
	require.NotNil(t, dirs[tr.under("in")].Survey)
	assert.False(t, dirs[tr.under("in")].Survey.Deep, "a known directory: its first level")
	assert.Equal(t, RoleServerRoot, dirs[tr.under("data", "srv1")].Role, "known, as it was")

	for _, path := range []string{
		tr.under("data"), tr.under("flux"), tr.under("flux", "a"), // above known directories
		tr.outside("elsewhere"), tr.outside("rule", "x"), // outside the home, not under the root of a server
	} {
		_, listed := dirs[path]
		assert.False(t, listed, path)
	}

	// Gone from the disk: out of the report at the next measure, forgotten.
	require.NoError(t, os.RemoveAll(stray))

	dirs = byPath(measureRoot(&a, tr.base, tr.known, later.Add(DiskPeriod)))
	_, listed = dirs[stray]
	assert.False(t, listed, "gone")
	assert.Equal(t, []string{old, leftover, junk}, paths(a.foundDirs()))

	// An object refers to one again: the role of the object, and the
	// discovery no longer finds it.
	again := append(slices.Clone(tr.known),
		LocalDir{Path: leftover, Role: RoleRuleLocal, Owner: "b", OwnerID: 11, Origin: OriginConfigured})

	dirs = byPath(measureRoot(&a, tr.base, again, later.Add(2*DiskPeriod)))
	assert.Equal(t, RoleRuleLocal, dirs[leftover].Role)
	assert.Equal(t, "b", dirs[leftover].Owner)

	a.Survey(ctx, now.Add(DiskSurveyPeriod))
	assert.Equal(t, []string{old, junk}, paths(a.foundDirs()), "known now")
}

// Without the home of the gateway, only the roots of the servers are read.
func TestDiscoveryNoHome(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	tr := makeTree(t)

	var a analytics

	measureRoot(&a, tr.base, tr.known, now)
	a.Survey(ctx, now)

	assert.Equal(t, []string{tr.under("data", "srv1", "leftover"), tr.outside("srv2", "junk")}, paths(a.foundDirs()))
}

// The discovery stops at the budget of the cycle, and a listing that outlasts
// it is left to end: the next cycle reads the other structuring directories
// and comes back to that one once its listing is over.
func TestDiscoveryBudget(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	tr := makeTree(t)
	orphan := tr.under("orphan")
	require.NoError(t, os.MkdirAll(orphan, 0o750))

	old, leftover := tr.under("data", "old"), tr.under("data", "srv1", "leftover")
	stray, junk := tr.under("flux", "stray"), tr.outside("srv2", "junk")

	var (
		a       analytics
		mu      sync.Mutex
		listed  int
		release = make(chan struct{})
	)

	a.SetHome(tr.home)
	a.SetDiskSettings(DiskSettings{SurveyPeriod: time.Second}) // a budget of 200 ms
	a.disks.subdirs = func(path string, maxEntries int) ([]string, error) {
		if path == tr.home {
			mu.Lock()
			listed++
			mu.Unlock()

			<-release // the home hangs
		}

		return listSubdirNames(path, maxEntries)
	}

	count := func() int {
		mu.Lock()
		defer mu.Unlock()

		return listed
	}

	measureRoot(&a, tr.base, tr.known, now)

	started := time.Now()
	a.Survey(ctx, now)
	assert.Less(t, time.Since(started), 800*time.Millisecond, "the cycle gives up at its budget")
	assert.Empty(t, a.foundDirs(), "the home came first and hung")
	assert.Equal(t, 1, count())

	a.Survey(ctx, now.Add(DiskSurveyPeriod))
	assert.Equal(t, 1, count(), "the home, still being listed, is left to end")
	assert.Equal(t, []string{old, leftover, stray, junk}, paths(a.foundDirs()), "the other structuring directories are read")

	close(release)

	assert.Eventually(t, func() bool {
		a.disks.mu.Lock()
		defer a.disks.mu.Unlock()

		return !a.disks.survey.listing.busy(pathKey(tr.home))
	}, 2*time.Second, 10*time.Millisecond, "the listing ends")

	a.Survey(ctx, now.Add(2*DiskSurveyPeriod))
	assert.Equal(t, 2, count(), "the home is read again")
	assert.Equal(t, []string{old, leftover, stray, orphan, junk}, paths(a.foundDirs()))
}

// The files of a directory at any depth, within a cap on the entries read
// over the whole walk; a symbolic link is not followed.
func TestListAllLevels(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	c1, c2 := filepath.Join(root, "c1"), filepath.Join(root, "c1", "c2")
	require.NoError(t, os.MkdirAll(c2, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "f0"), make([]byte, 1), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(c1, "f1"), make([]byte, 2), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(c2, "f2"), make([]byte, 3), 0o600))

	if err := os.Symlink(c1, filepath.Join(root, "link")); err != nil {
		t.Logf("no symbolic link on this system: %v", err)
	}

	type count struct {
		files int
		size  int64
		more  bool
	}

	files, size, more, err := listFirstLevel(root, DiskSurveyMaxEntries)
	require.NoError(t, err)
	assert.Equal(t, count{1, 1, false}, count{files, size, more}, "the first level")

	files, size, more, err = listAllLevels(root, DiskSurveyMaxEntries)
	require.NoError(t, err)
	assert.Equal(t, count{3, 6, false}, count{files, size, more}, "any depth, the link not followed")

	files, size, more, err = listAllLevels(c1, 2)
	require.NoError(t, err)
	assert.Equal(t, count{1, 2, true}, count{files, size, more}, "the cap counted over the whole walk: c2 unread")

	_, _, _, err = listAllLevels(filepath.Join(root, "none"), DiskSurveyMaxEntries)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

// A directory the discovery found yields to the use by a transfer and to any
// object that refers to it.
func TestMergeDirsFound(t *testing.T) {
	t.Parallel()

	path := filepath.Join("home", "data", "old")
	seenAt := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	found := LocalDir{Path: path, Role: RoleUnreferenced, Origin: OriginFound}
	seen := LocalDir{Path: path, Role: RoleTransfer, Origin: OriginSeen, SeenAt: seenAt}
	rule := LocalDir{Path: path, Role: RoleRuleLocal, Owner: "a", OwnerID: 7, Origin: OriginConfigured}
	ruleSeen := rule
	ruleSeen.SeenAt = seenAt

	assert.Equal(t, []LocalDir{found}, mergeDirs([]LocalDir{found}))
	assert.Equal(t, []LocalDir{seen}, mergeDirs([]LocalDir{found, seen}))
	assert.Equal(t, []LocalDir{seen}, mergeDirs([]LocalDir{seen, found}))
	assert.Equal(t, []LocalDir{ruleSeen}, mergeDirs([]LocalDir{found, seen, rule}))
	assert.Equal(t, []LocalDir{ruleSeen}, mergeDirs([]LocalDir{rule, found, seen}))
}

// The structuring directories that appear between two cycles (the home of the
// gateway, given once the service runs; the root of a server added) are read
// right after the measure that knows them, without waiting for the next cycle.
func TestDiscoveryNew(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	tr := makeTree(t)
	old, leftover := tr.under("data", "old"), tr.under("data", "srv1", "leftover")
	stray, junk := tr.under("flux", "stray"), tr.outside("srv2", "junk")

	var (
		a      analytics
		mu     sync.Mutex
		listed []string
	)

	a.disks.subdirs = func(path string, maxEntries int) ([]string, error) {
		mu.Lock()
		listed = append(listed, path)
		mu.Unlock()

		return listSubdirNames(path, maxEntries)
	}

	// The service starts without the home: the roots of the servers only.
	measureRoot(&a, tr.base, tr.known, now)
	a.Survey(ctx, now)
	assert.Equal(t, []string{leftover, junk}, paths(a.foundDirs()))

	// The home is given: the measure knows it, the directories under it are
	// read right away.
	a.SetHome(tr.home)

	later := now.Add(DiskPeriod)
	measureRoot(&a, tr.base, tr.known, later)
	a.SurveyNew(ctx, later)
	assert.Equal(t, []string{old, leftover, stray, junk}, paths(a.foundDirs()), "without waiting for the next cycle")

	mu.Lock()
	assert.ElementsMatch(t, []string{
		tr.under("data", "srv1"), tr.outside("srv2"), // the first cycle
		tr.home, tr.under("data"), tr.under("flux"), tr.under("flux", "a"), // the new structuring directories only
	}, listed)
	mu.Unlock()

	// Nothing new: nothing is read.
	measureRoot(&a, tr.base, tr.known, later.Add(DiskPeriod))
	a.SurveyNew(ctx, later.Add(DiskPeriod))

	mu.Lock()
	assert.Len(t, listed, 6, "every structuring directory read already")
	mu.Unlock()
}

// A directory found on the disk, or seen by a transfer, that is the local or
// the temporary directory of a rule under the root of a server, as the
// transfers of that server resolve it, is reported as the directory of that
// rule, surveyed at its first level; a directory next to it that no rule names
// stays unreferenced, surveyed at any depth. A declared directory is left as
// it is, and nothing under the home but the roots of the servers is matched.
func TestClassify(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 12, 10, 40, 0, 0, time.UTC)
	tr := makeTree(t)
	srv1 := tr.under("data", "srv1")
	out, in, tmp := filepath.Join(srv1, "out"), filepath.Join(srv1, "in"), filepath.Join(srv1, "tmp")
	fno, orphan := filepath.Join(out, "FNO"), filepath.Join(out, "ORPHAN")
	rcv, rcvTmp := filepath.Join(in, "RCV"), filepath.Join(tmp, "RCV")

	writeFiles(t, fno, 5)                          // a file to send, and "sub"
	writeFiles(t, orphan, 1)                       // nobody's: a file, and "sub"
	writeFiles(t, filepath.Join(orphan, "sub"), 2) // a file at depth 1
	require.NoError(t, os.MkdirAll(rcv, 0o750))
	require.NoError(t, os.MkdirAll(rcvTmp, 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "SEEN"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(out, "SEEN"), 0o750))

	known := append(slices.Clone(tr.known),
		LocalDir{Path: out, Role: RoleServerSend, Owner: "srv1", OwnerID: 1, Origin: OriginConfigured},
		LocalDir{Path: tmp, Role: RoleServerTmp, Owner: "srv1", OwnerID: 1, Origin: OriginConfigured},
		// The directory of the rule as the configuration declares it, under the home: it does not exist here.
		LocalDir{Path: tr.under("out", "FNO"), Role: RoleRuleLocal, Owner: "FNO", OwnerID: 21, Origin: OriginConfigured},
		// Transfers: one in the directory of the receive rule, two next to the rules.
		LocalDir{Path: rcv, Role: RoleTransfer, Origin: OriginSeen, SeenAt: now},
		LocalDir{Path: filepath.Join(out, "SEEN"), Role: RoleTransfer, Origin: OriginSeen, SeenAt: now},
		LocalDir{Path: filepath.Join(tmp, "SEEN"), Role: RoleTransfer, Origin: OriginSeen, SeenAt: now},
	)

	var a analytics

	a.SetHome(tr.home)
	a.SetRuleDirs([]string{srv1, tr.outside("srv2"), ""}, []LocalDir{
		{Path: "out/FNO", Role: RoleRuleLocal, Owner: "FNO", OwnerID: 21},
		{Path: "in/RCV", Role: RoleRuleLocal, Owner: "RCV", OwnerID: 22},
		{Path: "tmp/RCV", Role: RoleRuleTmp, Owner: "RCV", OwnerID: 22},
		{Path: tr.outside("rule", "in"), Role: RoleRuleLocal, Owner: "r", OwnerID: 9}, // absolute: declared as it is
	})

	measureRoot(&a, tr.base, known, now)
	a.Survey(ctx, now)

	later := now.Add(DiskPeriod)
	measureRoot(&a, tr.base, known, later)
	a.SurveyNew(ctx, later)

	dirs := byPath(a.Disks())
	assert.Equal(t, LocalDir{
		Path: fno, Role: RoleRuleLocal, Owner: "FNO", OwnerID: 21, Origin: OriginFound, Presence: PresencePresent,
		Survey: &DirSurvey{Files: 1, Size: 5, At: later},
	}, dirs[fno], "found: the directory of its rule, its first level")
	assert.Equal(t, LocalDir{
		Path: orphan, Role: RoleUnreferenced, Origin: OriginFound, Presence: PresencePresent,
		Survey: &DirSurvey{Files: 2, Size: 3, Deep: true, At: later},
	}, dirs[orphan], "found, nobody's: unreferenced, at any depth")
	assert.Equal(t, LocalDir{
		Path: rcv, Role: RoleRuleLocal, Owner: "RCV", OwnerID: 22, Origin: OriginSeen, SeenAt: now, Presence: PresencePresent,
		Survey: &DirSurvey{At: now},
	}, dirs[rcv], "seen: the directory of its rule, the last transfer kept, surveyed since the first cycle")
	assert.Equal(t, LocalDir{
		Path: rcvTmp, Role: RoleRuleTmp, Owner: "RCV", OwnerID: 22, Origin: OriginFound, Presence: PresencePresent,
		Survey: &DirSurvey{At: later},
	}, dirs[rcvTmp], "found: the temporary directory of its rule")
	assert.Equal(t, RoleTransfer, dirs[filepath.Join(out, "SEEN")].Role, "seen, no rule names it: a transfer")
	assert.Equal(t, PresenceToCreate, dirs[tr.under("out", "FNO")].Presence, "declared under the home, left as it is")
	assert.Equal(t, RoleUnreferenced, dirs[tr.outside("srv2", "junk")].Role, "under a root, no rule names it")
	assert.Equal(t, RoleRuleLocal, dirs[tr.outside("rule", "in")].Role, "absolute, declared")
	assert.Equal(t, OriginConfigured, dirs[tr.outside("rule", "in")].Origin)
}

// The directories of the rules relative to the root of a server: the local
// directory of every rule, the temporary directory of a receive rule; an
// absolute directory is declared as it is, an empty one gives nothing.
func TestRuleDirs(t *testing.T) {
	t.Parallel()

	abs := filepath.Join(t.TempDir(), "abs")
	rules := model.Rules{
		{ID: 1, Name: "send", IsSend: true, LocalDir: "out/send", TmpLocalRcvDir: "tmp/ignored"},
		{ID: 2, Name: "recv", IsSend: false, LocalDir: "in/recv", TmpLocalRcvDir: "tmp/recv"},
		{ID: 3, Name: "abs", IsSend: true, LocalDir: abs},
		{ID: 4, Name: "none", IsSend: false},
	}

	assert.Equal(t, []LocalDir{
		{Path: "out/send", Role: RoleRuleLocal, Owner: "send", OwnerID: 1},
		{Path: "in/recv", Role: RoleRuleLocal, Owner: "recv", OwnerID: 2},
		{Path: "tmp/recv", Role: RoleRuleTmp, Owner: "recv", OwnerID: 2},
	}, ruleDirs(rules))
}
