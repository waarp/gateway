package varlist

import (
	"slices"
	"strings"
	"time"
	"unicode"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The variables list is built from the markers the tasks implement
// (pkg/tasks), grouped by what they give, each with its label; a timestamp
// offers the elements of its format. The screens whose fields the gateway
// substitutes offer it (email templates, tasks of the rules; #812). The
// reserved keys of the transfer information (model.ReservedInfoKeys) are
// offered by family, next to the free key (#816).

const (
	// ListID is the id of the variables list the fields offer (data-vars).
	ListID = "vars-list"

	// The marker of a transfer info: its key follows the prefix.
	transferInfoName = "TI_"

	// The format a timestamp marker uses without one; inserted selected, to be
	// replaced.
	defaultFormat = "YYYY-MM-DD_HHmmss"

	// Prefixes of the catalogue keys chosen at run time.
	varKeyPrefix   = "vars.var."
	groupKeyPrefix = "vars.group."
	tokenKeyPrefix = "vars.token."
	infoKeyPrefix  = "vars.info."
)

// varGroups orders the markers of the tasks in the list, by what they give.
// The markers come from the tasks (tasks.Markers, tasks.FormatMarkers): a test
// checks that every marker they implement is here, but a duplicate
// (#ERRORSTRCODE# gives the error message, as #ERRORMSG#) and a deprecated one
// (#FULLTRANSFERID#), which stay accepted when typed by hand.
//
//nolint:gochecknoglobals // reference data of the screen
var varGroups = []struct {
	key   string
	names []string
}{
	{"file", []string{
		"TRUEFILENAME", "BASEFILENAME", "FILEEXTENSION", "FILESIZE", "ORIGINALFILENAME", "TRUEFULLPATH", "ORIGINALFULLPATH",
	}},
	{"transfer", []string{"RULE", "TRANSFERID", "REMOTETRANSFERID"}},
	{"partners", []string{"REMOTEHOST", "LOCALHOST", "REQUESTERHOST", "REQUESTEDHOST"}},
	{"dates", []string{"DATE", "HOUR", "TIMESTAMP", "STARTTIMESTAMP"}},
	{"error", []string{"ERRORMSG", "ERRORCODE"}},
	{"paths", []string{"HOMEPATH", "INPATH", "OUTPATH", "WORKPATH"}},
	{"info", []string{transferInfoName}},
}

// tokenKeys names the elements of a date format in the catalogue (the
// tokens differ by case only, the keys do not).
//
//nolint:gochecknoglobals // reference data of the screen
var tokenKeys = map[string]string{
	"YYYY": "year", "YY": "year_short", "MMMM": "month_name", "MMM": "month_abbr", "MM": "month", "M": "month_short",
	"DD": "day", "D": "day_short", "dddd": "weekday", "ddd": "weekday_abbr",
	"HH": "hour", "hh": "hour12", "h": "hour12_short", "PM": "ampm", "pm": "ampm_lower",
	"mm": "minute", "m": "minute_short", "ss": "second", "s": "second_short",
	"tz": "zone", "zz": "offset", "z": "offset_short",
}

// tokensNotOffered are elements of a date format the list does not offer:
// "pm" yields "p" and the minutes, the "m" of the minutes being replaced
// after it (a test checks it, so that a correction offers it again).
//
//nolint:gochecknoglobals // reference data of the screen
var tokensNotOffered = []string{"pm"}

// infoFamilyGroups names the group of the list that offers the reserved keys
// of each family: the keys of the transfer follow the free key in its group,
// the others have a group of their own.
//
//nolint:gochecknoglobals // reference data of the screen
var infoFamilyGroups = map[model.InfoKeyFamily]string{
	model.InfoFamilyTransfer: "info", model.InfoFamilyPesit: "info_pesit", model.InfoFamilyAck: "info_ack",
}

// List builds the variables list: the markers the tasks implement, by
// group, each with its label; a timestamp offers the elements of its format
// with the value they give at the given time.
func List(translate gui.Translator, now time.Time) gui.VarListProps {
	formats := tasks.FormatMarkers()
	tokens := formatTokens(translate, now)

	groups := make([]gui.VarGroup, 0, len(varGroups))

	for _, g := range varGroups {
		group := gui.VarGroup{Label: translate(groupKeyPrefix + g.key)}

		for _, name := range g.names {
			label := translate(varKeyPrefix + strings.ToLower(strings.TrimSuffix(name, "_")))

			switch {
			case name == transferInfoName:
				param := translate("vars.var.ti_param")
				group.Vars = append(group.Vars, gui.Var{Name: name + param, Label: label, Param: param})
			case slices.Contains(formats, name):
				group.Vars = append(group.Vars, gui.Var{
					Name: name + "(" + defaultFormat + ")", Label: label, Param: defaultFormat, Tokens: tokens,
				})
			default:
				group.Vars = append(group.Vars, gui.Var{Name: name, Label: label})
			}
		}

		if g.key == infoFamilyGroups[model.InfoFamilyTransfer] {
			group.Vars = append(group.Vars, reservedKeys(translate, model.InfoFamilyTransfer)...)
		}

		groups = append(groups, group)
	}

	for _, family := range []model.InfoKeyFamily{model.InfoFamilyPesit, model.InfoFamilyAck} {
		groups = append(groups, gui.VarGroup{
			Label: translate(groupKeyPrefix + infoFamilyGroups[family]), Vars: reservedKeys(translate, family),
		})
	}

	return gui.VarListProps{ID: ListID, Groups: groups}
}

// reservedKeys offers the reserved keys of a family (model.ReservedInfoKeys),
// each as #TI___key__# with its label.
func reservedKeys(translate gui.Translator, family model.InfoKeyFamily) []gui.Var {
	keys := model.ReservedInfoKeys()
	out := make([]gui.Var, 0, len(keys))

	for _, k := range keys {
		if k.Family == family {
			out = append(out, gui.Var{Name: transferInfoName + k.Name, Label: translate(InfoKeyLabel(k.Name))})
		}
	}

	return out
}

// InfoKeyLabel is the catalogue key of the label of a reserved key of the
// transfer information: __clientConnFreetext__ gives
// vars.info.client_conn_freetext, __followID__ gives vars.info.follow_id.
func InfoKeyLabel(name string) string {
	var b strings.Builder

	runes := []rune(strings.Trim(name, "_"))
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			b.WriteByte('_')
		}

		b.WriteRune(unicode.ToLower(r))
	}

	return infoKeyPrefix + b.String()
}

// formatTokens are the elements of a date format the list offers, each with
// the value the tasks give it at the given time.
func formatTokens(translate gui.Translator, now time.Time) []gui.VarToken {
	codes := tasks.TimestampTokens()
	out := make([]gui.VarToken, 0, len(codes))

	for _, code := range codes {
		if slices.Contains(tokensNotOffered, code) {
			continue
		}

		out = append(out, gui.VarToken{
			Code:    code,
			Label:   translate(tokenKeyPrefix + tokenKeys[code]),
			Example: tasks.FormatTimestamp("("+code+")", now),
		})
	}

	return out
}
