package dashboard

import (
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/format"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/transfers"
)

// The thresholds of the strip:
// the memory in warning at 80 % of what the machine has.
const (
	healthKeys = "dashboard.health."
	loadWarn   = 0.8
	hoursADay  = 24
)

// healthInput is what the strip of the node is built from.
type healthInput struct {
	node     Node
	now      time.Time
	dbType   string
	last     reading // the last reading of the figures: how long it took, since when it fails
	services services.Summary
	monitors Monitors
}

// healthCells builds the cells of the strip: uptime, services, database,
// memory, open connections, partner monitors. Each cell
// carries its state by a shape and a word; a figure that cannot be read is
// "unmeasured", never a false figure.
func healthCells(translate gui.Translator, in *healthInput) []gui.HealthItem {
	txt := texts{translate}

	return []gui.HealthItem{
		uptimeCell(txt, in.node, in.now),
		servicesCell(txt, in.services),
		databaseCell(txt, in.dbType, in.last),
		memoryCell(txt, in.node),
		networkCell(txt, in.node),
		monitorsCell(txt, in.monitors),
	}
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

func memoryCell(txt texts, node Node) gui.HealthItem {
	cell := gui.HealthItem{Label: txt.health("memory"), Title: txt.health("memory_title"), State: stateOK}

	used := node.MemoryUsed()
	total, hasTotal := node.MemoryTotal()
	cores := node.Cores()
	cell.Value = format.Bytes(txt.translate, int64(used)) //nolint:gosec // bytes of memory

	if hasTotal {
		totalText := format.Bytes(txt.translate, int64(total)) //nolint:gosec // bytes
		cell.Text = txt.health("memory_text", "cores", cores, "memory", totalText)
	} else {
		cell.Text = txt.health("cores", "cores", cores)
	}

	if hasTotal && total > 0 && float64(used)/float64(total) >= loadWarn {
		cell.State = stateWarn
	}

	return cell
}

func networkCell(txt texts, node Node) gui.HealthItem {
	incoming, outgoing := node.Connections()

	return gui.HealthItem{
		Label: txt.health("network"), Value: strconv.FormatInt(incoming+outgoing, 10), State: stateOK,
		Text:  txt.health("network_text", "in", incoming, "out", outgoing),
		Title: txt.health("network_title"),
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
