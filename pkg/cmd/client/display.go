package wg

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"golang.org/x/exp/constraints"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

//nolint:gochecknoglobals //basically constants, but can't be made constants
var (
	Style0    = &style{color: color.OpReverse, bulletPrefix: ""}
	Style1    = &style{color: color.HiMagenta, bulletPrefix: "-"}
	Style22   = &style{color: color.HiCyan, bulletPrefix: "  -"}
	Style333  = &style{color: color.Bold, bulletPrefix: "    -"}
	Style4444 = &style{color: color.FgDefault, bulletPrefix: "      -"}

	noPerm        = color.Gray.Render("---")
	empty         = color.Gray.Render("<empty>")
	none          = color.Gray.Render("<none>")
	notApplicable = color.Gray.Render("N/A")
	unrestricted  = color.Gray.Render("<unrestricted>")
	unspecified   = color.Gray.Render("<unspecified>")
	sizeUnknown   = color.Gray.Render("<unknown>")
)

type style struct {
	color        color.Color
	bulletPrefix string
}

func (s *style) PrintV(w io.Writer, msg string) {
	fmt.Fprintln(w, s.sprint(msg))
}

func (s *style) Printf(w io.Writer, format string, args ...any) {
	fmt.Fprintln(w, s.sprintf(format, args...))
}

func (s *style) sprintf(format string, args ...any) string {
	return s.sprint(fmt.Sprintf(format, args...))
}

func (s *style) sprint(text string) string {
	text = strings.ReplaceAll(text, color.ResetSet, color.StartSet+s.color.Code()+"m")
	text = strings.ReplaceAll(text, color.StartSet, color.StartSet+"0;")
	text = s.color.Render(text)

	return color.Sprintf("%s%s", s.bulletPrefix, text)
}

func (s *style) PrintL(w io.Writer, name string, value any) {
	fmt.Fprintln(w, s.sprintL(name, value))
}

func (s *style) sprintL(name string, value any) string {
	return color.Sprintf("%s%s: %v", s.bulletPrefix, s.color.Render(name), value)
}

func (s *style) Option(w io.Writer, name string, value any) {
	if value != nil && !reflect.ValueOf(value).IsZero() {
		s.PrintL(w, name, value)
	}
}

func (s *style) Defaul(w io.Writer, name string, value, defaultVal any) {
	if value != nil && !reflect.ValueOf(value).IsZero() {
		s.PrintL(w, name, value)
	} else {
		s.PrintL(w, name, defaultVal)
	}
}

func (s *style) MultiL(w io.Writer, name, value string) {
	lines := strings.Split(value, "\n")
	if len(lines) <= 1 {
		s.PrintL(w, name, value)

		return
	}

	indent := strings.Repeat(" ", utf8.RuneCountInString(nextStyle(s).bulletPrefix))
	s.PrintV(w, name+":")

	for _, line := range lines {
		fmt.Fprintln(w, indent+line)
	}
}

func nextStyle(style *style) *style {
	switch style {
	case Style0:
		return Style1
	case Style1:
		return Style22
	case Style22:
		return Style333
	case Style333:
		return Style4444
	default:
		return Style0
	}
}

func displayMap[T any](w io.Writer, style *style, m map[string]T) {
	pairs := make([]pair, 0, len(m))

	for key, val := range m {
		pairs = append(pairs, pair{key: key, val: val})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})

	for i := range pairs {
		style.PrintL(w, pairs[i].key, pairs[i].val)
	}
}

func displayProtoConfig(w io.Writer, cfg map[string]any) {
	if len(cfg) == 0 {
		Style22.PrintL(w, "Configuration", empty)

		return
	}

	Style22.Printf(w, "Configuration:")
	displayMap(w, Style333, cfg)
}

func displayAuthorizedRules(w io.Writer, auth api.AuthorizedRules) {
	targets := func(a []string) string {
		return withDefault(join(a), none)
	}

	Style22.Printf(w, "Authorized rules:")
	Style333.PrintL(w, "Send", targets(auth.Sending))
	Style333.PrintL(w, "Receive", targets(auth.Reception))
}

func displayRuleAccess(w io.Writer, access api.RuleAccess) {
	if len(access.Servers) == 0 && len(access.Partners) == 0 &&
		len(access.LocalAccounts) == 0 && len(access.RemoteAccounts) == 0 {
		Style22.PrintL(w, "Rule access", unrestricted)

		return
	}

	targets := func(a []string) string {
		return withDefault(join(a), none)
	}
	subTargets := func(m map[string][]string) string {
		return withDefault(mapFlatten(m), none)
	}

	Style22.Printf(w, "Rule access:")
	Style333.PrintL(w, "Local servers", targets(access.Servers))
	Style333.PrintL(w, "Remote partners", targets(access.Partners))
	Style333.PrintL(w, "Local accounts", subTargets(access.LocalAccounts))
	Style333.PrintL(w, "Remote accounts", subTargets(access.RemoteAccounts))
}

func join(s []string) string { return strings.Join(s, ", ") }

func joinStringers[T fmt.Stringer](stringers []T) string {
	str := make([]string, 0, len(stringers))
	for _, stringer := range stringers {
		str = append(str, stringer.String())
	}

	return join(str)
}

func mapFlatten(m map[string][]string) string {
	var fullValues []string

	for key, vals := range m {
		for _, val := range vals {
			fullValues = append(fullValues, fmt.Sprintf("%s: %s", key, val))
		}
	}

	sort.Strings(fullValues)

	return join(fullValues)
}

func displayTask(w io.Writer, index int, task *api.Task) {
	if len(task.Args) == 0 {
		Style333.Printf(w, "Task #%d %q", index+1, task.Type)
	} else {
		Style333.Printf(w, "Task #%d %q with args:", index+1, task.Type)
		displayMap(w, Style4444, task.Args)
	}
}

func displayTaskChain(w io.Writer, title string, chain []*api.Task) {
	if len(chain) == 0 {
		Style22.PrintL(w, title, none)

		return
	}

	Style22.PrintV(w, title+":")

	for i, task := range chain {
		displayTask(w, i, task)
	}
}

func prettyBytes[T constraints.Integer](val T) string {
	if val < 0 {
		return sizeUnknown
	}

	return humanize.Bytes(uint64(val))
}

func cardinal[T constraints.Integer](n T) string {
	if n > 0 {
		return fmt.Sprintf("#%d", n)
	}

	return ""
}
