package wg

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"golang.org/x/exp/constraints"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

//nolint:gochecknoglobals //basically constants, but can't be made constants
var (
	style0    = &style{color: color.OpReverse, bulletPrefix: ""}
	style1    = &style{color: color.HiMagenta, bulletPrefix: "‣"}
	style22   = &style{color: color.HiCyan, bulletPrefix: "  •"}
	style333  = &style{color: color.Bold, bulletPrefix: "    ⁃"}
	style4444 = &style{color: color.FgDefault, bulletPrefix: "      $"}

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

func (s *style) printf(w io.Writer, format string, args ...any) {
	fmt.Fprintln(w, s.sprintf(format, args...))
}

func (s *style) sprintf(format string, args ...any) string {
	text := fmt.Sprintf(format, args...)
	text = strings.ReplaceAll(text, color.ResetSet, color.StartSet+s.color.Code()+"m")
	text = strings.ReplaceAll(text, color.StartSet, color.StartSet+"0;")

	if strings.HasSuffix(text, ":") {
		text = s.color.Render(strings.TrimSuffix(text, ":")) + ":"
	} else {
		text = s.color.Render(text)
	}

	return color.Sprintf("%s%s", s.bulletPrefix, text)
}

func (s *style) printL(w io.Writer, name string, value any) {
	fmt.Fprintln(w, s.sprintL(name, value))
}

func (s *style) sprintL(name string, value any) string {
	return color.Sprintf("%s%s: %s", s.bulletPrefix, s.color.Render(name), value)
}

func (s *style) option(w io.Writer, name string, value any) {
	if value != nil && !reflect.ValueOf(value).IsZero() {
		s.printL(w, name, value)
	}
}

func nextStyle(style *style) *style {
	switch style {
	case style0:
		return style1
	case style1:
		return style22
	case style22:
		return style333
	case style333:
		return style4444
	default:
		return style0
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
		style.printL(w, pairs[i].key, pairs[i].val)
	}
}

func displayProtoConfig(w io.Writer, cfg map[string]any) {
	if len(cfg) == 0 {
		style22.printL(w, "Configuration", empty)

		return
	}

	style22.printf(w, "Configuration:")
	displayMap(w, style333, cfg)
}

func displayAuthorizedRules(w io.Writer, auth api.AuthorizedRules) {
	targets := func(a []string) string {
		return withDefault(join(a), none)
	}

	style22.printf(w, "Authorized rules:")
	style333.printL(w, "Send", targets(auth.Sending))
	style333.printL(w, "Receive", targets(auth.Reception))
}

func displayRuleAccess(w io.Writer, access api.RuleAccess) {
	if len(access.LocalServers) == 0 && len(access.RemotePartners) == 0 &&
		len(access.LocalAccounts) == 0 && len(access.RemoteAccounts) == 0 {
		style22.printL(w, "Rule access", unrestricted)

		return
	}

	targets := func(a []string) string {
		return withDefault(join(a), none)
	}
	subTargets := func(m map[string][]string) string {
		return withDefault(mapFlatten(m), none)
	}

	style22.printf(w, "Rule access:")
	style333.printL(w, "Local servers", targets(access.LocalServers))
	style333.printL(w, "Remote partners", targets(access.RemotePartners))
	style333.printL(w, "Local accounts", subTargets(access.LocalAccounts))
	style333.printL(w, "Remote accounts", subTargets(access.RemoteAccounts))
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
		style333.printf(w, "Task #%d %q", index+1, task.Type)
	} else {
		style333.printf(w, "Task #%d %q with args:", index+1, task.Type)
		displayMap(w, style4444, task.Args)
	}
}

func displayTaskChain(w io.Writer, title string, chain []*api.Task) {
	if len(chain) == 0 {
		style22.printL(w, title, none)

		return
	}

	style22.printf(w, title+":")

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
