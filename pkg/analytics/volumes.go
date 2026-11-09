package analytics

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/disk"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// The disk space the transfers use: the local directories the gateway
// reads from or writes to, from three sources (configured, destinations of
// the tasks, seen by the transfers), reduced to the volumes that hold them
// and measured in the background with a time limit, so that a mount that no
// longer answers blocks neither the web interface nor SNMP.

const (
	// DiskPeriod is how often the service measures the volumes.
	DiskPeriod = 30 * time.Second
	// DiskTimeout is how long one volume may take to answer before it is
	// reported as not responding.
	DiskTimeout = 2 * time.Second
	// SeenWindow is how long a directory a transfer used stays in the list,
	// and how far the service looks back when it starts.
	SeenWindow = 7 * 24 * time.Hour

	// DirLog is the directory of the log file, when the gateway logs to a file.
	DirLog = "log"

	variableMark = "#" // what starts a variable in a task argument

	// The arguments several tasks name their destination by.
	argPath       = "path"
	argOutputFile = "outputFile"
)

// The thresholds of a volume: a warning at 80 % used or under 10 GB free,
// critical at 90 % used or under 2 GB free, the first reached; the same
// shares for the inodes.
const (
	DiskWarnPercent     = 80.0
	DiskCriticalPercent = 90.0
	DiskWarnFree        = 10 << 30
	DiskCriticalFree    = 2 << 30
)

// The roles of the directories read from the configuration of the objects
// and from the transfers (the gateway's own keep their name: in, out, tmp,
// database, log).
const (
	RoleServerRoot    = "server_root"
	RoleServerReceive = "server_receive"
	RoleServerSend    = "server_send"
	RoleServerTmp     = "server_tmp"
	RoleRuleLocal     = "rule_local"
	RoleRuleTmp       = "rule_tmp"
	RoleTask          = "task"
	RoleTransfer      = "transfer"
)

// DirOrigin is where the service learnt a directory from.
type DirOrigin int

const (
	// OriginConfigured: the configuration file, a server or a rule.
	OriginConfigured DirOrigin = iota
	// OriginTask: the fixed part of the destination of a task of a rule.
	OriginTask
	// OriginSeen: the local path of a transfer, when it started.
	OriginSeen
)

// LocalDir is a local directory the transfers use: its path, its role (a
// constant above, named by the catalogue), the object it belongs to (the
// name of the server or of the rule, the type of the task; "" for the
// gateway's own directories), its origin and, for a directory seen by a
// transfer, when the last one used it.
type LocalDir struct {
	Path   string
	Role   string
	Owner  string
	Origin DirOrigin
	SeenAt time.Time
}

// DiskState is the state of a volume against the thresholds.
type DiskState int

const (
	DiskOK DiskState = iota
	DiskWarning
	DiskCritical
)

// Volume is a measured volume: the mount point (or the drive) that holds its
// directories, the directories, the usage and the state.
type Volume struct {
	Key           string
	Dirs          []LocalDir
	UsedPercent   float64
	Free, Total   uint64
	InodesPercent float64 // 0 when the system does not say (Windows)
	InodesKnown   bool
	State         DiskState
	MeasuredAt    time.Time
}

// DiskReason is why a directory could not be measured.
type DiskReason int

const (
	// ReasonCloud: the directory is on a cloud storage, which has no volume.
	ReasonCloud DiskReason = iota + 1
	// ReasonAbsent: the directory does not exist and nothing above it does
	// (a drive that is gone); a missing directory under an existing one is
	// measured through its volume, the gateway creates it at the first
	// transfer.
	ReasonAbsent
	// ReasonDenied: the process may not read it.
	ReasonDenied
	// ReasonTimeout: the volume did not answer within DiskTimeout.
	ReasonTimeout
	// ReasonError: the system refused the measure for another reason (Error).
	ReasonError
)

// UnmeasuredDir is a directory whose volume could not be measured, with the
// reason; a cloud directory is one, by nature.
type UnmeasuredDir struct {
	LocalDir
	Reason DiskReason
	Error  string
}

// Local says whether the directory should have been measured: every reason
// but the cloud storage is an incident of production.
func (u UnmeasuredDir) Local() bool { return u.Reason != ReasonCloud }

// DiskReport is the last measure of the volumes: the volumes, the most severe
// first then the fullest, the directories that could not be measured, and
// when the measure was taken.
type DiskReport struct {
	Volumes    []Volume
	Unmeasured []UnmeasuredDir
	MeasuredAt time.Time
}

// Worst is the most severe volume of the report, the fullest among equals;
// false without a measured volume.
func (r *DiskReport) Worst() (Volume, bool) {
	if r == nil || len(r.Volumes) == 0 {
		return Volume{}, false
	}

	return r.Volumes[0], true
}

// Usage is the usage of the volume that holds a directory of the given role,
// as the former per-directory objects say it; false when no volume holds one.
func (r *DiskReport) Usage(role string) (DiskUsage, bool) {
	if r == nil {
		return DiskUsage{}, false
	}

	for _, v := range r.Volumes {
		for _, d := range v.Dirs {
			if d.Role == role {
				return DiskUsage{Name: d.Role, Path: d.Path, UsedPercent: v.UsedPercent, Free: v.Free}, true
			}
		}
	}

	return DiskUsage{}, false
}

// stateOf is the state of a measure against the thresholds: the share used
// and the bytes free, then the share of the inodes.
func stateOf(usage *disk.UsageStat) DiskState {
	switch {
	case usage.UsedPercent >= DiskCriticalPercent, usage.Free < DiskCriticalFree,
		usage.InodesTotal > 0 && usage.InodesUsedPercent >= DiskCriticalPercent:
		return DiskCritical
	case usage.UsedPercent >= DiskWarnPercent, usage.Free < DiskWarnFree,
		usage.InodesTotal > 0 && usage.InodesUsedPercent >= DiskWarnPercent:
		return DiskWarning
	default:
		return DiskOK
	}
}

// disks is the part of the service that knows the volumes: the home of the
// gateway (what a relative directory is resolved against), the directories
// seen by the transfers, the last report, and the functions that read the
// system, which the tests replace.
type disks struct {
	mu      sync.Mutex
	home    string
	seen    map[string]time.Time
	report  atomic.Pointer[DiskReport]
	usage   func(path string) (*disk.UsageStat, error)
	mounts  func() ([]string, error)
	timeout time.Duration
}

// SetHome tells the service the home directory of the gateway, which a
// relative directory of a server, of a rule or of a task is resolved against.
func (a *analytics) SetHome(home string) {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	a.disks.home = home
}

// Seen records that a transfer uses a local path, now: its directory joins
// the volumes measured for SeenWindow.
func (a *analytics) Seen(localPath string, at time.Time) {
	if localPath == "" || !fs.IsLocalPath(localPath) {
		return
	}

	dir := fs.Dir(localPath)
	if dir == "" {
		return
	}

	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	if a.disks.seen == nil {
		a.disks.seen = map[string]time.Time{}
	}

	if at.After(a.disks.seen[dir]) {
		a.disks.seen[dir] = at
	}
}

// SeenLocalPath records the local path of a transfer that starts on the
// gateway.
func SeenLocalPath(localPath string) {
	if GlobalService != nil {
		GlobalService.Seen(localPath, time.Now())
	}
}

// seenDirs lists the directories seen within the window before now, the
// older ones forgotten.
func (a *analytics) seenDirs(now time.Time) []LocalDir {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	var out []LocalDir

	for dir, at := range a.disks.seen {
		if now.Sub(at) > SeenWindow {
			delete(a.disks.seen, dir)

			continue
		}

		out = append(out, LocalDir{Path: dir, Role: RoleTransfer, Origin: OriginSeen, SeenAt: at})
	}

	slices.SortFunc(out, func(x, y LocalDir) int { return strings.Compare(x.Path, y.Path) })

	return out
}

// Disks is the last report of the volumes, nil before the first measure.
func (a *analytics) Disks() *DiskReport { return a.disks.report.Load() }

// Home is the home directory of the gateway given to the service.
func (a *analytics) Home() string {
	a.disks.mu.Lock()
	defer a.disks.mu.Unlock()

	return a.disks.home
}

// Resolve gives the absolute path of a directory of an object: an absolute
// one as it is, a relative one under its parents in order (the root of its
// server, then the home of the gateway), as the transfers resolve theirs.
func (a *analytics) Resolve(dir string, parents ...string) string {
	if dir == "" {
		return ""
	}

	elems := make([]utils.Elem, 0, len(parents)+1)
	for _, parent := range parents {
		elems = append(elems, utils.Branch(parent))
	}

	elems = append(elems, utils.Branch(a.Home()))

	resolved, err := utils.GetPath(dir, elems...)
	if err != nil {
		return dir
	}

	return resolved
}

// TaskDestinations names, by task type, the argument that holds a local
// destination (a path the task writes to), and whether it names a directory
// (a file otherwise, whose directory is measured). The arguments of the
// scripts (EXEC, EXECMOVE, EXECOUTPUT) are out of reach. A test of the tasks
// package checks the table against the tasks.
//
//nolint:gochecknoglobals // reference data of the tasks
var TaskDestinations = map[string]struct {
	Arg string
	Dir bool
}{
	"COPY": {argPath, true}, "COPYRENAME": {argPath, false}, "MOVE": {argPath, true}, "MOVERENAME": {argPath, false},
	"RENAME": {argPath, false}, "ARCHIVE": {"outputPath", false}, "EXTRACT": {"outputDir", true},
	"ENCRYPT": {argOutputFile, false}, "DECRYPT": {argOutputFile, false}, "SIGN": {argOutputFile, false},
	"ENCRYPT&SIGN": {argOutputFile, false}, "DECRYPT&VERIFY": {argOutputFile, false},
	"TRANSFER": {"file", false}, "PREREGISTER": {"file", false},
}

// DestinationDir is the directory a task destination names before its first
// variable (#…#): the fixed part of a directory argument, the directory of a
// file argument; "" when nothing fixed is left, and whether the destination
// is a cloud path, which has no volume.
func DestinationDir(arg string, isDir bool) (dir string, cloud bool) {
	fixed, cut := arg, false
	if before, _, ok := strings.Cut(arg, variableMark); ok {
		fixed, cut = before, true
	}

	if fixed == "" {
		return "", false
	}

	if !fs.IsLocalPath(fixed) {
		return fixed, true
	}

	switch {
	case cut && !strings.HasSuffix(fixed, "/") && !strings.HasSuffix(fixed, `\`):
		// The variable started inside a name: the directory above it.
		dir = fs.Dir(fixed)
	case !isDir && !cut:
		dir = fs.Dir(fixed)
	default:
		dir = strings.TrimRight(fixed, `/\`)
		if dir == "" || strings.HasSuffix(dir, ":") {
			dir = fixed
		}
	}

	return filepath.Clean(dir), false // written as the system writes its paths
}

// Measure measures the volumes of the directories, now: the directories
// grouped by the volume that holds them, each volume read within the time
// limit in its own routine, and keeps the report.
func (a *analytics) Measure(ctx context.Context, dirs []LocalDir, now time.Time) *DiskReport {
	report := &DiskReport{MeasuredAt: now}
	groups := map[string][]LocalDir{}

	var keys []string

	mounts := a.mountPoints()

	mounted := map[string]bool{}

	for _, d := range dirs {
		if !fs.IsLocalPath(d.Path) {
			report.Unmeasured = append(report.Unmeasured, UnmeasuredDir{LocalDir: d, Reason: ReasonCloud})

			continue
		}

		key, isMount := volumeKey(d.Path, mounts)
		if _, seen := groups[key]; !seen {
			keys = append(keys, key)
			mounted[key] = isMount
		}

		groups[key] = append(groups[key], d)
	}

	results := make([]measure, len(keys))

	var wg sync.WaitGroup

	for i, key := range keys {
		wg.Go(func() {
			results[i] = a.measureVolume(ctx, key, mounted[key], groups[key])
		})
	}

	wg.Wait()

	for i, key := range keys {
		res := results[i]

		var measured []LocalDir

		for j, d := range groups[key] {
			if err, bad := res.bad[j]; bad {
				report.Unmeasured = append(report.Unmeasured, UnmeasuredDir{LocalDir: d, Reason: reasonOf(err), Error: err.Error()})

				continue
			}

			measured = append(measured, d)
		}

		switch {
		case len(measured) == 0:
			// Every directory of the volume refused: nothing to measure.
		case res.err != nil:
			for _, d := range measured {
				report.Unmeasured = append(report.Unmeasured,
					UnmeasuredDir{LocalDir: d, Reason: res.reason, Error: res.err.Error()})
			}
		default:
			report.Volumes = append(report.Volumes, Volume{
				Key: key, Dirs: measured, UsedPercent: res.usage.UsedPercent, Free: res.usage.Free, Total: res.usage.Total,
				InodesPercent: res.usage.InodesUsedPercent, InodesKnown: res.usage.InodesTotal > 0,
				State: stateOf(res.usage), MeasuredAt: now,
			})
		}
	}

	slices.SortFunc(report.Volumes, func(x, y Volume) int {
		if x.State != y.State {
			return int(y.State) - int(x.State)
		}

		if x.UsedPercent != y.UsedPercent {
			if x.UsedPercent > y.UsedPercent {
				return -1
			}

			return 1
		}

		return strings.Compare(x.Key, y.Key)
	})
	slices.SortFunc(report.Unmeasured, func(x, y UnmeasuredDir) int { return strings.Compare(x.Path, y.Path) })

	a.disks.report.Store(report)

	return report
}

// measure is the reading of one volume: its usage, or the refusal of the
// volume (err, reason); bad holds, by their index, the directories that
// could not be looked up (access denied, nothing above them).
type measure struct {
	usage  *disk.UsageStat
	err    error
	reason DiskReason
	bad    map[int]error
}

// measureVolume reads one volume within the time limit. The volume is read
// on its mount point when the system names one, else on the first directory
// that exists or on the nearest ancestor of a missing one: a directory the
// transfers have not used yet does not exist until the gateway creates it,
// and what matters is the room of the volume that will hold it. A directory
// the process may not look up is reported, as a directory with nothing
// above it. The reading runs in its own routine (a statfs on a mount that
// no longer answers blocks, and the context cannot interrupt it), which is
// abandoned past the limit.
func (a *analytics) measureVolume(ctx context.Context, key string, mounted bool, dirs []LocalDir) measure {
	done := make(chan measure, 1)

	go func() {
		res := measure{bad: map[int]error{}}
		probe := ""

		if mounted {
			probe = key
		}

		for i, d := range dirs {
			existing, err := existingAncestor(d.Path)
			if err != nil {
				res.bad[i] = err

				continue
			}

			if probe == "" {
				probe = existing
			}
		}

		if probe != "" {
			res.usage, res.err = a.usageOf(probe)
			res.reason = reasonOf(res.err)
		}

		done <- res
	}()

	timeout := a.disks.timeout
	if timeout <= 0 {
		timeout = DiskTimeout
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-done:
		return res
	case <-timer.C:
		return measure{err: errDiskTimeout, reason: ReasonTimeout}
	case <-ctx.Done():
		return measure{err: ctx.Err(), reason: ReasonTimeout}
	}
}

var errDiskTimeout = errors.New("the volume did not answer in time")

// existingAncestor is the path itself when it exists, else the nearest
// directory above it that does; the error of the system when the path may
// not be looked up, or when nothing above it exists (a drive that is gone).
func existingAncestor(path string) (string, error) {
	for p := path; ; {
		_, err := os.Stat(p)

		switch {
		case err == nil:
			return p, nil
		case !errors.Is(err, os.ErrNotExist):
			return "", err //nolint:wrapcheck // the error of the system, read by reasonOf
		}

		parent := filepath.Dir(p)
		if parent == p {
			return "", err //nolint:wrapcheck // the error of the system, read by reasonOf
		}

		p = parent
	}
}

// reasonOf reads the reason of a refused measure in the error of the system.
func reasonOf(err error) DiskReason {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, os.ErrNotExist):
		return ReasonAbsent
	case errors.Is(err, os.ErrPermission):
		return ReasonDenied
	default:
		return ReasonError
	}
}

// usageOf reads the file system of a path, with the function of the system
// unless the tests gave another.
func (a *analytics) usageOf(path string) (*disk.UsageStat, error) {
	a.disks.mu.Lock()
	usage := a.disks.usage
	a.disks.mu.Unlock()

	if usage == nil {
		usage = disk.Usage
	}

	return usage(path) //nolint:wrapcheck // the error of the system, read by reasonOf
}

// mountPoints lists the mount points of the system (the drives on Windows),
// the longest first; nothing when the system does not say.
func (a *analytics) mountPoints() []string {
	a.disks.mu.Lock()
	mounts := a.disks.mounts
	a.disks.mu.Unlock()

	if mounts == nil {
		mounts = systemMountPoints
	}

	list, err := mounts()
	if err != nil {
		return nil
	}

	slices.SortFunc(list, func(x, y string) int { return len(y) - len(x) })

	return list
}

func systemMountPoints() ([]string, error) {
	parts, err := disk.Partitions(true)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error of the system
	}

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p.Mountpoint)
	}

	return out, nil
}

// volumeKey is the volume that holds a directory: the longest mount point
// the directory lies under, when the system names one (mounted), else its
// drive (Windows), else the directory itself.
func volumeKey(dir string, mounts []string) (key string, mounted bool) {
	clean, err := filepath.Abs(dir) // the drive made explicit on Windows
	if err != nil {
		clean = filepath.Clean(dir)
	}

	for _, mount := range mounts {
		if under(clean, mount) {
			return mount, true
		}
	}

	if volume := filepath.VolumeName(clean); volume != "" {
		return volume + string(filepath.Separator), false
	}

	return clean, false
}

// under says whether a clean path lies under a mount point (or is it).
func under(path, mount string) bool {
	if mount == "" {
		return false
	}

	p, m := path, filepath.Clean(mount)
	if runtime.GOOS == "windows" {
		p, m = strings.ToLower(p), strings.ToLower(m)
	}

	root := m
	m = strings.TrimRight(m, `/\`)

	if m == "" { // the root of the system
		return strings.HasPrefix(p, root)
	}

	return p == m || strings.HasPrefix(p, m+"/") || strings.HasPrefix(p, m+`\`)
}
