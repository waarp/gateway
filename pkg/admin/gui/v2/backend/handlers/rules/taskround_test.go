package rules

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The arguments left out of a task with every argument set: TRANSFER takes a
// partner on one side only.
//
//nolint:gochecknoglobals // reference data of the test
var sampleOmit = map[string][]string{tasks.Transfer: {"from"}}

// fullArgs are the arguments of a type, every one set with a value the
// product accepts; the boxes take boxValue.
func fullArgs(spec *taskSpec, boxValue string) map[string]string {
	args := map[string]string{}

	for i := range spec.Args {
		arg := &spec.Args[i]
		if slices.Contains(sampleOmit[spec.Type], arg.Name) {
			continue
		}

		switch arg.Kind {
		case argPairs:
			maps.Copy(args, map[string]string{"lot": "A", "vide": ""})

			continue
		case argBool:
			args[arg.Name] = boxValue
		case argChoice:
			args[arg.Name] = arg.Choices()[0].Value
		case argRef:
			args[arg.Name] = sampleRefs[arg.Ref]
		default:
			args[arg.Name] = sampleKinds[arg.Kind]
		}

		if value, ok := sampleOverrides[spec.Type][arg.Name]; ok {
			args[arg.Name] = value
		}
	}

	return args
}

func attr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val, true
		}
	}

	return "", false
}

// selectedOf is the value a select sends: its selected option, or its first
// one.
func selectedOf(sel *html.Node) string {
	var first, chosen *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "option" {
			if first == nil {
				first = n
			}

			if _, ok := attr(n, "selected"); ok && chosen == nil {
				chosen = n
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(sel)

	if chosen == nil {
		chosen = first
	}

	value, _ := attr(chosen, "value")

	return value
}

// postedForm reads the form of a page as a browser sends it: its action, and
// the values of its fields (a box when checked, the chosen option of a
// select, the text of a text area).
func postedForm(t *testing.T, body string) (string, url.Values) {
	t.Helper()

	doc, err := html.Parse(strings.NewReader(body))
	require.NoError(t, err)

	var form *html.Node

	var find func(*html.Node)
	find = func(n *html.Node) {
		class, _ := attr(n, "class")
		if method, _ := attr(n, "method"); form == nil && n.Data == "form" && method == "post" &&
			slices.Contains(strings.Fields(class), "w-form") { // not the sign-out form of the chrome
			form = n
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	require.NotNil(t, form, "the page has a form")

	values := url.Values{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		name, named := attr(n, "name")

		switch {
		case n.Type != html.ElementNode || !named:
		case n.Data == "input":
			value, _ := attr(n, "value")
			_, checked := attr(n, "checked")

			if typ, _ := attr(n, "type"); typ != "checkbox" || checked {
				values.Add(name, value)
			}
		case n.Data == "textarea":
			var text strings.Builder
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				text.WriteString(c.Data)
			}

			values.Add(name, text.String())
		case n.Data == "select":
			values.Add(name, selectedOf(n))

			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(form)

	action, _ := attr(form, "action")

	return action, values
}

// A task opened in the tab and saved without any change keeps exactly its
// arguments: every type, every argument set, the boxes ticked then left
// empty (a box stored as "false" stays "false").
func TestTaskRoundTrip(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)
	s.keyFixtures(t)

	for i := range taskSpecs {
		spec := &taskSpecs[i]

		boxes := []string{trueValue}
		if slices.ContainsFunc(spec.Args, func(a argSpec) bool { return a.Kind == argBool }) {
			boxes = append(boxes, falseValue)
		}

		for _, box := range boxes {
			name := "aller-retour-" + spec.Slug + "-" + box
			rule := &model.Rule{Name: name, IsSend: true, Path: name}
			require.NoError(t, s.db.Insert(rule).Run())

			stored := fullArgs(spec, box)
			require.NoError(t, s.db.Insert(&model.Task{RuleID: rule.ID, Chain: model.ChainPre, Type: spec.Type, Args: stored}).Run(),
				"the product saves %s with every argument", spec.Type)

			action, values := postedForm(t, s.get(t, permWrite, path(rule, "/tasks/pre/0/edit")).Body.String())
			rec := s.post(t, permWrite, action, values)

			if assert.Equal(t, http.StatusSeeOther, rec.Code, "%s (boxes %s) is saved", spec.Type, box) {
				assert.Contains(t, rec.Header().Get("Location"), "done=task_updated")
			}

			assert.Equal(t, stored, map[string]string(s.chain(t, rule, model.ChainPre)[0].Args),
				"%s (boxes %s) keeps exactly its arguments", spec.Type, box)
		}
	}
}
