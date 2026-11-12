package analytics

import (
	"context"
	"errors"
	"io"
	"os"
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
// the next measure of the volumes.

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

// DirSurvey is the survey of the first level of a directory: the regular
// files it holds and their total size, when it was taken, and whether the cap
// on the entries was hit, in which case the directory holds at least that many
// files, of at least that size. Error is the refusal of the system when the
// directory could not be listed (the count and the size are then zero).
type DirSurvey struct {
	Files int
	Size  int64
	More  bool
	At    time.Time
	Error string
}

// surveyState is what the service keeps of the survey between the cycles,
// under the lock of disks: the result of each directory by its path key, the
// directories being listed, the key of the directory the last cycle stopped
// at, and when the last cycle ran.
type surveyState struct {
	results map[string]surveyResult
	pending map[string]struct{}
	last    string
	at      time.Time
}

// surveyResult is the survey of a directory with the error of the system,
// which says whether the directory may be read.
type surveyResult struct {
	DirSurvey
	err error
}

// Survey surveys, now, the first level of the present local directories of the
// last report, within the budget of the settings: the directories in the
// order of their paths from the one after the last surveyed, round after
// round, each listed in its own routine, abandoned when the time left runs
// out (its result is kept when it comes, if no newer one has). A directory the
// budget leaves out is surveyed at the next cycle. The results of the
// directories the report no longer holds are forgotten, and the surveys reach
// the current report. One survey runs at a time: a cycle waits for a survey of
// the new directories to end.
func (a *analytics) Survey(ctx context.Context, now time.Time) {
	report := a.Disks()
	if report == nil {
		return
	}

	a.disks.surveying.Lock()
	defer a.disks.surveying.Unlock()

	keys, dirs := presentDirs(report)

	a.disks.mu.Lock()
	start, found := slices.BinarySearch(keys, a.disks.survey.last)
	a.disks.mu.Unlock()

	if found {
		start++
	}

	if last := a.surveyRound(ctx, dirs, start, now); last >= 0 {
		a.setSurveyLast(keys[last])
	}

	a.endCycle(keys, now)
	a.publishSurveys()
}

// SurveyNew surveys, now, the present local directories of the last report
// that were never surveyed, within the budget of the settings, so that a
// directory does not wait for the next cycle when it joins the report: the
// directories of the gateway, given once the service runs, or a server or a
// rule added. The directories surveyed already are left to the cycles; nothing
// is done while a survey runs.
func (a *analytics) SurveyNew(ctx context.Context, now time.Time) {
	report := a.Disks()
	if report == nil {
		return
	}

	if !a.disks.surveying.TryLock() {
		return
	}
	defer a.disks.surveying.Unlock()

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

	a.surveyRound(ctx, fresh, 0, now)
	a.publishSurveys()
}

// surveyRound lists the directories from the one at start, round after round,
// within the budget of the settings, each in its own routine abandoned when
// the time left runs out; the index of the last directory it came to, -1 when
// it came to none.
func (a *analytics) surveyRound(ctx context.Context, dirs []LocalDir, start int, now time.Time) int {
	settings := a.DiskSettings()
	budget := settings.Budget()
	started := time.Now()
	last := -1

	for n := range dirs {
		left := budget - time.Since(started)
		if left <= 0 || ctx.Err() != nil {
			break
		}

		last = (start + n) % len(dirs)

		if !a.surveyDir(pathKey(dirs[last].Path), dirs[last].Path, settings.SurveyMaxEntries, now, left) {
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

// surveyDir lists one directory in its own routine and keeps its survey;
// false when the listing outlasted the time left, in which case the routine
// is abandoned and keeps the survey itself when it comes. A directory being
// listed already is left to that listing.
func (a *analytics) surveyDir(key, path string, maxEntries int, now time.Time, left time.Duration) bool {
	if !a.claim(key) {
		return true
	}

	done := make(chan struct{})

	go func() {
		files, size, more, err := a.listDir(path, maxEntries)

		survey := DirSurvey{Files: files, Size: size, More: more, At: now}
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

// claim marks a directory as being listed; false when a listing of it runs
// already, so that a listing that outlasts its cycle is left to end, not
// started again.
func (a *analytics) claim(key string) bool {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	if _, pending := a.disks.survey.pending[key]; pending {
		return false
	}

	if a.disks.survey.pending == nil {
		a.disks.survey.pending = map[string]struct{}{}
	}

	a.disks.survey.pending[key] = struct{}{}

	return true
}

// keepSurvey keeps the survey of a directory, whose listing is over, unless a
// newer one is kept.
func (a *analytics) keepSurvey(key string, res surveyResult) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	delete(a.disks.survey.pending, key)

	if a.disks.survey.results == nil {
		a.disks.survey.results = map[string]surveyResult{}
	}

	if old, kept := a.disks.survey.results[key]; kept && old.At.After(res.At) {
		return
	}

	a.disks.survey.results[key] = res
}

// attachSurveys gives every present directory of a report the last survey of
// its first level, and the report the time of the last cycle; a directory the
// survey may not read is reported as denied.
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

// listDir lists the first level of a directory, with the function of the
// system unless the tests gave another.
func (a *analytics) listDir(path string, maxEntries int) (files int, size int64, more bool, err error) {
	a.disks.mu.Lock()
	readDir := a.disks.readDir
	a.disks.mu.Unlock()

	if readDir == nil {
		readDir = listFirstLevel
	}

	return readDir(path, maxEntries)
}

// listFirstLevel reads the first level of a directory, at most maxEntries
// entries of it, and counts the regular files and their sizes; more when
// entries were left unread. An entry gone between the listing and the reading
// of its size is left out.
func listFirstLevel(path string, maxEntries int) (files int, size int64, more bool, err error) {
	dir, err := os.Open(path)
	if err != nil {
		return 0, 0, false, err //nolint:wrapcheck // the error of the system, shown as it is
	}

	defer dir.Close() //nolint:errcheck // nothing to do on a failed close of a directory read

	for left := maxEntries; ; {
		entries, rErr := dir.ReadDir(min(left+1, readDirBatch))

		for _, entry := range entries {
			if left == 0 {
				return files, size, true, nil
			}

			left--

			if !entry.Type().IsRegular() {
				continue
			}

			if info, iErr := entry.Info(); iErr == nil {
				files++
				size += info.Size()
			}
		}

		switch {
		case errors.Is(rErr, io.EOF):
			return files, size, false, nil
		case rErr != nil:
			return files, size, false, rErr //nolint:wrapcheck // the error of the system, shown as it is
		}
	}
}
