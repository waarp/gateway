package dashboard

import (
	"math"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/format"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/transfers"
	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
)

// The thresholds of the strip (screen sheet of the second round, 2026-11-03):
// a disk in warning at 80 % used, in failure at 90 %; the processor and the
// memory in warning at 80 % of what the machine has.
const (
	healthKeys = "dashboard.health."
	diskWarn   = 80.0
	diskKO     = 90.0
	loadWarn   = 0.8
	hoursADay  = 24
	percent    = 100
)

// healthInput is what the strip of the node is built from.
type healthInput struct {
	node     Node
	now      time.Time
	dbType   string
	last     reading // the last reading of the figures: how long it took, since when it fails
	services services.Summary
	certs    *certSummary // the certificates of the node; nil when the figures are not read
	monitors Monitors
}

// healthCells builds the cells of the strip (#834): uptime, services, database,
// processor and memory, disk space, network, certificates (#836), partner
// monitors. Each cell carries its state by a shape and a word; a figure that
// cannot be read is "unmeasured", never a false figure.
func healthCells(translate gui.Translator, in *healthInput) []gui.HealthItem {
	txt := texts{translate}

	return []gui.HealthItem{
		uptimeCell(txt, in.node, in.now),
		servicesCell(txt, in.services),
		databaseCell(txt, in.dbType, in.last),
		loadCell(txt, in.node),
		diskCell(txt, in.node),
		networkCell(txt, in.node),
		certsCell(txt, in.certs),
		monitorsCell(txt, in.monitors),
	}
}

// certsCell counts the certificates of the node about to expire (#836): the
// expired ones first, then those ending within thirty days, named with their
// date in the tooltip; "all valid" otherwise.
func certsCell(txt texts, certs *certSummary) gui.HealthItem {
	cell := gui.HealthItem{Label: txt.health("certs")}

	if certs == nil || certs.Unread {
		cell.Value, cell.State, cell.Text = txt.health("unmeasured"), stateOff, txt.health("certs_unread")

		return cell
	}

	items := make([]string, 0, len(certs.Items))
	for _, item := range certs.Items {
		_, date := transfers.TimeAndDate(txt.translate, item.NotAfter)
		items = append(items, txt.health("cert_item", "owner", txt.health("cert_"+item.Owner, "name", item.Name), "date", date))
	}

	cell.Title = strings.Join(items, " · ")

	switch {
	case certs.Total == 0:
		cell.Value, cell.State, cell.Text = "0", stateOff, txt.health("certs_none")
	case certs.Expired > 0:
		cell.Value, cell.Unit = strconv.Itoa(certs.Expired+certs.Expiring), "/ "+strconv.Itoa(certs.Total)
		cell.State, cell.Text = stateKO, txt.count(healthKeys, "certs_expired", uint64(certs.Expired)) //nolint:gosec // a count
	case certs.Expiring > 0:
		cell.Value, cell.Unit = strconv.Itoa(certs.Expiring), "/ "+strconv.Itoa(certs.Total)
		cell.State, cell.Text = stateWarn, txt.count(healthKeys, "certs_expiring", uint64(certs.Expiring)) //nolint:gosec // a count
	default:
		cell.Value, cell.Unit = strconv.Itoa(certs.Total), ""
		cell.State, cell.Text = stateOK, txt.health("certs_ok")
	}

	return cell
}

func (txt texts) health(key string, args ...any) string {
	return txt.translate(healthKeys+key, args...)
}

func uptimeCell(txt texts, node Node, now time.Time) gui.HealthItem {
	cell := gui.HealthItem{Label: txt.health("uptime"), Href: common.WebuiURL("/about")}

	started, ok := node.StartTime()
	if !ok {
		cell.Value, cell.State = txt.health("unmeasured"), stateOff
		cell.Text = txt.health("uptime_unknown", "version", node.Version())

		return cell
	}

	clock, date := transfers.TimeAndDate(txt.translate, started)
	cell.Value, cell.State = daysHours(txt, now.Sub(started)), stateOK
	cell.Text = txt.health("uptime_text", "version", node.Version(), "date", date)
	cell.Title = txt.health("uptime_title", "version", node.Version(), "date", date, "time", clock)

	return cell
}

// daysHours writes a duration in days and hours ("12 j 04 h"), in hours and
// minutes under a day.
func daysHours(txt texts, d time.Duration) string {
	hours := int(d.Hours())
	if hours < hoursADay {
		return txt.health("hours_minutes", "hours", hours, "minutes", pad(int(d.Minutes())%60)) //nolint:mnd // minutes
	}

	return txt.health("days_hours", "days", hours/hoursADay, "hours", pad(hours%hoursADay))
}

// pad writes a figure on two digits ("04").
func pad(n int) string {
	if n < 10 { //nolint:mnd // one digit
		return "0" + strconv.Itoa(n)
	}

	return strconv.Itoa(n)
}

func servicesCell(txt texts, sum services.Summary) gui.HealthItem {
	cell := gui.HealthItem{
		Label: txt.health("services"), Href: common.WebuiURL("/services"),
		Value: strconv.Itoa(sum.Running), Unit: "/ " + strconv.Itoa(sum.Total),
	}

	groups := make([]string, 0, len(sum.Groups))
	for _, g := range sum.Groups {
		groups = append(groups, txt.health("group_running", "group", g.Label, "running", g.Running, "total", g.Total))
	}

	cell.Title = strings.Join(groups, " · ")

	switch {
	case sum.Total == 0:
		cell.State, cell.Text = stateOff, txt.translate("state.none_yet")
	case sum.Failed > 0:
		cell.State, cell.Text = stateKO, txt.translate("services.summary.errors", "n", sum.Failed)
	case sum.Offline+sum.Unknown > 0:
		cell.State, cell.Text = stateOff, txt.translate("services.summary.offline", "n", sum.Offline+sum.Unknown)
	default:
		cell.State, cell.Text = stateOK, txt.translate("state.all_running")
	}

	return cell
}

func databaseCell(txt texts, dbType string, last reading) gui.HealthItem {
	cell := gui.HealthItem{Label: txt.health("database"), Href: common.WebuiURL("/services?group=core")}

	switch {
	case !last.failing.IsZero():
		cell.Value, cell.State = txt.health("db_down"), stateKO
		cell.Text = txt.health("db_ko", "time", last.failing.Format(timeLayout))
		cell.Title = txt.health("db_title_down", "type", dbType)
	case last.at.IsZero():
		cell.Value, cell.State, cell.Text = txt.health("unmeasured"), stateOff, txt.health("db_unread")
		cell.Title = txt.health("db_title_down", "type", dbType)
	default:
		ms := max(last.took.Milliseconds(), 1)
		cell.Value, cell.Unit, cell.State = strconv.FormatInt(ms, 10), txt.health("ms"), stateOK
		cell.Text = txt.health("db_ok")
		cell.Title = txt.health("db_title", "type", dbType, "ms", ms)
	}

	return cell
}

func loadCell(txt texts, node Node) gui.HealthItem {
	cell := gui.HealthItem{Label: txt.health("cpu_mem"), Title: txt.health("cpu_mem_title"), State: stateOK}

	used := node.MemoryUsed()
	total, hasTotal := node.MemoryTotal()
	share, measured := node.CPUShare()
	cores := node.Cores()
	memory := format.Bytes(txt.translate, int64(used)) //nolint:gosec // bytes of memory

	if measured {
		cell.Value = strconv.Itoa(int(math.Round(share * percent)))
		cell.Unit = "% · " + memory
	} else {
		cell.Value = memory
	}

	switch {
	case hasTotal:
		totalText := format.Bytes(txt.translate, int64(total)) //nolint:gosec // bytes
		cell.Text = txt.health("cpu_mem_text", "cores", cores, "memory", totalText)
	default:
		cell.Text = txt.health("cores", "cores", cores)
	}

	cpuHigh := measured && cores > 0 && share/float64(cores) >= loadWarn
	memoryHigh := hasTotal && total > 0 && float64(used)/float64(total) >= loadWarn

	if cpuHigh || memoryHigh {
		cell.State = stateWarn
	}

	if !measured {
		cell.State, cell.Text = stateOff, txt.health("cpu_unmeasured")
	}

	return cell
}

func diskCell(txt texts, node Node) gui.HealthItem {
	cell := gui.HealthItem{Label: txt.health("disk")}

	var (
		usages []analytics.DiskUsage
		titles []string
	)

	for _, dir := range node.Dirs() {
		usage, err := node.DiskUsageOf(dir.Name)
		if err != nil {
			continue
		}

		usages = append(usages, usage)
		freeText := format.Bytes(txt.translate, int64(usage.Free)) //nolint:gosec // bytes
		titles = append(titles, txt.health("disk_item", "dir", dirLabel(txt, dir.Name),
			"percent", int(math.Round(usage.UsedPercent)), "free", freeText))
	}

	if len(usages) == 0 {
		cell.Value, cell.State, cell.Text = txt.health("unmeasured"), stateOff, txt.health("disk_none")

		return cell
	}

	fullest := usages[0]
	for _, usage := range usages[1:] {
		if usage.UsedPercent > fullest.UsedPercent {
			fullest = usage
		}
	}

	cell.Value, cell.Unit = strconv.Itoa(int(math.Round(fullest.UsedPercent))), "%"
	fullestFree := format.Bytes(txt.translate, int64(fullest.Free)) //nolint:gosec // bytes
	cell.Text = txt.health("disk_text", "dir", dirLabel(txt, fullest.Name), "free", fullestFree)
	cell.Title = strings.Join(titles, " · ")

	switch {
	case fullest.UsedPercent >= diskKO:
		cell.State = stateKO
	case fullest.UsedPercent >= diskWarn:
		cell.State = stateWarn
	default:
		cell.State = stateOK
	}

	return cell
}

// dirLabel names a directory of the gateway: the directory of the database
// by its role, the working ones by their name.
func dirLabel(txt texts, name string) string {
	if name == analytics.DirDatabase {
		return txt.health("dir_database")
	}

	return name
}

func networkCell(txt texts, node Node) gui.HealthItem {
	received, sent := node.Throughput()
	incoming, outgoing := node.Connections()
	rate := func(bytesPerSecond float64) string {
		return txt.health("rate", "bytes", format.Bytes(txt.translate, int64(bytesPerSecond)))
	}

	return gui.HealthItem{
		Label: txt.health("network"), Value: rate(received), Unit: txt.health("received"), State: stateOK,
		Text:  txt.count(healthKeys, "network_text", uint64(incoming+outgoing), "rate", rate(sent)), //nolint:gosec // counts
		Title: txt.health("network_title", "in", incoming, "out", outgoing),
	}
}

func monitorsCell(txt texts, monitors Monitors) gui.HealthItem {
	cell := gui.HealthItem{Label: txt.health("monitors"), Href: common.WebuiURL("/watchers")}

	total, failing, err := monitors()

	switch {
	case err != nil:
		cell.Value, cell.State, cell.Text = txt.health("unmeasured"), stateOff, txt.health("monitors_error")
	case total == 0:
		cell.Value, cell.State, cell.Text = "0", stateOff, txt.health("monitors_none")
	case failing > 0:
		cell.Value, cell.Unit = strconv.Itoa(total-failing), "/ "+strconv.Itoa(total)
		cell.State, cell.Text = stateKO, txt.count(healthKeys, "monitors_ko", uint64(failing)) //nolint:gosec // a count
	default:
		cell.Value, cell.Unit = strconv.Itoa(total), "/ "+strconv.Itoa(total)
		cell.State, cell.Text = stateOK, txt.health("monitors_ok")
	}

	return cell
}
