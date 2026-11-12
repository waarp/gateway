package analytics

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The survey of the directories: the first level of every present local
// directory of the last report, never below, read on its own period within a
// time budget, so that the free space of the volumes is measured on time
// whatever a listing costs. The free space of a volume is a counter the system
// keeps; the content of a directory exists nowhere and has to be listed: an
// exchange directory takes milliseconds, an archive of hundreds of thousands
// of files or a network share can take minutes and load the disk. Hence the
// cap on the entries read per directory, past which the count says "more
// than", and the budget per cycle: the cycles go round the directories in the
// order of their paths, each resuming after the directory the previous one
// stopped at, so that a directory left out is surveyed at the next cycle. A
// present directory never surveyed (the directories of the gateway, given once
// the service runs; a server or a rule added) is surveyed right after the
// measure that finds it, within the same budget, without waiting for the next
// cycle. A directory is listed once at a time: a listing that outlasts its
// cycle is left to end, not started again. The surveys reach the current
// report at the end of each cycle; a result that comes later reaches it at
// the next measure of the volumes. Each cycle starts by the discovery of the
// directories no object refers to (discovery.go), the only ones whose files
// are counted at any depth, since nothing else looks under them.

const (
	// DiskSurveyPeriod is how often the directories are surveyed, by default.
	DiskSurveyPeriod = 5 * time.Minute
	// DiskSurveyMaxEntries is how many entries of a directory are read before
	// the survey stops and says "more than", by default.
	DiskSurveyMaxEntries = 10_000

	// A cycle spends at most this share of its period (surveyBudgetShare).
	surveyBudgetShare = 5
	// How many entries are asked from the system at once (readDirBatch).
	readDirBatch = 512
)

// DirSurvey is the survey of a directory: the regular files it holds and
// their total size, at its first level, or at any depth (Deep) for a directory
// no object refers to; when it was taken; and whether the cap on the entries
// was hit, in which case the directory holds at least that many files, of at
// least that size. Error is the refusal of the system when the directory could
// not be listed (the count and the size are then zero).
type DirSurvey struct {
	Files int
	Size  int64
	More  bool
	Deep  bool
	At    time.Time
	Error string
}

// surveyState is what the service keeps of the survey between the cycles,
// under the lock of disks: the result of each directory by its path key, the
// directories being listed, the structuring directories the discovery is
// reading and those it has read (by their key, with when), the directories it
// found, the key of the directory the last cycle stopped at, and when the last
// cycle ran.
type surveyState struct {
	results  map[string]surveyResult
	pending  readings
	listing  readings
	explored map[string]time.Time
	found    map[string]foundDir
	last     string
	at       time.Time
}

// surveyResult is the survey of a directory with the error of the system,
// which says whether the directory may be read.
type surveyResult struct {
	DirSurvey
	err error
}

// readings is the set of the paths being read, by their key, under the lock
// of disks: a reading that outlasts its cycle is left to end, not started
// again.
type readings struct {
	set map[string]struct{}
}

// add marks a key as being read; false when it is already.
func (r *readings) add(key string) bool {
	if r.busy(key) {
		return false
	}

	if r.set == nil {
		r.set = map[string]struct{}{}
	}

	r.set[key] = struct{}{}

	return true
}

// drop marks the end of the reading of a key.
func (r *readings) drop(key string) { delete(r.set, key) }

// busy says whether a key is being read.
func (r *readings) busy(key string) bool {
	_, busy := r.set[key]

	return busy
}

// Survey runs a cycle, now: the discovery of the directories no object refers
// to, then the survey of the present local directories of the last report,
// within the budget of the settings: the directories in the order of their
// paths from the one after the last surveyed, round after round, each listed
// in its own routine, abandoned when the time left runs out (its result is
// kept when it comes, if no newer one has). A directory the budget leaves out
// is surveyed at the next cycle. The results of the directories the report no
// longer holds are forgotten, and the surveys reach the current report. One
// survey runs at a time: a cycle waits for a survey of the new directories to
// end.
func (a *analytics) Survey(ctx context.Context, now time.Time) {
	report := a.Disks()
	if report == nil {
		return
	}

	a.disks.surveying.Lock()
	defer a.disks.surveying.Unlock()

	deadline := time.Now().Add(a.DiskSettings().Budget())

	a.discover(ctx, report, now, deadline)

	keys, dirs := presentDirs(report)

	a.disks.mu.Lock()
	start, found := slices.BinarySearch(keys, a.disks.survey.last)
	a.disks.mu.Unlock()

	if found {
		start++
	}

	if last := a.surveyRound(ctx, dirs, start, now, deadline); last >= 0 {
		a.setSurveyLast(keys[last])
	}

	a.endCycle(keys, now)
	a.publishSurveys()
}

// SurveyNew surveys, now, the present local directories of the last report
// that were never surveyed, within the budget of the settings, so that a
// directory does not wait for the next cycle when it joins the report: the
// directories of the gateway, given once the service runs, a server or a rule
// added, a directory the discovery found. It reads as well the structuring
// directories that were never read (the home given once the service runs, the
// root of a server added), so that what they hold that no object refers to
// joins the report at the next measure. The directories surveyed already are
// left to the cycles; nothing is done while a survey runs.
func (a *analytics) SurveyNew(ctx context.Context, now time.Time) {
	report := a.Disks()
	if report == nil {
		return
	}

	if !a.disks.surveying.TryLock() {
		return
	}
	defer a.disks.surveying.Unlock()

	deadline := time.Now().Add(a.DiskSettings().Budget())

	a.discoverNew(ctx, report, now, deadline)

	_, dirs := presentDirs(report)

	a.disks.mu.Lock()

	fresh := slices.DeleteFunc(dirs, func(d LocalDir) bool {
		_, kept := a.disks.survey.results[pathKey(d.Path)]

		return kept
	})

	a.disks.mu.Unlock()

	if len(fresh) == 0 {
		return
	}

	a.surveyRound(ctx, fresh, 0, now, deadline)
	a.publishSurveys()
}

// surveyRound lists the directories from the one at start, round after round,
// until the deadline, each in its own routine abandoned when the time left
// runs out, at any depth for a directory no object refers to; the index of
// the last directory it came to, -1 when it came to none.
func (a *analytics) surveyRound(ctx context.Context, dirs []LocalDir, start int, now, deadline time.Time) int {
	maxEntries := a.DiskSettings().SurveyMaxEntries
	last := -1

	for n := range dirs {
		left := time.Until(deadline)
		if left <= 0 || ctx.Err() != nil {
			break
		}

		last = (start + n) % len(dirs)
		dir := &dirs[last]

		if !a.surveyDir(pathKey(dir.Path), dir.Path, maxEntries, dir.Origin == OriginFound, now, left) {
			break
		}
	}

	return last
}

// endCycle forgets the surveys of the directories the report no longer holds
// and dates the cycle.
func (a *analytics) endCycle(keys []string, now time.Time) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	for key := range a.disks.survey.results {
		if _, kept := slices.BinarySearch(keys, key); !kept {
			delete(a.disks.survey.results, key)
		}
	}

	a.disks.survey.at = now
}

// publishSurveys gives the current report the surveys kept so far, in a fresh
// copy, the report handed out being read by others; retried on the report of
// a measure that lands meanwhile, which may have attached them before the
// last one came.
func (a *analytics) publishSurveys() {
	for {
		current := a.disks.report.Load()
		if current == nil {
			return
		}

		fresh := current.clone()
		a.attachSurveys(fresh)

		if a.disks.report.CompareAndSwap(current, fresh) {
			return
		}
	}
}

// presentDirs lists the present local directories of a report, by the key of
// their path, in that order.
func presentDirs(report *DiskReport) (keys []string, dirs []LocalDir) {
	for _, v := range report.Volumes {
		for _, d := range v.Dirs {
			if d.Presence == PresencePresent {
				dirs = append(dirs, d)
			}
		}
	}

	slices.SortFunc(dirs, func(x, y LocalDir) int { return strings.Compare(pathKey(x.Path), pathKey(y.Path)) })

	keys = make([]string, len(dirs))
	for i, d := range dirs {
		keys[i] = pathKey(d.Path)
	}

	return keys, dirs
}

// setSurveyLast remembers the directory the cycle is at, where the next one
// resumes after.
func (a *analytics) setSurveyLast(key string) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	a.disks.survey.last = key
}

// surveyDir lists one directory in its own routine, at its first level or at
// any depth, and keeps its survey; false when the listing outlasted the time
// left, in which case the routine is abandoned and keeps the survey itself
// when it comes. A directory being listed already is left to that listing.
func (a *analytics) surveyDir(key, path string, maxEntries int, deep bool, now time.Time, left time.Duration) bool {
	if !a.claim(&a.disks.survey.pending, key) {
		return true
	}

	done := make(chan struct{})

	go func() {
		files, size, more, err := a.countFiles(path, maxEntries, deep)

		survey := DirSurvey{Files: files, Size: size, More: more, Deep: deep, At: now}
		if err != nil {
			survey.Error = err.Error()
		}

		a.keepSurvey(key, surveyResult{DirSurvey: survey, err: err})
		close(done)
	}()

	timer := time.NewTimer(left)
	defer timer.Stop()

	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

// claim marks a key as being read in a set; false when it is already, so
// that a reading that outlasts its cycle is left to end, not started again.
func (a *analytics) claim(set *readings, key string) bool {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	return set.add(key)
}

// release marks the end of the reading of a key in a set.
func (a *analytics) release(set *readings, key string) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	set.drop(key)
}

// keepSurvey keeps the survey of a directory, whose listing is over, unless a
// newer one is kept.
func (a *analytics) keepSurvey(key string, res surveyResult) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	a.disks.survey.pending.drop(key)

	if a.disks.survey.results == nil {
		a.disks.survey.results = map[string]surveyResult{}
	}

	if old, kept := a.disks.survey.results[key]; kept && old.At.After(res.At) {
		return
	}

	a.disks.survey.results[key] = res
}

// attachSurveys gives every present directory of a report its last survey,
// and the report the time of the last cycle; a directory the survey may not
// read is reported as denied.
func (a *analytics) attachSurveys(report *DiskReport) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	report.SurveyedAt = a.disks.survey.at

	for i := range report.Volumes {
		dirs := report.Volumes[i].Dirs

		for j := range dirs {
			res, kept := a.disks.survey.results[pathKey(dirs[j].Path)]
			if !kept || dirs[j].Presence != PresencePresent {
				continue
			}

			survey := res.DirSurvey
			dirs[j].Survey = &survey

			if errors.Is(res.err, os.ErrPermission) {
				dirs[j].Presence = PresenceDenied
			}
		}
	}
}

// countFiles counts the files of a directory, at its first level or at any
// depth, with the functions of the system unless the tests gave others.
func (a *analytics) countFiles(path string, maxEntries int, deep bool) (files int, size int64, more bool, err error) {
	a.disks.mu.Lock()
	readDir, walkDir := a.disks.readDir, a.disks.walkDir
	a.disks.mu.Unlock()

	switch {
	case deep && walkDir != nil:
		return walkDir(path, maxEntries)
	case deep:
		return listAllLevels(path, maxEntries)
	case readDir != nil:
		return readDir(path, maxEntries)
	default:
		return listFirstLevel(path, maxEntries)
	}
}

// listFirstLevel reads the first level of a directory, at most maxEntries
// entries of it, and counts the regular files and their sizes; more when
// entries were left unread.
func listFirstLevel(path string, maxEntries int) (files int, size int64, more bool, err error) {
	return listFiles(path, maxEntries, false)
}

// listAllLevels reads a directory and its sub-directories at any depth, at
// most maxEntries entries of the whole walk, and counts the regular files and
// their sizes; more when entries were left unread.
func listAllLevels(path string, maxEntries int) (files int, size int64, more bool, err error) {
	return listFiles(path, maxEntries, true)
}

// listFiles reads a directory, at most maxEntries entries of it (of the whole
// walk when deep), and counts the regular files and their sizes, in its
// sub-directories too when deep, symbolic links not followed; more when
// entries were left unread. An entry gone between the listing and the reading
// of its size, and a sub-directory that may not be read, are left out. The
// error is the refusal of the system on the directory itself.
func listFiles(path string, maxEntries int, deep bool) (files int, size int64, more bool, err error) {
	tally := walker{left: maxEntries, deep: deep}

	capped, err := tally.read(path)
	if err != nil {
		return tally.files, tally.size, false, err
	}

	for !capped && len(tally.todo) > 0 {
		next := tally.todo[len(tally.todo)-1]
		tally.todo = tally.todo[:len(tally.todo)-1]

		if capped, err = tally.read(next); err != nil {
			capped = false // a sub-directory that may not be read is left out
		}
	}

	return tally.files, tally.size, capped, nil
}

// walker counts the regular files of a directory and their sizes within a cap
// on the entries read, down its sub-directories when deep.
type walker struct {
	left  int // the entries that may still be read
	files int
	size  int64
	deep  bool
	todo  []string // the sub-directories left to read
}

// read reads one directory within the entries left, the regular files counted
// and the sub-directories noted when deep; true when the cap was hit.
func (w *walker) read(path string) (capped bool, err error) {
	dir, err := os.Open(path)
	if err != nil {
		return false, err //nolint:wrapcheck // the error of the system, shown as it is
	}

	defer dir.Close() //nolint:errcheck // nothing to do on a failed close of a directory read

	for {
		entries, rErr := dir.ReadDir(min(w.left+1, readDirBatch))

		for _, entry := range entries {
			if w.left == 0 {
				return true, nil
			}

			w.left--

			if mode := entry.Type(); mode.IsRegular() {
				if info, iErr := entry.Info(); iErr == nil {
					w.files++
					w.size += info.Size()
				}
			} else if mode.IsDir() && w.deep { // a symbolic link is neither
				w.todo = append(w.todo, filepath.Join(path, entry.Name()))
			}
		}

		switch {
		case errors.Is(rErr, io.EOF):
			return false, nil
		case rErr != nil:
			return false, rErr //nolint:wrapcheck // the error of the system, shown as it is
		}
	}
}
