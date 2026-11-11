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
// stopped at, so that a directory left out is surveyed at the next cycle. The
// results reach the report at the next measure of the volumes.

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
// key of the directory the last cycle stopped at, and when the last cycle
// ran.
type surveyState struct {
	results map[string]surveyResult
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
// directories the report no longer holds are forgotten.
func (a *analytics) Survey(ctx context.Context, now time.Time) {
	report := a.Disks()
	if report == nil {
		return
	}

	keys, dirs := presentDirs(report)
	settings := a.DiskSettings()
	budget := settings.Budget()
	started := time.Now()

	a.disks.mu.Lock()
	start, found := slices.BinarySearch(keys, a.disks.survey.last)
	a.disks.mu.Unlock()

	if found {
		start++
	}

	for n := range dirs {
		left := budget - time.Since(started)
		if left <= 0 || ctx.Err() != nil {
			break
		}

		i := (start + n) % len(dirs)
		a.setSurveyLast(keys[i])

		if !a.surveyDir(keys[i], dirs[i].Path, settings.SurveyMaxEntries, now, left) {
			break
		}
	}

	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	for key := range a.disks.survey.results {
		if _, kept := slices.BinarySearch(keys, key); !kept {
			delete(a.disks.survey.results, key)
		}
	}

	a.disks.survey.at = now
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
// is abandoned and keeps the survey itself when it comes.
func (a *analytics) surveyDir(key, path string, maxEntries int, now time.Time, left time.Duration) bool {
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

// keepSurvey keeps the survey of a directory, unless a newer one is kept.
func (a *analytics) keepSurvey(key string, res surveyResult) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

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
