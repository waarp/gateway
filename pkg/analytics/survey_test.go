package analytics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// measureRoot measures the volume of the root directory, named as the mount
// point that holds the given directories, now.
func measureRoot(a *analytics, root string, dirs []LocalDir, now time.Time) *DiskReport {
	a.disks.mounts = func() ([]string, error) { return []string{root}, nil }
	a.disks.usage = func(path string) (*disk.UsageStat, error) {
		return &disk.UsageStat{Path: path, Total: 100 << 30, Free: 50 << 30, UsedPercent: 50}, nil
	}

	return a.Measure(context.Background(), dirs, now)
}

// writeFiles creates a directory with a file of each given size and a
// subdirectory, which the survey does not count.
func writeFiles(t *testing.T, dir string, sizes ...int) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o750))

	for i, size := range sizes {
		require.NoError(t, os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.dat", i)), make([]byte, size), 0o600))
	}
}

// surveyOf is the survey of each directory of the first volume of a report,
// by its base name.
func surveyOf(t *testing.T, report *DiskReport) map[string]LocalDir {
	t.Helper()

	require.Len(t, report.Volumes, 1)

	out := map[string]LocalDir{}
	for _, d := range report.Volumes[0].Dirs {
		out[filepath.Base(d.Path)] = d
	}

	return out
}

// The first level of a directory: the regular files and their sizes, up to
// the cap on the entries.
func TestListFirstLevel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFiles(t, dir, 1, 2, 3) // three files and a subdirectory: four entries

	files, size, more, err := listFirstLevel(dir, 10)
	require.NoError(t, err)
	assert.Equal(t, 3, files, "the regular files only, not the subdirectory")
	assert.Equal(t, int64(6), size)
	assert.False(t, more)

	files, size, more, err = listFirstLevel(dir, 4)
	require.NoError(t, err)
	assert.Equal(t, 3, files, "a cap equal to the entries is not hit")
	assert.Equal(t, int64(6), size)
	assert.False(t, more)

	files, _, more, err = listFirstLevel(dir, 2)
	require.NoError(t, err)
	assert.True(t, more, "entries left past the cap")
	assert.LessOrEqual(t, files, 2, "the files among the entries read")

	_, _, _, err = listFirstLevel(filepath.Join(dir, "absent"), 10)
	require.ErrorIs(t, err, os.ErrNotExist)

	_, _, _, err = listFirstLevel(filepath.Join(dir, "f0.dat"), 10)
	require.Error(t, err, "a file is not a directory")
}

// The survey of the directories reaches the report at the next measure: the
// present directories only, with the time of the cycle; a directory the
// report no longer holds is forgotten.
func TestSurvey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 11, 22, 0, 0, 0, time.UTC)
	root := t.TempDir()
	in, out, later := filepath.Join(root, "in"), filepath.Join(root, "out"), filepath.Join(root, "later")
	writeFiles(t, in, 10, 20)
	writeFiles(t, out, 5)

	dirs := []LocalDir{
		{Path: in, Role: DirIn, Origin: OriginConfigured},
		{Path: out, Role: DirOut, Origin: OriginConfigured},
		{Path: later, Role: RoleRuleLocal, Owner: "flux", OwnerID: 7, Origin: OriginConfigured},
	}

	var a analytics

	report := measureRoot(&a, root, dirs, now)
	require.Len(t, report.Volumes, 1)
	require.Len(t, report.Volumes[0].Dirs, 3)
	assert.True(t, report.SurveyedAt.IsZero(), "no survey yet")

	for _, d := range report.Volumes[0].Dirs {
		assert.Nil(t, d.Survey, d.Path)
	}

	surveyedAt := now.Add(10 * time.Second)
	a.Survey(ctx, surveyedAt)

	report = measureRoot(&a, root, dirs, now.Add(DiskPeriod))
	assert.Equal(t, surveyedAt, report.SurveyedAt, "the time of the last cycle")

	byName := surveyOf(t, report)
	assert.Equal(t, &DirSurvey{Files: 2, Size: 30, At: surveyedAt}, byName["in"].Survey)
	assert.Equal(t, &DirSurvey{Files: 1, Size: 5, At: surveyedAt}, byName["out"].Survey)
	assert.Equal(t, PresenceToCreate, byName["later"].Presence)
	assert.Nil(t, byName["later"].Survey, "a directory to create has nothing to survey")
	assert.Equal(t, int64(7), byName["later"].OwnerID, "the identifier of the rule, for a link")

	// The directory is created: the next cycle surveys it.
	require.NoError(t, os.MkdirAll(later, 0o750))

	measureRoot(&a, root, dirs, now.Add(2*DiskPeriod))
	a.Survey(ctx, surveyedAt.Add(DiskSurveyPeriod))

	byName = surveyOf(t, measureRoot(&a, root, dirs, now.Add(3*DiskPeriod)))
	assert.Equal(t, PresencePresent, byName["later"].Presence)
	assert.Equal(t, &DirSurvey{At: surveyedAt.Add(DiskSurveyPeriod)}, byName["later"].Survey, "empty, surveyed")

	// The "out" directory leaves the configuration: its survey is forgotten.
	measureRoot(&a, root, dirs[:1], now.Add(4*DiskPeriod))
	a.Survey(ctx, surveyedAt.Add(2*DiskSurveyPeriod))

	a.disks.mu.Lock()
	_, kept := a.disks.survey.results[pathKey(out)]
	a.disks.mu.Unlock()

	assert.False(t, kept, "forgotten with the directory")
}

// Past the cap on the entries, the survey says "more than".
func TestSurveyCap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 11, 22, 0, 0, 0, time.UTC)
	root := t.TempDir()
	in := filepath.Join(root, "in")
	writeFiles(t, in, 1, 1, 1, 1, 1) // five files and a subdirectory

	var a analytics

	a.SetDiskSettings(DiskSettings{SurveyMaxEntries: 3})
	dirs := []LocalDir{{Path: in, Role: DirIn, Origin: OriginConfigured}}

	measureRoot(&a, root, dirs, now)
	a.Survey(ctx, now)

	survey := surveyOf(t, measureRoot(&a, root, dirs, now))["in"].Survey
	require.NotNil(t, survey)
	assert.True(t, survey.More, "more than the entries read")
	assert.LessOrEqual(t, survey.Files, 3)
	assert.GreaterOrEqual(t, survey.Files, 2, "at most one subdirectory among the three entries read")
	assert.Equal(t, int64(survey.Files), survey.Size, "the size of the files read")
}

// A cycle stops at its budget: a directory it leaves out is surveyed at the
// next cycle, which resumes after the one it stopped at; a listing that
// outlasts the budget is left to end, not started again, and its result is
// kept when it comes.
func TestSurveyBudget(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 11, 22, 0, 0, 0, time.UTC)
	root := t.TempDir()

	var dirs []LocalDir

	for _, name := range []string{"a", "b", "c"} {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(path, 0o750))
		dirs = append(dirs, LocalDir{Path: path, Role: RoleTransfer, Origin: OriginSeen})
	}

	var (
		a       analytics
		mu      sync.Mutex
		listed  []string
		release = make(chan struct{})
	)

	a.SetDiskSettings(DiskSettings{SurveyPeriod: time.Second}) // a budget of 200 ms
	a.disks.readDir = func(path string, _ int) (int, int64, bool, error) {
		mu.Lock()
		listed = append(listed, filepath.Base(path))
		mu.Unlock()

		if filepath.Base(path) == "b" {
			<-release // a listing that hangs
		}

		return 1, 1, false, nil
	}

	taken := func(report *DiskReport) map[string]time.Time {
		out := map[string]time.Time{}

		for name, d := range surveyOf(t, report) {
			if d.Survey != nil {
				out[name] = d.Survey.At
			}
		}

		return out
	}

	measureRoot(&a, root, dirs, now)

	started := time.Now()
	a.Survey(ctx, now)
	assert.Less(t, time.Since(started), 800*time.Millisecond, "the cycle gives up at its budget")
	assert.Equal(t, map[string]time.Time{"a": now}, taken(measureRoot(&a, root, dirs, now)),
		"a listed, b hung until the budget ran out, c left out")

	later := now.Add(DiskSurveyPeriod)
	a.Survey(ctx, later)
	assert.Equal(t, map[string]time.Time{"a": later, "c": later}, taken(measureRoot(&a, root, dirs, later)),
		"the cycle resumed after b: c, then a again; b, still being listed, is left to end")

	mu.Lock()
	assert.Equal(t, []string{"a", "b", "c", "a"}, listed)
	mu.Unlock()

	close(release)

	assert.Eventually(t, func() bool {
		return taken(measureRoot(&a, root, dirs, later))["b"].Equal(now)
	}, 2*time.Second, 10*time.Millisecond, "the abandoned listing ends and its result is kept, dated from its cycle")
}

// A directory the survey may not read is reported as denied, with the error.
func TestSurveyDenied(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 11, 22, 0, 0, 0, time.UTC)
	root := t.TempDir()
	in := filepath.Join(root, "in")
	require.NoError(t, os.MkdirAll(in, 0o750))

	var a analytics

	a.disks.readDir = func(path string, _ int) (int, int64, bool, error) {
		return 0, 0, false, &os.PathError{Op: "open", Path: path, Err: os.ErrPermission}
	}

	dirs := []LocalDir{{Path: in, Role: DirIn, Origin: OriginConfigured}}

	assert.Equal(t, PresencePresent, surveyOf(t, measureRoot(&a, root, dirs, now))["in"].Presence)
	a.Survey(ctx, now)

	dir := surveyOf(t, measureRoot(&a, root, dirs, now))["in"]
	assert.Equal(t, PresenceDenied, dir.Presence)
	require.NotNil(t, dir.Survey)
	assert.Contains(t, dir.Survey.Error, "permission denied")
	assert.Zero(t, dir.Survey.Files)
}

// The settings take their defaults when left empty; the budget is a share of
// the period.
func TestDiskSettings(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultDiskSettings(), DiskSettings{}.withDefaults())
	assert.Equal(t, DiskSurveyPeriod/surveyBudgetShare, DefaultDiskSettings().Budget())

	set := DiskSettings{MeasurePeriod: time.Minute, SurveyPeriod: 10 * time.Minute, SurveyMaxEntries: 5}
	assert.Equal(t, set, set.withDefaults(), "settings given are kept")
	assert.Equal(t, 2*time.Minute, set.Budget())

	var a analytics

	assert.Equal(t, DefaultDiskSettings(), a.DiskSettings(), "without settings, the defaults")

	a.SetDiskSettings(DiskSettings{SurveyMaxEntries: -1, SurveyPeriod: time.Minute})
	assert.Equal(t, DiskSurveyMaxEntries, a.DiskSettings().SurveyMaxEntries, "a value below zero takes the default")
	assert.Equal(t, time.Minute, a.DiskSettings().SurveyPeriod)
}

// The surveys of a cycle reach the current report at its end, in a fresh copy
// of the report, without waiting for the next measure; the report already
// handed out is left as it was, its readers may still be reading it.
func TestSurveyPublishes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 12, 9, 39, 0, 0, time.UTC)
	root := t.TempDir()
	in := filepath.Join(root, "in")
	writeFiles(t, in, 10, 20)

	dirs := []LocalDir{{Path: in, Role: DirIn, Origin: OriginConfigured}}

	var a analytics

	measured := measureRoot(&a, root, dirs, now)
	surveyedAt := now.Add(10 * time.Second)
	a.Survey(ctx, surveyedAt)

	report := a.Disks()
	require.NotNil(t, report)
	assert.NotSame(t, measured, report, "a fresh report, the measured one being read by others")
	assert.Equal(t, now, report.MeasuredAt, "the same measure")
	assert.Equal(t, surveyedAt, report.SurveyedAt, "the time of the cycle, already")
	assert.Equal(t, &DirSurvey{Files: 2, Size: 30, At: surveyedAt}, surveyOf(t, report)["in"].Survey,
		"the survey, without waiting for the next measure")
	assert.Nil(t, surveyOf(t, measured)["in"].Survey, "the report handed out before the cycle is left as it was")
}

// A present directory never surveyed is surveyed right after the measure that
// finds it, without waiting for the next cycle: the directories of the
// gateway, given once the service runs, or a server or a rule added; the
// directories already surveyed are left to the cycles.
func TestSurveyNew(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 12, 9, 39, 0, 0, time.UTC)
	root := t.TempDir()
	in, out := filepath.Join(root, "in"), filepath.Join(root, "out")
	writeFiles(t, in, 10, 20)
	writeFiles(t, out, 5)

	first := []LocalDir{{Path: in, Role: RoleTask, Owner: "flux", OwnerID: 7, Origin: OriginTask}}
	both := []LocalDir{first[0], {Path: out, Role: DirOut, Origin: OriginConfigured}}

	var a analytics

	measureRoot(&a, root, first, now)
	a.Survey(ctx, now)

	// The home of the gateway is given, or an object added: "out" joins the report.
	later := now.Add(DiskPeriod)
	measureRoot(&a, root, both, later)
	assert.Nil(t, surveyOf(t, a.Disks())["out"].Survey, "not surveyed yet")

	a.SurveyNew(ctx, later)

	byName := surveyOf(t, a.Disks())
	assert.Equal(t, &DirSurvey{Files: 1, Size: 5, At: later}, byName["out"].Survey, "surveyed without waiting for the cycle")
	assert.Equal(t, &DirSurvey{Files: 2, Size: 30, At: now}, byName["in"].Survey, "left to the cycles")
	assert.Equal(t, now, a.Disks().SurveyedAt, "the time of the last cycle, unchanged")

	// Nothing new: nothing is listed.
	var (
		mu     sync.Mutex
		listed int
	)

	a.disks.readDir = func(string, int) (int, int64, bool, error) {
		mu.Lock()
		defer mu.Unlock()

		listed++

		return 0, 0, false, nil
	}

	a.SurveyNew(ctx, later.Add(DiskPeriod))

	mu.Lock()
	assert.Zero(t, listed, "every directory surveyed already")
	mu.Unlock()
}

// A directory is listed once at a time: a listing that outlasts its cycle is
// left to end, neither the next cycle nor a survey of the new directories
// starts another, and its result is kept when it comes.
func TestSurveyPending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 11, 12, 9, 39, 0, 0, time.UTC)
	root := t.TempDir()
	slow := filepath.Join(root, "slow")
	require.NoError(t, os.MkdirAll(slow, 0o750))

	dirs := []LocalDir{{Path: slow, Role: RoleTransfer, Origin: OriginSeen}}

	var (
		a       analytics
		mu      sync.Mutex
		listed  int
		release = make(chan struct{})
	)

	a.SetDiskSettings(DiskSettings{SurveyPeriod: time.Second}) // a budget of 200 ms
	a.disks.readDir = func(string, int) (int, int64, bool, error) {
		mu.Lock()
		listed++
		mu.Unlock()

		<-release // a listing that hangs

		return 3, 30, false, nil
	}

	count := func() int {
		mu.Lock()
		defer mu.Unlock()

		return listed
	}

	measureRoot(&a, root, dirs, now)
	a.Survey(ctx, now)
	assert.Equal(t, 1, count(), "listed, until the budget ran out")
	assert.Nil(t, surveyOf(t, a.Disks())["slow"].Survey, "no result yet")

	a.SurveyNew(ctx, now.Add(DiskPeriod))
	assert.Equal(t, 1, count(), "never surveyed, but being listed: not started again")

	a.Survey(ctx, now.Add(DiskSurveyPeriod))
	assert.Equal(t, 1, count(), "the next cycle leaves it to end")

	close(release)

	assert.Eventually(t, func() bool {
		a.disks.mu.Lock()
		defer a.disks.mu.Unlock()

		return !a.disks.survey.pending.busy(pathKey(slow))
	}, 2*time.Second, 10*time.Millisecond, "the listing ends")

	third := now.Add(2 * DiskSurveyPeriod)
	a.Survey(ctx, third)
	assert.Equal(t, 2, count(), "listed again by the next cycle")
	assert.Equal(t, &DirSurvey{Files: 3, Size: 30, At: third}, surveyOf(t, a.Disks())["slow"].Survey)
}
