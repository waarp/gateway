package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The Tasks tab of a rule: its three chains (before the transfer, after it,
// on error), each task with its type and its arguments in words, and the
// pages that add, edit, delete and move a task. Each change rewrites the
// chain in one transaction, as the REST API does: the product checks every
// task, and its refusal comes back on the form. A change to a task carries
// the state of its chain as the page showed it, and is refused when the
// chain changed since: it never applies to another task than the one shown.
// A chain is ordered again by dragging its tasks on the tab (the ordered list
// of the kit), then saved in one go. The tasks are part of the
// rule: the write right on the rules adds, edits, moves and removes them, as
// in the REST API. A task that names an object the gateway no longer has (a
// key, an SMTP credential, a template deleted since) says so in its chain,
// with the screen where to create it again.

const (
	fieldType = "type"
	argPrefix = "arg_"

	// The rule of the former tasks page, in its address.
	formerRuleParam = "ruleID"

	doneTaskAdded    = "task_added"
	doneTaskUpdated  = "task_updated"
	doneTaskDeleted  = "task_deleted"
	doneTaskMoved    = "task_moved"
	doneTaskStale    = "task_stale"
	doneTasksOrdered = "tasks_ordered"

	// The ranks the page showed, in the new order of the chain.
	orderField = "rank"

	// The state of a chain as a page showed it, in the address of a change.
	versionParam  = "v"
	versionDigits = 16

	taskTypesPrefix = "task_types."
	chainKeyPrefix  = "rules.tasks.chain."
	chainLowerKey   = "rules.tasks.chain_lower."

	trueValue      = "true"
	falseValue     = "false"
	pairsSeparator = "="
	jsonLines      = 3
	pairsLines     = 5
)

//nolint:gochecknoglobals // reference data of the screen
var (
	// The chains in the address, in their order.
	chainSlugs = []string{"pre", "post", "error"}

	// The screen (its key in the screen map) that lists the objects of a
	// kind, where a missing one is created again.
	refScreens = map[refKind]string{
		refKey: "keys", refSMTP: "smtp", refTemplate: "email_templates", refPartner: "partners",
		refRemoteAccount: "partners", refClient: "clients", refServer: "servers", refLocalAccount: "servers",
		refRule: "rules",
	}
)

func chainOf(slug string) (model.Chain, bool) {
	switch slug {
	case "pre":
		return model.ChainPre, true
	case "post":
		return model.ChainPost, true
	case "error":
		return model.ChainError, true
	}

	return "", false
}

func tasksTabURL(id int64) string { return ruleURL(id, "/tasks") }

// FormerTasksPage serves GET /webui/tasks: the address of the former tasks
// page of a rule (?ruleID=), which links and bookmarks may still hold, leads
// to its Tasks tab; without a valid rule, to the list.
func FormerTasksPage(w http.ResponseWriter, r *http.Request) {
	target := common.WebuiURL(pagePath)
	if id, err := strconv.ParseInt(r.URL.Query().Get(formerRuleParam), 10, 64); err == nil && id > 0 {
		target = tasksTabURL(id)
	}

	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

func newTaskURL(id int64, slug string) string { return ruleURL(id, "/tasks/"+slug+"/new") }

func taskURL(id int64, slug string, rank int, suffix string) string {
	return ruleURL(id, "/tasks/"+slug+"/"+strconv.Itoa(rank)+suffix)
}

// typeLabel is the title of a type of task, or its code for a type the
// screen does not know.
func typeLabel(translate gui.Translator, typ string) string {
	if spec := specOf(typ); spec != nil {
		return translate(taskTypesPrefix + spec.Slug + ".label")
	}

	return typ
}

func argKey(spec *taskSpec, arg, part string) string {
	return taskTypesPrefix + spec.Slug + ".arg." + arg + "." + part
}

// chainTasks reads the tasks of a rule, each chain by rank (sorted here: rank
// is a reserved word of MySQL).
func (h *Handlers) chainTasks(ruleID int64) (map[model.Chain][]*model.Task, error) {
	var list model.Tasks
	if err := h.db.Select(&list).Where("rule_id=?", ruleID).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	out := map[model.Chain][]*model.Task{}
	for _, task := range list {
		out[task.Chain] = append(out[task.Chain], task)
	}

	for chain := range out {
		slices.SortFunc(out[chain], func(a, b *model.Task) int { return int(a.Rank) - int(b.Rank) })
	}

	return out, nil
}

// errChainChanged: the chain changed since the page that sent the change
// showed it.
var errChainChanged = errors.New("the chain changed since the page showed it")

// versionOf is the state of a chain: a digest of its tasks in order, their
// types and their arguments by name, each string after its length (no string
// can end inside another one).
func versionOf(list []*model.Task) string {
	digest := sha256.New()
	write := func(text string) { digest.Write([]byte(strconv.Itoa(len(text)) + ":" + text)) }

	for _, task := range list {
		write(task.Type)
		write(strconv.Itoa(len(task.Args)))

		for _, name := range slices.Sorted(maps.Keys(task.Args)) {
			write(name)
			write(task.Args[name])
		}
	}

	return hex.EncodeToString(digest.Sum(nil))[:versionDigits]
}

func withVersion(u, version string) string {
	return liststate.WithQuery(u, url.Values{versionParam: {version}})
}

// modifyChain changes a chain of a rule in one transaction, as the REST API
// does: the rule is locked, its chain read again, then emptied and its tasks
// inserted again by rank, as change makes them; the product checks every
// task (a validation error comes back to the form). With a version, the
// chain must be the one a page showed (errChainChanged otherwise).
func (h *Handlers) modifyChain(ruleID int64, chain model.Chain, version string,
	change func([]*model.Task) []*model.Task,
) error {
	//nolint:wrapcheck // the caller tells a validation error and a changed chain from the others
	return h.db.Transaction(func(ses *database.Session) error {
		var rules model.Rules
		if err := ses.SelectForUpdate(&rules).Where("id=?", ruleID).Run(); err != nil {
			return err
		}

		var current model.Tasks
		if err := ses.SelectForUpdate(&current).Where("rule_id=? AND chain=?", ruleID, chain).Run(); err != nil {
			return err
		}

		slices.SortFunc(current, func(a, b *model.Task) int { return int(a.Rank) - int(b.Rank) })

		if version != "" && versionOf(current) != version {
			return errChainChanged
		}

		if err := ses.DeleteAll(&model.Task{}).Where("rule_id=? AND chain=?", ruleID, chain).Run(); err != nil {
			return err
		}

		for i, task := range change(current) {
			row := &model.Task{
				RuleID: ruleID, Chain: chain, Rank: int8(i), Type: task.Type, Args: task.Args, //nolint:gosec // a rank
			}
			if err := ses.Insert(row).Run(); err != nil {
				return err
			}
		}

		return nil
	})
}

// argView is an argument of a task in words; Missing marks an object the
// gateway does not have, with the screen that lists its kind when the user
// may open it.
type argView struct {
	Label       string
	Value       string
	Missing     bool
	ScreenLabel string
	ScreenURL   string
}

// millisLabel says a stored time limit in milliseconds: in words when it is
// whole seconds ("1 h 30 min", "30 s"), as the form shows it otherwise
// ("1.5s"); absent or zero: no limit (#784).
func millisLabel(translate gui.Translator, stored string) string {
	text := millisText(stored)
	if text == "" {
		return translate("rules.tasks.no_limit")
	}

	if d, err := time.ParseDuration(text); err == nil && d%time.Second == 0 {
		return protoform.FormatDuration(translate, text)
	}

	return text
}

// summaryOf is the part an argument takes on the folded line of its task: the
// base name of a program, after its last separator whatever the system; the
// value shown otherwise.
func summaryOf(arg *argSpec, stored, shown string) string {
	if arg.Kind == argText && stored != "" {
		return stored[strings.LastIndexAny(stored, `/\`)+1:]
	}

	return shown
}

// lookup is what argsOf checks the named objects against: their names, and
// the screens the user may open.
type lookup struct {
	found   namedObjects
	screens map[string]common.ScreenEntry
}

// lookupOf reads the objects the tasks of the list name, and the screens of
// the user.
func (h *Handlers) lookupOf(r *http.Request, list []*model.Task) (*lookup, error) {
	specs := make([]*taskSpec, 0, len(list))
	for _, task := range list {
		if spec := specOf(task.Type); spec != nil {
			specs = append(specs, spec)
		}
	}

	found, err := h.namesFor(specs...)
	if err != nil {
		return nil, err
	}

	look := &lookup{found: found, screens: map[string]common.ScreenEntry{}}

	for _, section := range common.Screens(common.GetUser(r).Permissions) {
		for _, screen := range section.Screens {
			look.screens[screen.Key] = screen
		}
	}

	return look, nil
}

// missing marks a named object the gateway does not have, with the screen
// where its kind is listed.
func (l *lookup) missing(translate gui.Translator, arg *argSpec, view *argView) {
	if l == nil || arg.Kind != argRef || l.found.has(arg.Ref, view.Value) {
		return
	}

	view.Missing = true
	if screen, ok := l.screens[refScreens[arg.Ref]]; ok {
		view.ScreenLabel, view.ScreenURL = translate(screen.LabelKey), screen.URL
	}
}

// argsOf says the arguments of a task in words: those of its description in
// its order (a ticked box as "Yes", or as its mode, a duration asked in
// seconds in seconds, a choice by its label, a missing object
// marked when look is given), then the others by name as they are (the keys
// of a SETINFO task among them).
func argsOf(translate gui.Translator, task *model.Task, look *lookup) (views []argView, meta string) {
	spec := specOf(task.Type)
	known := map[string]bool{}

	var (
		out     []argView
		summary []string
	)

	if spec != nil {
		for i := range spec.Args {
			arg := &spec.Args[i]
			if arg.Kind == argPairs {
				continue
			}

			known[arg.Name] = true

			value := task.Args[arg.Name]
			if arg.Millis {
				value = millisLabel(translate, value)
			}

			if value == "" || arg.Kind == argBool && value != trueValue {
				continue
			}

			switch {
			case arg.Kind == argBool && arg.Modes[1] != "":
				value = translate(arg.Modes[1])
			case arg.Kind == argBool:
				value = translate("rules.tasks.ticked")
			case arg.Seconds:
				value = secondsText(value)
			}

			if arg.Kind == argChoice {
				value = choiceLabel(translate, arg, value)
			}

			view := argView{Label: translate(argKey(spec, arg.Name, "label")), Value: value}
			look.missing(translate, arg, &view)
			out = append(out, view)

			if arg.Summary {
				summary = append(summary, summaryOf(arg, task.Args[arg.Name], value))
			}
		}
	}

	names := make([]string, 0, len(task.Args))
	for name := range task.Args {
		if !known[name] {
			names = append(names, name)
		}
	}

	slices.Sort(names)

	for _, name := range names {
		out = append(out, argView{Label: name, Value: task.Args[name]})
	}

	return out, strings.Join(summary, " · ")
}

// choiceLabel is the label of a value of a choice, or the value itself.
func choiceLabel(translate gui.Translator, arg *argSpec, value string) string {
	for _, c := range arg.Choices() {
		if c.Value == value && c.Key != "" {
			return translate(c.Key)
		}
	}

	return value
}

type taskRow struct {
	ID      string
	Rank    int
	Label   string
	Code    string
	Meta    string // the folded line: what tells the task apart (the program, its limit)
	Missing bool   // an object the task names that the gateway no longer has
	Tip     string // the label and what the task does, in its tooltip
	Unknown bool
	Args    []argView
	Actions []gui.Action
}

type chainView struct {
	Slug   string
	Title  string
	Lower  string // the chain in a sentence: "Add a task" + " before the transfer"
	Count  string
	AddURL string
	// The form that saves a new order (empty when the chain cannot be
	// ordered: no write right, fewer than two tasks) and its id.
	OrderURL string
	OrderID  string
	Rows     []taskRow
}

type tasksData struct {
	Title   string
	Tabs    []means.Tab
	BackURL string
	Actions []gui.Action // the menu of the rule: its transfers, its deletion
	Chains  []chainView
}

// TasksTab serves GET /webui/rules/{id}/tasks: the three chains of the rule.
func (h *Handlers) TasksTab(w http.ResponseWriter, r *http.Request) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	chains, err := h.chainTasks(item.rule.ID)
	if err != nil {
		h.internal(w, "read the tasks of the transfer rule", err)

		return
	}

	look, err := h.lookupOf(r, slices.Concat(chains[model.ChainPre], chains[model.ChainPost], chains[model.ChainError]))
	if err != nil {
		h.internal(w, "read the objects the tasks name", err)

		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)
	data := tasksData{
		Title:   ruleTitle(translate, item.rule),
		Tabs:    tabs(translate, &item, tabTasks),
		BackURL: listURL(carried),
		Actions: actions(translate, &item, user, true, carried),
	}

	for _, slug := range chainSlugs {
		chain, _ := chainOf(slug)
		list := chains[chain]
		view := chainView{
			Slug: slug, Title: translate(chainKeyPrefix + slug), Lower: translate(chainLowerKey + slug),
			Count: taskCount(translate, len(list)),
		}

		if user.Write {
			view.AddURL = withRet(newTaskURL(item.rule.ID, slug), carried)

			if len(list) > 1 {
				view.OrderID = "order-" + slug
				view.OrderURL = withRet(withVersion(ruleURL(item.rule.ID, "/tasks/"+slug+"/order"), versionOf(list)), carried)
			}
		}

		for i, task := range list {
			place := &taskPlace{ruleID: item.rule.ID, slug: slug, rank: i, count: len(list)}
			args, meta := argsOf(translate, task, look)
			row := taskRow{
				ID: slug + "-" + strconv.Itoa(i), Rank: i + 1, Label: typeLabel(translate, task.Type),
				Code: task.Type, Meta: meta, Unknown: specOf(task.Type) == nil, Args: args,
			}

			row.Missing = slices.ContainsFunc(row.Args, func(a argView) bool { return a.Missing })

			if spec := specOf(task.Type); spec != nil {
				row.Tip = translate("rules.tasks.tip", "label", row.Label,
					"text", translate(taskTypesPrefix+spec.Slug+".text"))
			}

			if user.Write {
				row.Actions = taskActions(translate, place, row.Unknown, versionOf(list), carried)
			}

			view.Rows = append(view.Rows, row)
		}

		data.Chains = append(data.Chains, view)
	}

	screen := h.screen(translate, data.Title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	screen.Head = gui.SortableAssets()
	h.shell.Render(w, h.logger, r, screen, tasksView(data))
}

// taskPlace is where a task stands: its rule, its chain, its rank and the
// length of the chain.
type taskPlace struct {
	ruleID int64
	slug   string
	rank   int
	count  int
}

func (p *taskPlace) chain() model.Chain {
	chain, _ := chainOf(p.slug)

	return chain
}

// taskCount is the number of tasks of a chain, in the head of its section.
func taskCount(translate gui.Translator, n int) string {
	switch n {
	case 0:
		return translate("rules.tasks.count_none")
	case 1:
		return translate("rules.tasks.count_one")
	default:
		return translate("rules.tasks.count", "n", n)
	}
}

// taskActions: the menu of a task: edit (a known type), move up, move down
// (the kit has no upward chevron: the stylesheet turns the downward one),
// delete; each one carries the version of the chain shown.
func taskActions(translate gui.Translator, place *taskPlace, unknown bool, version, carried string,
) []gui.Action {
	var acts []gui.Action

	at := func(suffix string) string {
		return withRet(withVersion(taskURL(place.ruleID, place.slug, place.rank, suffix), version), carried)
	}

	if !unknown {
		acts = append(acts, gui.Action{Icon: "pen", Label: translate("rules.tasks.action.edit"), URL: at("/edit")})
	}

	if place.rank > 0 {
		acts = append(acts, gui.Action{Icon: "chevron-down", Label: translate("rules.tasks.action.up"), Action: at("/up")})
	}

	if place.rank < place.count-1 {
		acts = append(acts, gui.Action{
			Icon: "chevron-down", Label: translate("rules.tasks.action.down"), Action: at("/down"),
		})
	}

	return append(acts, gui.Action{
		Icon: "trash", Label: translate("rules.tasks.action.delete"), URL: at("/delete"), Danger: true,
	})
}

// stale answers a change from a page that no longer shows its chain: the
// tab, with its current state and the notice that nothing was saved.
func (h *Handlers) stale(w http.ResponseWriter, r *http.Request, ruleID int64) {
	liststate.RedirectTo(w, r, withRet(tasksTabURL(ruleID), ret(r)), url.Values{doneParam: {doneTaskStale}})
}

// taskRoute reads the {chain} and, withRank, the {rank} of the route; a chain
// or a rank the rule does not have renders the "not found" page. A version
// in the address must be the one of the chain (the tab otherwise, with its
// notice); a change to a task (a POST with a rank) must carry one.
func (h *Handlers) taskRoute(w http.ResponseWriter, r *http.Request, item *entry, withRank bool,
) (*taskPlace, []*model.Task, bool) {
	vars := mux.Vars(r)
	slug := vars["chain"]

	if chain, known := chainOf(slug); known {
		chains, err := h.chainTasks(item.rule.ID)
		if err != nil {
			h.internal(w, "read the tasks of the transfer rule", err)

			return nil, nil, false
		}

		list := chains[chain]
		rank, rankErr := strconv.Atoi(vars["rank"])
		version := r.URL.Query().Get(versionParam)

		if (version != "" || withRank && r.Method == http.MethodPost) && version != versionOf(list) {
			h.stale(w, r, item.rule.ID)

			return nil, nil, false
		}

		if !withRank || rankErr == nil && rank >= 0 && rank < len(list) {
			return &taskPlace{ruleID: item.rule.ID, slug: slug, rank: rank, count: len(list)}, list, true
		}
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("rules.tasks.not_found.title")
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound),
		means.NotFoundView(means.NotFoundData{
			Title: title, Text: translate("rules.tasks.not_found.text"), Icon: "list",
			BackLabel: translate("rules.tasks.back"), BackURL: withRet(tasksTabURL(item.rule.ID), carried),
		}))

	return nil, nil, false
}

type taskDeleteData struct {
	Title     string
	Label     string
	Code      string
	Args      []argView
	CancelURL string
	Action    string
}

// TaskDeleteForm serves GET /webui/rules/{id}/tasks/{chain}/{rank}/delete.
func (h *Handlers) TaskDeleteForm(w http.ResponseWriter, r *http.Request) {
	h.taskDelete(w, r, false)
}

// TaskDelete serves POST /webui/rules/{id}/tasks/{chain}/{rank}/delete: the
// task leaves the chain, the next ones move up one rank.
func (h *Handlers) TaskDelete(w http.ResponseWriter, r *http.Request) {
	h.taskDelete(w, r, true)
}

func (h *Handlers) taskDelete(w http.ResponseWriter, r *http.Request, post bool) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	place, list, ok := h.taskRoute(w, r, &item, true)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	task := list[place.rank]
	label := typeLabel(translate, task.Type)

	if post {
		err := h.modifyChain(place.ruleID, place.chain(), versionOf(list), func(current []*model.Task) []*model.Task {
			return slices.Delete(current, place.rank, place.rank+1)
		})

		switch {
		case errors.Is(err, errChainChanged):
			h.stale(w, r, place.ruleID)

			return
		case err != nil:
			h.internal(w, "delete a task of the transfer rule", err)

			return
		}

		liststate.RedirectTo(w, r, withRet(tasksTabURL(place.ruleID), carried),
			url.Values{doneParam: {doneTaskDeleted}, nameParam: {label}})

		return
	}

	args, _ := argsOf(translate, task, nil)
	title := translate("rules.tasks.delete_title", "n", place.rank+1, "chain", translate(chainLowerKey+place.slug))
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), taskDeleteView(taskDeleteData{
		Title:     title,
		Label:     label,
		Code:      task.Type,
		Args:      args,
		CancelURL: withRet(tasksTabURL(place.ruleID), carried),
		Action:    withRet(withVersion(taskURL(place.ruleID, place.slug, place.rank, "/delete"), versionOf(list)), carried),
	}))
}

// TaskOrder serves POST /webui/rules/{id}/tasks/{chain}/order: the new order
// of a chain, dragged on the tab, as the ranks the page showed in their new
// order. The chain, rewritten in one transaction, must be the one the page
// showed (the tab otherwise, with its notice).
func (h *Handlers) TaskOrder(w http.ResponseWriter, r *http.Request) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	place, list, ok := h.taskRoute(w, r, &item, false)
	if !ok {
		return
	}

	version := r.URL.Query().Get(versionParam)
	if err := r.ParseForm(); err != nil || version == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	order, valid := orderOf(r.PostForm[orderField], len(list))
	if !valid {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	err := h.modifyChain(place.ruleID, place.chain(), version, func(current []*model.Task) []*model.Task {
		ordered := make([]*model.Task, 0, len(current))
		for _, rank := range order {
			ordered = append(ordered, current[rank])
		}

		return ordered
	})

	switch {
	case errors.Is(err, errChainChanged):
		h.stale(w, r, place.ruleID)

		return
	case err != nil:
		h.internal(w, "order the tasks of the transfer rule", err)

		return
	}

	liststate.RedirectTo(w, r, withRet(tasksTabURL(place.ruleID), ret(r)),
		url.Values{doneParam: {doneTasksOrdered}, nameParam: {h.shell.Translator(r)(chainLowerKey + place.slug)}})
}

// orderOf reads the posted order of a chain of n tasks: each rank once.
func orderOf(values []string, n int) ([]int, bool) {
	if len(values) != n {
		return nil, false
	}

	seen := make([]bool, n)
	order := make([]int, 0, n)

	for _, value := range values {
		rank, err := strconv.Atoi(value)
		if err != nil || rank < 0 || rank >= n || seen[rank] {
			return nil, false
		}

		seen[rank] = true
		order = append(order, rank)
	}

	return order, true
}

// TaskUp serves POST /webui/rules/{id}/tasks/{chain}/{rank}/up.
func (h *Handlers) TaskUp(w http.ResponseWriter, r *http.Request) { h.taskMove(w, r, -1) }

// TaskDown serves POST /webui/rules/{id}/tasks/{chain}/{rank}/down.
func (h *Handlers) TaskDown(w http.ResponseWriter, r *http.Request) { h.taskMove(w, r, 1) }

// taskMove swaps a task with the previous or the next one of its chain,
// which is rewritten in one transaction; at either end, nothing moves.
func (h *Handlers) taskMove(w http.ResponseWriter, r *http.Request, step int) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	place, list, ok := h.taskRoute(w, r, &item, true)
	if !ok {
		return
	}

	result := url.Values{}

	if target := place.rank + step; target >= 0 && target < len(list) {
		err := h.modifyChain(place.ruleID, place.chain(), versionOf(list), func(current []*model.Task) []*model.Task {
			current[place.rank], current[target] = current[target], current[place.rank]

			return current
		})

		switch {
		case errors.Is(err, errChainChanged):
			h.stale(w, r, place.ruleID)

			return
		case err != nil:
			h.internal(w, "move a task of the transfer rule", err)

			return
		}

		result = url.Values{doneParam: {doneTaskMoved}, nameParam: {typeLabel(h.shell.Translator(r), list[place.rank].Type)}}
	}

	liststate.RedirectTo(w, r, withRet(tasksTabURL(place.ruleID), ret(r)), result)
}
