// Package ruleuse finds and renders the rules whose tasks name an object of
// the gateway (a cryptographic key, an SMTP credential): a task names it in
// its arguments, without a link in the database. The screen of the object
// shows these rules, each with its direction and a link to its tasks, keeps
// the name of a used object, and replaces its deletion by the list of the
// rules to change first.
package ruleuse

import (
	"cmp"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Use is a rule whose tasks name an object: what the screens show of it, the
// tasks themselves being always the same ones for an object.
type Use struct {
	RuleID int64
	Rule   string
	Send   bool
}

// Params are, for each task type that reads the object, the arguments that
// name it.
type Params map[string][]string

// Of finds, for every name, the rules whose tasks name it, one line per rule,
// by rule name then direction (sending first).
func Of(list model.Tasks, rules model.Rules, params Params) map[string][]Use {
	byID := make(map[int64]*model.Rule, len(rules))
	for _, r := range rules {
		byID[r.ID] = r
	}

	uses := map[string][]Use{}

	for _, t := range list {
		r, ok := byID[t.RuleID]
		if !ok {
			continue
		}

		for _, param := range params[t.Type] {
			name := t.Args[param]
			u := Use{RuleID: r.ID, Rule: r.Name, Send: r.IsSend}

			if name != "" && !slices.Contains(uses[name], u) {
				uses[name] = append(uses[name], u)
			}
		}
	}

	for name := range uses {
		slices.SortFunc(uses[name], func(a, b Use) int {
			return cmp.Or(strings.Compare(a.Rule, b.Rule), directionOrder(a.Send)-directionOrder(b.Send))
		})
	}

	return uses
}

// directionOrder puts the sending rule before the receiving one of the same
// name.
func directionOrder(send bool) int {
	if send {
		return 0
	}

	return 1
}

// Read reads the tasks of the given types and the rules, and finds the rules
// that name each object.
func Read(db database.ReadAccess, params Params) (map[string][]Use, error) {
	var (
		list  model.Tasks
		rules model.Rules
	)

	types := make([]any, 0, len(params))
	for typ := range params {
		types = append(types, typ)
	}

	if err := db.Select(&list).In("type", types...).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	if err := db.Select(&rules).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	return Of(list, rules, params), nil
}

// Stylesheet is the stylesheet of List, for the screens that show it.
const Stylesheet = "/static_v2/ruleuse/ruleuse.css"

// Row is a rule that names an object, ready to render: a link to the tasks
// of the rule, where the object is changed, and its direction.
type Row struct {
	Rule      string
	URL       string
	Direction string
}

// Rows prepares the rules for the list.
func Rows(translate gui.Translator, uses []Use) []Row {
	out := make([]Row, 0, len(uses))
	for _, u := range uses {
		direction := translate("rule_uses.direction.receive")
		if u.Send {
			direction = translate("rule_uses.direction.send")
		}

		out = append(out, Row{
			Rule:      u.Rule,
			URL:       common.WebuiURL("/tasks") + "?" + url.Values{"ruleID": {strconv.FormatInt(u.RuleID, 10)}}.Encode(),
			Direction: direction,
		})
	}

	return out
}

// Count says in a cell or a sentence how many rules use an object.
func Count(translate gui.Translator, n int) string {
	switch n {
	case 0:
		return translate("rule_uses.count.none")
	case 1:
		return translate("rule_uses.count.one")
	default:
		return translate("rule_uses.count.many", "n", n)
	}
}
