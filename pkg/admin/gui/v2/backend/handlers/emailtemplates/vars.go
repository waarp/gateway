package emailtemplates

import (
	"slices"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

const (
	// The id of the variables list the fields of the form offer.
	varListID = "email-template-vars"

	// The marker of a transfer info: its key follows the prefix.
	transferInfoName = "TI_"

	// The format a timestamp marker uses without one; inserted selected, to be
	// replaced.
	defaultFormat = "YYYY-MM-DD_HHmmss"

	// Prefixes of the catalogue keys chosen at run time.
	varKeyPrefix   = "email_templates.var."
	groupKeyPrefix = "email_templates.var_group."
	tokenKeyPrefix = "email_templates.token."
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

// varList builds the variables list of the form: the markers the tasks
// implement, by group, each with its label; a timestamp offers the elements
// of its format with the value they give at the given time.
func varList(translate gui.Translator, now time.Time) gui.VarListProps {
	formats := tasks.FormatMarkers()
	tokens := formatTokens(translate, now)

	groups := make([]gui.VarGroup, 0, len(varGroups))

	for _, g := range varGroups {
		group := gui.VarGroup{Label: translate(groupKeyPrefix + g.key)}

		for _, name := range g.names {
			label := translate(varKeyPrefix + strings.ToLower(strings.TrimSuffix(name, "_")))

			switch {
			case name == transferInfoName:
				param := translate("email_templates.var.ti_param")
				group.Vars = append(group.Vars, gui.Var{Name: name + param, Label: label, Param: param})
			case slices.Contains(formats, name):
				group.Vars = append(group.Vars, gui.Var{
					Name: name + "(" + defaultFormat + ")", Label: label, Param: defaultFormat, Tokens: tokens,
				})
			default:
				group.Vars = append(group.Vars, gui.Var{Name: name, Label: label})
			}
		}

		groups = append(groups, group)
	}

	return gui.VarListProps{ID: varListID, Groups: groups}
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
