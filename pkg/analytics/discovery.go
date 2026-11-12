package analytics

import (
	"context"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

// The discovery of the directories no object refers to: the survey
// cycle lists the sub-directories of the directories that structure the tree
// of the gateway (its home, every directory between the home and a known
// directory, the root of each server) and keeps those that are neither known
// nor above a known one: the root of a server that was deleted, the local
// directory of a rule that was, a directory made by hand next to the others.
// Each one joins the report at the next measure as a directory of its own,
// with the role RoleUnreferenced and no owner, measured with its volume like
// the others and surveyed at any depth since nothing else looks under it; it
// leaves the report when it is gone from the disk, and takes the role of an
// object that refers to it again. A directory found (or seen by a transfer)
// that is the directory of a rule under the root of a server, where the
// transfers of that server put their files, is reported as the directory of
// that rule (candidates). Nothing is read above the home nor above a root
// declared outside it, so the discovery never wanders over the disk. Nothing
// is ever deleted nor moved: the measure reports, the operator decides.

// errListingTimeout: the listing of a structuring directory did not end
// within the time left; the next cycle comes back to it.
var errListingTimeout = errors.New("the listing did not end in time")

// errListingBusy: a structuring directory is being listed by an earlier
// routine that outlasted its cycle.
var errListingBusy = errors.New("the directory is being listed already")

// foundDir is a directory the discovery found: its path, the key of the
// structuring directory it was found under, and when.
type foundDir struct {
	path   string
	parent string
	at     time.Time
}

// candidates are the directories of the rules under the roots of the servers,
// where the transfers of a server put their files (pkg/pipeline, makeLocalPath:
// the directory of the rule, then the root of the server, then the home of the
// gateway): the local roots of the servers, and the directories of the rules
// relative to any root, by the key of their cleaned path. They exist only once
// a transfer or an operator created them, so they are not declared, which
// would list every server with every rule: a directory found on the disk or
// seen by a transfer that matches one is reported as the directory of its
// rule.
type candidates struct {
	roots []string
	byRel map[string]LocalDir
}

// SetRuleDirs gives the service the local roots of the servers and the
// directories of the rules relative to them (the role, the name and the
// identifier of the rule), so that a directory found or seen under a root is
// reported as the directory of its rule. Of two rules that share a directory,
// the first is kept.
func (a *analytics) SetRuleDirs(roots []string, rules []LocalDir) {
	cand := candidates{byRel: make(map[string]LocalDir, len(rules))}

	for _, root := range roots {
		if root != "" && fs.IsLocalPath(root) {
			cand.roots = append(cand.roots, filepath.Clean(root))
		}
	}

	for i := range rules {
		rule := &rules[i]
		if rule.Path == "" || fs.IsAbsPath(rule.Path) || !fs.IsLocalPath(rule.Path) {
			continue
		}

		key := pathKey(filepath.Clean(rule.Path))
		if _, taken := cand.byRel[key]; !taken {
			cand.byRel[key] = *rule
		}
	}

	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	a.disks.rules = cand
}

// ruleDirs is the candidates in force.
func (a *analytics) ruleDirs() candidates {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	return a.disks.rules
}

// classify gives a directory found on the disk or seen by a transfer the role
// and the owner of the rule whose directory it is under the root of a server,
// when one is; a declared directory is left as it is.
func (c *candidates) classify(dir *LocalDir) {
	if dir.Origin != OriginSeen && dir.Origin != OriginFound {
		return
	}

	for _, root := range c.roots {
		if len(dir.Path) <= len(root) || !under(dir.Path, root) {
			continue
		}

		rel := strings.TrimLeft(dir.Path[len(root):], `/\`)

		if rule, ok := c.byRel[pathKey(filepath.Clean(rel))]; ok {
			dir.Role, dir.Owner, dir.OwnerID = rule.Role, rule.Owner, rule.OwnerID

			return
		}
	}
}

// foundDirs lists the directories found, as entries of the measure.
func (a *analytics) foundDirs() []LocalDir {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	out := make([]LocalDir, 0, len(a.disks.survey.found))
	for _, f := range a.disks.survey.found {
		out = append(out, LocalDir{Path: f.path, Role: RoleUnreferenced, Origin: OriginFound})
	}

	slices.SortFunc(out, func(x, y LocalDir) int { return strings.Compare(x.Path, y.Path) })

	return out
}

// discover lists, until the deadline, the sub-directories of the structuring
// directories of the report and keeps those no object refers to; what was
// found under a directory that no longer structures the tree is forgotten.
func (a *analytics) discover(ctx context.Context, report *DiskReport, now, deadline time.Time) {
	known := knownDirs(report)
	roots := structuringDirs(a.Home(), known)

	a.discoverRoots(ctx, roots, known, now, deadline)
	a.pruneFound(roots)
}

// discoverNew lists, until the deadline, the structuring directories of the
// report that were never read: the home of the gateway, given once the service
// runs, or the root of a server added; so that what they hold does not wait
// for the next cycle.
func (a *analytics) discoverNew(ctx context.Context, report *DiskReport, now, deadline time.Time) {
	known := knownDirs(report)
	roots := structuringDirs(a.Home(), known)

	a.disks.mu.Lock()

	fresh := slices.DeleteFunc(roots, func(root string) bool {
		_, explored := a.disks.survey.explored[pathKey(root)]

		return explored
	})

	a.disks.mu.Unlock()

	a.discoverRoots(ctx, fresh, known, now, deadline)
}

// discoverRoots lists, until the deadline, the sub-directories of structuring
// directories and keeps those no object refers to. A directory the time does
// not reach, or that cannot be read, keeps what was found under it before.
func (a *analytics) discoverRoots(ctx context.Context, roots []string, known []LocalDir, now, deadline time.Time) {
	maxEntries := a.DiskSettings().SurveyMaxEntries

	for _, root := range roots {
		left := time.Until(deadline)
		if left <= 0 || ctx.Err() != nil {
			break
		}

		names, err := a.listSubdirs(root, maxEntries, left)
		if errors.Is(err, errListingTimeout) {
			break
		}

		if err != nil {
			continue
		}

		a.keepFound(pathKey(root), unreferenced(root, names, known), now)
	}
}

// knownDirs lists the local directories of a report that an object refers to
// or that a transfer used: every one but those the discovery found.
func knownDirs(report *DiskReport) []LocalDir {
	var out []LocalDir

	for _, v := range report.Volumes {
		for _, d := range v.Dirs {
			if d.Origin != OriginFound {
				out = append(out, d)
			}
		}
	}

	for i := range report.Unmeasured {
		if d := report.Unmeasured[i].LocalDir; d.Origin != OriginFound && fs.IsLocalPath(d.Path) {
			out = append(out, d)
		}
	}

	return out
}

// structuringDirs lists the directories whose sub-directories the discovery
// reads: the home of the gateway, every directory between it and a known
// directory, and the root of each server; nothing above the home nor above a
// root declared outside it. In the order of their keys.
func structuringDirs(home string, known []LocalDir) []string {
	byKey := map[string]string{}
	add := func(path string) { byKey[pathKey(path)] = path }

	if home != "" {
		home = filepath.Clean(home)
		add(home)
	}

	for i := range known {
		dir := &known[i]

		if dir.Role == RoleServerRoot {
			add(dir.Path)
		}

		if home == "" || !under(dir.Path, home) {
			continue
		}

		for p := filepath.Dir(dir.Path); under(p, home) && pathKey(p) != pathKey(home); p = filepath.Dir(p) {
			add(p)
		}
	}

	keys := slices.Sorted(maps.Keys(byKey))

	out := make([]string, len(keys))
	for i, key := range keys {
		out[i] = byKey[key]
	}

	return out
}

// unreferenced keeps, of the sub-directories of a structuring directory,
// those that are neither known nor above a known directory.
func unreferenced(parent string, names []string, known []LocalDir) []string {
	keys := make([]string, len(known))
	for i := range known {
		keys[i] = pathKey(known[i].Path)
	}

	slices.Sort(keys)

	var out []string

	for _, name := range names {
		child := filepath.Join(parent, name)

		if _, isKnown := slices.BinarySearch(keys, pathKey(child)); isKnown {
			continue
		}

		if slices.ContainsFunc(known, func(d LocalDir) bool { return under(d.Path, child) }) {
			continue // above a known directory
		}

		out = append(out, child)
	}

	return out
}

// listSubdirs reads the sub-directories of a structuring directory in its own
// routine, abandoned past the time left (errListingTimeout); a directory an
// earlier routine is still reading is left to it (errListingBusy).
func (a *analytics) listSubdirs(path string, maxEntries int, left time.Duration) ([]string, error) {
	key := pathKey(path)
	if !a.claim(&a.disks.survey.listing, key) {
		return nil, errListingBusy
	}

	type listing struct {
		names []string
		err   error
	}

	done := make(chan listing, 1)

	go func() {
		names, err := a.readSubdirs(path, maxEntries)
		a.release(&a.disks.survey.listing, key)
		done <- listing{names: names, err: err}
	}()

	timer := time.NewTimer(left)
	defer timer.Stop()

	select {
	case res := <-done:
		return res.names, res.err
	case <-timer.C:
		return nil, errListingTimeout
	}
}

// readSubdirs reads the names of the sub-directories of a directory, with
// the function of the system unless the tests gave another.
func (a *analytics) readSubdirs(path string, maxEntries int) ([]string, error) {
	a.disks.mu.Lock()
	subdirs := a.disks.subdirs
	a.disks.mu.Unlock()

	if subdirs == nil {
		subdirs = listSubdirNames
	}

	return subdirs(path, maxEntries)
}

// listSubdirNames reads the names of the sub-directories at the first level
// of a directory, at most maxEntries entries read, symbolic links left out.
func listSubdirNames(path string, maxEntries int) ([]string, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error of the system
	}

	defer dir.Close() //nolint:errcheck // nothing to do on a failed close of a directory read

	var names []string

	for left := maxEntries; left > 0; {
		entries, rErr := dir.ReadDir(min(left, readDirBatch))

		for _, entry := range entries {
			left--

			if entry.IsDir() { // a symbolic link to a directory is not
				names = append(names, entry.Name())
			}
		}

		switch {
		case errors.Is(rErr, io.EOF):
			return names, nil
		case rErr != nil:
			return names, rErr //nolint:wrapcheck // the error of the system
		}
	}

	return names, nil
}

// keepFound replaces what was found under a structuring directory, read now.
func (a *analytics) keepFound(parent string, paths []string, now time.Time) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	if a.disks.survey.found == nil {
		a.disks.survey.found = map[string]foundDir{}
		a.disks.survey.explored = map[string]time.Time{}
	}

	a.disks.survey.explored[parent] = now

	maps.DeleteFunc(a.disks.survey.found, func(_ string, f foundDir) bool { return f.parent == parent })

	for _, path := range paths {
		a.disks.survey.found[pathKey(path)] = foundDir{path: path, parent: parent, at: now}
	}
}

// pruneFound forgets the directories that no longer structure the tree (the
// root of a server that was deleted, which the discovery now finds itself)
// and what was found under them.
func (a *analytics) pruneFound(roots []string) {
	keys := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		keys[pathKey(root)] = struct{}{}
	}

	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	maps.DeleteFunc(a.disks.survey.explored, func(key string, _ time.Time) bool {
		_, kept := keys[key]

		return !kept
	})
	maps.DeleteFunc(a.disks.survey.found, func(_ string, f foundDir) bool {
		_, kept := keys[f.parent]

		return !kept
	})
}

// dropGone takes out of a report the directories the discovery found that
// the measure found gone from the disk, and forgets them; a volume left
// without a directory goes with them.
func (a *analytics) dropGone(report *DiskReport) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	gone := func(d LocalDir) bool {
		if d.Origin != OriginFound || (d.Presence != PresenceToCreate && d.Presence != PresenceMissing) {
			return false
		}

		delete(a.disks.survey.found, pathKey(d.Path))

		return true
	}

	for i := range report.Volumes {
		report.Volumes[i].Dirs = slices.DeleteFunc(report.Volumes[i].Dirs, gone)
	}

	report.Volumes = slices.DeleteFunc(report.Volumes, func(v Volume) bool { return len(v.Dirs) == 0 })

	kept := report.Unmeasured[:0]

	for i := range report.Unmeasured {
		if !gone(report.Unmeasured[i].LocalDir) {
			kept = append(kept, report.Unmeasured[i])
		}
	}

	report.Unmeasured = kept
}
