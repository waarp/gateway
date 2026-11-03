package dashboard

import (
	"fmt"
	"slices"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/format"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/transfers"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Prefixes of the catalogue keys of the counted texts, each with its form for
// one ("dashboard.foot.planned", "dashboard.foot.planned_one").
const (
	footKeys   = "dashboard.foot."
	srKeys     = "dashboard.sr."
	errorsKeys = "dashboard.errors."
	valuesKeys = "dashboard.values."
	resumeKeys = "dashboard."
	oneSuffix  = "_one"
)

// texts translates the texts of the figures.
type texts struct{ translate gui.Translator }

// count is a counted text: its form for one, else the plural one with the
// number, its digits grouped; args are the other arguments of both forms.
func (txt texts) count(prefix, key string, n uint64, args ...any) string {
	if n == 1 {
		return txt.translate(prefix+key+oneSuffix, args...)
	}

	return txt.translate(prefix+key, append([]any{"n", groupDigits(txt.translate, n)}, args...)...)
}

// figures are the figures of the transfers the page shows, read together.
// The live transfers (the table of the transfers not over yet) give what
// runs, waits and fails: a failed transfer stays there until it is resumed,
// retried or cancelled, the history records done and cancelled transfers
// only. The history gives the activity of the window, bounded by the start
// of the transfers, its indexed column (screen sheet of 2026-10-26); the
// normalized view that joins both tables is for the lists, not for the counts.
type figures struct {
	Live   map[string]uint64 // the live transfers by status
	Recent uint64            // the failed transfers started in the last hour
	// Resumable: the failed transfers the grouped resumption of the page will
	// resume (#818): the client ones that are not the child of a synchronous
	// transfer (the rule of the model, CheckResumable), counted in one reading.
	Resumable uint64
	Day       map[string]uint64 // the transfers of the window by status, from the history
	Volume    int64             // bytes of the transfers of the window done
	Hours     []hour            // the hours of the window, the oldest first
	Errors    []failure         // the latest failed transfers, the most recent first

	// What the tiles say beyond their figure (#835): the failed transfers
	// where the gateway is the client, those where it is the server (the
	// partner resumes them), the transfers running for more than stuckAfter,
	// the planned ones whose date is past, and when the last transfer was
	// done (zero when none was).
	FailedClient, FailedServer uint64
	Stuck, Late                uint64
	LastDone                   time.Time

	Certs certSummary // the certificates of the node, judged at the reading (#836)
}

// readTileFigures reads what the tiles say beyond their figure (#835): the
// failed transfers by role (the live counts being read), the stuck and the
// overdue ones, the last transfer done.
func (h *Handlers) readTileFigures(now time.Time, figs *figures) error {
	var (
		live model.Transfers
		err  error
	)

	if figs.FailedClient, err = h.db.Select(&live).Where("status=? AND local_account_id IS NULL",
		types.StatusError).Count(); err != nil {
		return fmt.Errorf("failed to count the failed client transfers: %w", err)
	}

	if total := figs.Live[statusKey(types.StatusError)]; total > figs.FailedClient {
		figs.FailedServer = total - figs.FailedClient
	}

	if figs.Stuck, err = h.db.Select(&live).Where("status=? AND start<?", types.StatusRunning,
		now.Add(-stuckAfter).UTC()).Count(); err != nil {
		return fmt.Errorf("failed to count the stuck transfers: %w", err)
	}

	if figs.Late, err = h.db.Select(&live).Where("status=? AND start<?", types.StatusPlanned,
		now.UTC()).Count(); err != nil {
		return fmt.Errorf("failed to count the overdue transfers: %w", err)
	}

	var lastDone model.HistoryEntries
	if err = h.db.Select(&lastDone).Where("status=?", types.StatusDone).
		OrderBy("start", false).Limit(1, 0).Run(); err != nil {
		return fmt.Errorf("failed to read the last transfer done: %w", err)
	}

	if len(lastDone) == 1 {
		figs.LastDone = lastDone[0].Start
	}

	return nil
}

// hour is one hour of the window: how many transfers started in it are done,
// how many are failed.
type hour struct {
	Start        time.Time
	Done, Failed uint64
}

// failure is a failed transfer with the names its row shows.
type failure struct {
	Transfer *model.Transfer
	Rule     string
	Agent    string // the partner of a client transfer, the server of a server one
}

// windowStart is the first hour of the window: 23 whole hours before the hour
// under way, in the time zone of the gateway.
func windowStart(now time.Time) time.Time {
	current := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	return current.Add(-(windowHours - 1) * time.Hour)
}

// readFigures reads the figures from the base: a few counts on the live table and on
// the history, bounded by the start of the window, then the latest failures
// with their names. Any failure of the base ends the reading: the page then
// keeps the figures read before.
func (h *Handlers) readFigures(now time.Time) (*figures, error) {
	start := windowStart(now)
	since := start.UTC()
	figs := &figures{}

	var (
		live model.Transfers
		hist model.HistoryEntries
		err  error
	)

	if figs.Live, err = h.db.Select(&live).CountBy("status"); err != nil {
		return nil, fmt.Errorf("failed to count the live transfers: %w", err)
	}

	if figs.Resumable, err = h.db.Select(&live).Where("status=? AND local_account_id IS NULL AND id NOT IN "+
		"(SELECT transfer_id FROM transfer_info WHERE name=? AND transfer_id IS NOT NULL)",
		types.StatusError, model.SyncTransferID).Count(); err != nil {
		return nil, fmt.Errorf("failed to count the failed transfers to resume: %w", err)
	}

	if tileErr := h.readTileFigures(now, figs); tileErr != nil {
		return nil, tileErr
	}

	if figs.Day, err = h.db.Select(&hist).Where("start>=?", since).CountBy("status"); err != nil {
		return nil, fmt.Errorf("failed to count the transfers of the window: %w", err)
	}

	// The certificates are read apart: one that cannot be read does not
	// take the other figures down, the cell says they are unread.
	if figs.Certs, err = readCerts(h.db, now); err != nil {
		h.logger.Warningf("Failed to read the certificates for the dashboard: %v", err)
		figs.Certs = certSummary{Unread: true}
	}

	if figs.Volume, err = h.db.Select(&hist).Where("start>=? AND status=? AND filesize>0",
		since, types.StatusDone).Sum("filesize"); err != nil {
		return nil, fmt.Errorf("failed to sum the volume of the window: %w", err)
	}

	for i := range windowHours {
		from := since.Add(time.Duration(i) * time.Hour)

		counts, countErr := h.db.Select(&hist).Where("start>=? AND start<?", from, from.Add(time.Hour)).CountBy("status")
		if countErr != nil {
			return nil, fmt.Errorf("failed to count the transfers of an hour: %w", countErr)
		}

		figs.Hours = append(figs.Hours, hour{
			Start: start.Add(time.Duration(i) * time.Hour), Done: counts[statusKey(types.StatusDone)],
		})
	}

	// The failed transfers of the window, by hour, and the recent ones: the
	// live table is small, its failed rows are counted here.
	var failedInWindow model.Transfers
	if err = h.db.Select(&failedInWindow).Where("status=? AND start>=?", types.StatusError, since).Run(); err != nil {
		return nil, fmt.Errorf("failed to read the failures of the window: %w", err)
	}

	for _, trans := range failedInWindow {
		if slot := int(trans.Start.Sub(start) / time.Hour); slot >= 0 && slot < len(figs.Hours) {
			figs.Hours[slot].Failed++
		}

		if now.Sub(trans.Start) < recentFailures {
			figs.Recent++
		}
	}

	var failed model.Transfers
	if err = h.db.Select(&failed).Where("status=?", types.StatusError).
		OrderBy("start", false).Limit(latestErrors, 0).Eager().Run(); err != nil {
		return nil, fmt.Errorf("failed to read the latest failures: %w", err)
	}

	names := namer{db: h.db, rules: map[int64]string{}, agents: map[string]string{}}

	for _, trans := range failed {
		row := failure{Transfer: trans}

		if row.Rule, row.Agent, err = names.of(trans); err != nil {
			return nil, err
		}

		figs.Errors = append(figs.Errors, row)
	}

	return figs, nil
}

// namer gives a live transfer the names of its rule and of its partner (the
// server for a server transfer), each read once.
type namer struct {
	db     database.ReadAccess
	rules  map[int64]string
	agents map[string]string // by role and account
}

func (n *namer) of(trans *model.Transfer) (rule, agent string, err error) {
	if rule, err = n.ruleOf(trans.RuleID); err != nil {
		return "", "", err
	}

	if agent, err = n.agentOf(trans); err != nil {
		return "", "", err
	}

	return rule, agent, nil
}

func (n *namer) ruleOf(id int64) (string, error) {
	if name, ok := n.rules[id]; ok {
		return name, nil
	}

	var rule model.Rule
	if err := n.db.Get(&rule, "id=?", id).Run(); err != nil {
		return "", fmt.Errorf("failed to read the rule %d: %w", id, err)
	}

	n.rules[id] = rule.Name

	return rule.Name, nil
}

func (n *namer) agentOf(trans *model.Transfer) (string, error) {
	key := fmt.Sprintf("remote-%d", trans.RemoteAccountID.Int64)
	if trans.IsServer() {
		key = fmt.Sprintf("local-%d", trans.LocalAccountID.Int64)
	}

	if name, ok := n.agents[key]; ok {
		return name, nil
	}

	var name string

	if trans.IsServer() {
		var account model.LocalAccount
		if err := n.db.Get(&account, "id=?", trans.LocalAccountID.Int64).Run(); err != nil {
			return "", fmt.Errorf("failed to read the local account of the transfer %d: %w", trans.ID, err)
		}

		var server model.LocalAgent
		if err := n.db.Get(&server, "id=?", account.LocalAgentID).Run(); err != nil {
			return "", fmt.Errorf("failed to read the server of the transfer %d: %w", trans.ID, err)
		}

		name = server.Name
	} else {
		var account model.RemoteAccount
		if err := n.db.Get(&account, "id=?", trans.RemoteAccountID.Int64).Run(); err != nil {
			return "", fmt.Errorf("failed to read the remote account of the transfer %d: %w", trans.ID, err)
		}

		var partner model.RemoteAgent
		if err := n.db.Get(&partner, "id=?", account.RemoteAgentID).Run(); err != nil {
			return "", fmt.Errorf("failed to read the partner of the transfer %d: %w", trans.ID, err)
		}

		name = partner.Name
	}

	n.agents[key] = name

	return name, nil
}

// statusKey is the key of a status in the counts by status.
func statusKey(status types.TransferStatus) string { return string(status) }

func groupDigits(translate gui.Translator, n uint64) string { return format.GroupDigits(translate, n) }

// tiles are the three tiles of the transfers: what runs (and what waits, in
// its foot), what is failed (and how many since an hour), what was done in
// the window (with its volume and the cancellations). Each one leads to the
// list of the transfers in that state, the list of the done ones bounded by
// the window.
func tiles(txt texts, figs *figures) []gui.TileProps {
	nbRunning := figs.Live[statusKey(types.StatusRunning)]
	running := gui.TileProps{
		Label:   txt.translate("dashboard.tile.running"),
		Value:   groupDigits(txt.translate, nbRunning),
		Variant: stateInfo,
		SR:      txt.count(srKeys, "running", nbRunning),
		Href:    transfers.FilteredListURL(transfers.StateRunning, time.Time{}),
	}

	for _, waiting := range []struct {
		status types.TransferStatus
		key    string
		shape  string
	}{
		{types.StatusPlanned, "planned", stateOff},
		{types.StatusPaused, "paused", stateOff},
		{types.StatusInterrupted, "interrupted", stateWarn},
		{types.StatusAvailable, "available", stateOff},
	} {
		if n := figs.Live[statusKey(waiting.status)]; n > 0 {
			running.Statuses = append(running.Statuses,
				gui.Status{State: waiting.shape, Label: txt.count(footKeys, waiting.key, n)})
		}
	}

	// What asks for a look (#835): a transfer running for too long, a planned
	// one whose date is past.
	for _, look := range []struct {
		n   uint64
		key string
	}{{figs.Stuck, "stuck"}, {figs.Late, "late"}} {
		if look.n > 0 {
			running.Statuses = append(running.Statuses,
				gui.Status{State: stateWarn, Label: txt.count(footKeys, look.key, look.n)})
		}
	}

	nbFailed := figs.Live[statusKey(types.StatusError)]
	failed := gui.TileProps{
		Label:   txt.translate("dashboard.tile.error"),
		Value:   groupDigits(txt.translate, nbFailed),
		Variant: stateKO,
		SR:      txt.count(srKeys, "error", nbFailed),
		Href:    transfers.FilteredListURL(transfers.StateError, time.Time{}),
	}

	switch {
	case nbFailed == 0:
		failed.Variant = stateOK
		failed.Statuses = []gui.Status{{State: stateOK, Label: txt.translate("dashboard.errors.none")}}
	case figs.Recent > 0:
		failed.Statuses = []gui.Status{{State: stateKO, Label: txt.count(footKeys, "recent_errors", figs.Recent)}}
	}

	// By role (#835): the client ones are those the grouped resumption takes,
	// the server ones are resumed by the partner; each figure leads to the
	// monitoring filtered by its role (#837), the tile staying the way to
	// all of them.
	if nbFailed > 0 {
		failed.Links = []gui.Link{
			{Label: txt.count(footKeys, "client", figs.FailedClient),
				URL: transfers.RoleListURL(transfers.StateError, transfers.RoleClient)},
			{Label: txt.count(footKeys, "server", figs.FailedServer),
				URL: transfers.RoleListURL(transfers.StateError, transfers.RoleServer)},
		}
	}

	since := time.Time{}
	if len(figs.Hours) > 0 {
		since = figs.Hours[0].Start
	}

	nbDone := figs.Day[statusKey(types.StatusDone)]
	done := gui.TileProps{
		Label:   txt.translate("dashboard.tile.done"),
		Value:   groupDigits(txt.translate, nbDone),
		Variant: stateOK,
		SR:      txt.count(srKeys, "done", nbDone),
		Href:    transfers.FilteredListURL(transfers.StateDone, since),
	}

	if figs.Volume > 0 {
		done.Foot = txt.translate("dashboard.foot.volume", "volume", format.Bytes(txt.translate, figs.Volume))
	}

	if n := figs.Day[statusKey(types.StatusCancelled)]; n > 0 {
		done.Statuses = []gui.Status{{State: stateOff, Label: txt.count(footKeys, "cancelled", n)}}
	}

	if !figs.LastDone.IsZero() {
		done.Statuses = append(done.Statuses, gui.Status{
			State: stateOK, Label: txt.translate("dashboard.foot.last_done", "time", figs.LastDone.Local().Format("15:04")),
		})
	}

	return []gui.TileProps{running, failed, done}
}

// hourRow is one row of the values of the chart.
type hourRow struct {
	Hour         string
	Done, Failed string
}

// dayValues are the values of one day of the window: its label (today,
// yesterday, with its date), its totals, its hours from the most recent to
// the oldest.
type dayValues struct {
	Label string
	Meta  string
	Rows  []hourRow
}

// day is one day of the window as the chart groups it: its consecutive slots
// and their totals.
type day struct {
	start        time.Time
	slots        int
	rows         []hourRow
	done, failed uint64
}

// activity is the chart of the window and its values: one bar per hour, the
// transfers done plain and the failed ones hatched with their figure above,
// the hours grouped by day (a line at midnight, the day under its hours);
// no chart (nil) without any transfer in the window, the empty state says
// so. The values go by day, the most recent first, each day headed by its
// totals; the heading of the values gives the totals of the window.
func activity(txt texts, figs *figures) (*gui.ChartProps, []dayValues, string) {
	labels := make([]string, 0, len(figs.Hours))
	names := make([]string, 0, len(figs.Hours))
	done := make([]int, 0, len(figs.Hours))
	failed := make([]int, 0, len(figs.Hours))

	var (
		totalDone, totalFailed uint64
		days                   []*day
	)

	for _, slot := range figs.Hours {
		labels = append(labels, txt.translate("dashboard.activity.hour", "hour", slot.Start.Format("15")))
		names = append(names, slot.Start.Format("15:04"))
		done = append(done, int(slot.Done))       //nolint:gosec // a count of transfers
		failed = append(failed, int(slot.Failed)) //nolint:gosec // a count of transfers
		totalDone += slot.Done
		totalFailed += slot.Failed

		if n := len(days); n == 0 || days[n-1].start.YearDay() != slot.Start.YearDay() {
			days = append(days, &day{start: slot.Start})
		}

		current := days[len(days)-1]
		current.slots++
		current.done += slot.Done
		current.failed += slot.Failed
		current.rows = append(current.rows, hourRow{
			Hour:   slot.Start.Format("15:04"),
			Done:   groupDigits(txt.translate, slot.Done),
			Failed: groupDigits(txt.translate, slot.Failed),
		})
	}

	meta := txt.count(valuesKeys, "total_done", totalDone) + " · " + txt.count(valuesKeys, "total_error", totalFailed)
	groups, values := dayGroups(txt, days)

	if totalDone+totalFailed == 0 {
		return nil, values, meta
	}

	chart := &gui.ChartProps{
		ID:     "tdb-activity-chart",
		Title:  txt.translate("dashboard.activity.chart"),
		Desc:   txt.translate("dashboard.activity.desc"),
		Labels: labels,
		Names:  names,
		Groups: groups,
		Series: []gui.Serie{
			{
				Label: txt.translate("dashboard.legend.done"), Unit: txt.translate("dashboard.unit.done"),
				Values: done, Kind: stateOK,
			},
			{
				Label: txt.translate("dashboard.legend.error"), Unit: txt.translate("dashboard.unit.error"),
				Values: failed, Kind: stateKO,
			},
		},
	}

	return chart, values, meta
}

// dayGroups labels the days of the window: the last one is today (the window
// ends with the hour under way), the one before is yesterday. The groups of
// the chart follow the hours; the values go the other way, the most recent
// day first, its most recent hour first.
func dayGroups(txt texts, days []*day) ([]gui.ChartGroup, []dayValues) {
	groups := make([]gui.ChartGroup, 0, len(days))
	values := make([]dayValues, 0, len(days))

	for i, d := range days {
		key := "dashboard.activity.yesterday"
		if i == len(days)-1 {
			key = "dashboard.activity.today"
		}

		label := txt.translate(key, "date", d.start.Format(txt.translate("dashboard.activity.date")))
		groups = append(groups, gui.ChartGroup{Label: label, Slots: d.slots})

		rows := make([]hourRow, 0, len(d.rows))
		for _, row := range slices.Backward(d.rows) {
			rows = append(rows, row)
		}

		values = append([]dayValues{{
			Label: label,
			Meta:  txt.count(valuesKeys, "total_done", d.done) + " · " + txt.count(valuesKeys, "total_error", d.failed),
			Rows:  rows,
		}}, values...)
	}

	return groups, values
}

// errorRow is one of the latest failed transfers: when it started (its date
// under the time when it is not today), its file (a link to its sheet) with
// its rule and its partner, the cause of the failure with its detail, and the
// actions its state allows.
type errorRow struct {
	ID      int64
	Time    string
	Date    string
	File    string
	URL     string
	Sub     string
	Cause   string
	Detail  string
	Actions []gui.Action
}

func errorRows(translate gui.Translator, figs *figures, now time.Time, write bool) []errorRow {
	rows := make([]errorRow, 0, len(figs.Errors))

	for _, failed := range figs.Errors {
		trans := failed.Transfer
		clock, date := transfers.TimeAndDate(translate, trans.Start)

		if start := trans.Start.Local(); start.Year() == now.Year() && start.YearDay() == now.YearDay() {
			date = ""
		}

		cause := transfers.ErrorLabel(translate, trans.ErrCode)
		if cause == "" {
			cause = translate("transfers.state.error")
		}

		rows = append(rows, errorRow{
			ID:      trans.ID,
			Time:    clock,
			Date:    date,
			File:    transfers.FileNameOf(trans.LocalPath, trans.SrcFilename, trans.DestFilename),
			URL:     transfers.SheetURL(trans.ID),
			Sub:     failed.Rule + " · " + failed.Agent,
			Cause:   cause,
			Detail:  trans.ErrDetails,
			Actions: transfers.LiveActions(translate, trans, write),
		})
	}

	return rows
}
