package rules

import (
	"cmp"
	"net/http"
	"strings"
	"sync"
	"unicode"

	"code.waarp.fr/lib/gui"
	"code.waarp.fr/lib/transcode"

	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The TRANSCODE task asks the direction of the conversion first (Driss,
// 2026-11-05): ASCII stands for the character sets of open systems, UTF-8
// among them, EBCDIC for the code pages of IBM hosts. Each list then offers
// the character sets of its side only, in groups, each with what it is for.
// The direction is not stored: a saved task gives it by its two character
// sets, so that the API, the command line and the existing rules do not
// change. Without JavaScript, the four lists are shown and the one of the
// direction chosen is read.

const (
	sideASCII  = "ascii"
	sideEBCDIC = "ebcdic"

	// The fields of the encoding card, after the type: the direction, and a
	// list per argument and side ("fromCharset_ebcdic").
	encodingDirection = "direction"
	encodingFrom      = "fromCharset"
	encodingTo        = "toCharset"

	charsetKeyPrefix         = "task_types.transcode.charset."
	charsetGroupKeyPrefix    = "task_types.transcode.charset_group."
	encodingDirectionsPrefix = "task_types.transcode.direction."
)

// encodingDirections are the directions of a conversion, "source-target".
func encodingDirections() []string {
	return []string{
		sideASCII + "-" + sideEBCDIC, sideEBCDIC + "-" + sideASCII,
		sideASCII + "-" + sideASCII, sideEBCDIC + "-" + sideEBCDIC,
	}
}

// sidesOf reads a direction: the side of the source, the side of the target.
func sidesOf(direction string) (from, to string, ok bool) {
	from, to, _ = strings.Cut(direction, "-")
	isSide := func(s string) bool { return s == sideASCII || s == sideEBCDIC }

	return from, to, isSide(from) && isSide(to)
}

// charsetKey is the key of a character set in the catalogue, after its name
// ("IBM Code Page 297" gives "ibm_code_page_297").
func charsetKey(name string) string {
	var key strings.Builder

	for _, r := range strings.ToLower(name) {
		switch {
		case 'a' <= r && r <= 'z', '0' <= r && r <= '9':
			key.WriteRune(r)
		case key.Len() > 0 && !strings.HasSuffix(key.String(), "_"):
			key.WriteByte('_')
		}
	}

	return strings.TrimSuffix(key.String(), "_")
}

// charsetItem is a character set offered by the form, with its group.
type charsetItem struct {
	name       string
	group      string
	deprecated bool
}

// The groups of the lists, in their order.
const (
	groupUnicode     = "unicode"
	groupISO8859     = "iso8859"
	groupWindows     = "windows"
	groupPC          = "pc"
	groupAsian       = "asian"
	groupEBCDICLatin = "ebcdic_latin"
	groupEBCDICOther = "ebcdic_other"
)

func charsetGroupOrder() []string {
	return []string{groupUnicode, groupISO8859, groupWindows, groupPC, groupAsian, groupEBCDICLatin, groupEBCDICOther}
}

// charsetSides are the character sets of each side, in the order of the
// registry, each in its group; built once, the built-in character sets do not
// change.
//
//nolint:gochecknoglobals //computed once from the registry, then only read
var charsetSides = sync.OnceValue(func() map[string][]charsetItem {
	registry := tasks.TranscodeRegistry()
	sides := map[string][]charsetItem{}

	for _, entry := range registry.List() {
		if entry.Kind == transcode.KindTable {
			continue
		}

		side, group := sideASCII, asciiGroup(&entry)
		if entry.EBCDIC {
			side, group = sideEBCDIC, ebcdicGroup(registry, entry.Name)
		}

		sides[side] = append(sides[side], charsetItem{name: entry.Name, group: group, deprecated: entry.Deprecated})
	}

	return sides
})

// asciiGroup is the group of a character set of open systems.
func asciiGroup(entry *transcode.Entry) string {
	switch {
	case entry.Kind == transcode.KindUnicode:
		return groupUnicode
	case entry.Kind == transcode.KindMultiByte:
		return groupAsian
	case strings.HasPrefix(entry.Name, "ISO"):
		return groupISO8859
	case strings.HasPrefix(entry.Name, "Windows ") && !strings.Contains(entry.Name, "Code Page"):
		return groupWindows
	default:
		return groupPC // the code pages of PCs (DOS), of the Macintosh, KOI8
	}
}

// ebcdicGroup is the group of an EBCDIC page: the Latin alphabet when all its
// letters are Latin ones, other alphabets otherwise.
func ebcdicGroup(registry *transcode.Registry, name string) string {
	page, err := registry.CodePage(name)
	if err != nil {
		return groupEBCDICOther
	}

	for _, r := range page.Chars() {
		if unicode.IsLetter(r) && !unicode.In(r, unicode.Latin, unicode.Common, unicode.Inherited) {
			return groupEBCDICOther
		}
	}

	return groupEBCDICLatin
}

// charsetSide is the side of a character set; an unknown one is taken for
// ASCII.
func charsetSide(name string) string {
	if entry, ok := tasks.TranscodeRegistry().Lookup(name); ok && entry.EBCDIC {
		return sideEBCDIC
	}

	return sideASCII
}

// canonicalCharset is the name of a character set given by one of its
// aliases; an unknown name stays as it is.
func canonicalCharset(name string) string {
	if entry, ok := tasks.TranscodeRegistry().Lookup(name); ok {
		return entry.Name
	}

	return name
}

// encodingInput is the encoding of a TRANSCODE task: its direction and its
// two character sets, with their errors.
type encodingInput struct {
	direction, from, to    string
	dirErr, fromErr, toErr string
}

// encodingOfStored is the encoding of a stored task, its direction given by
// its two character sets.
func encodingOfStored(stored map[string]string) encodingInput {
	from, to := stored[encodingFrom], stored[encodingTo]
	if from == "" && to == "" {
		return encodingInput{}
	}

	from, to = canonicalCharset(from), canonicalCharset(to)

	return encodingInput{direction: charsetSide(from) + "-" + charsetSide(to), from: from, to: to}
}

// encodingView is the encoding card of a TRANSCODE task: the direction, then
// the lists of the source and of the target.
type encodingView struct {
	Slug       string
	Directions []taskOption
	DirErr     string
	From, To   encodingSide
}

// encodingSide is an argument of the encoding card: its texts, and its list
// for each side, shown for the directions where it applies.
type encodingSide struct {
	ID, Label, Hint, Err, None string
	Lists                      []charsetList
}

// charsetList is the list of a side, shown for its directions (data-case).
type charsetList struct {
	Cases    string
	ID, Name string
	Groups   []taskGroup
}

// encodingOf builds the encoding card of a TRANSCODE task with what the form
// holds; an error of the product under an argument goes under its list.
func encodingOf(translate gui.Translator, spec *taskSpec, held *encodingInput, errs map[string]string) *encodingView {
	view := &encodingView{Slug: spec.Slug, DirErr: held.dirErr}

	for _, dir := range encodingDirections() {
		label := translate(encodingDirectionsPrefix + strings.ReplaceAll(dir, "-", "_"))
		view.Directions = append(view.Directions, taskOption{Value: dir, Label: label, Selected: held.direction == dir})
	}

	from, to, _ := sidesOf(held.direction)
	view.From = encodingSideOf(translate, spec, encodingFrom, 0, from, held.from, cmp.Or(held.fromErr, errs[encodingFrom]))
	view.To = encodingSideOf(translate, spec, encodingTo, 1, to, held.to, cmp.Or(held.toErr, errs[encodingTo]))

	return view
}

// encodingSideOf builds an argument of the encoding card: a list per side,
// shown for the directions whose source (position 0) or target (1) is that
// side; the value held is selected in the list of its side.
func encodingSideOf(translate gui.Translator, spec *taskSpec, arg string, position int, heldSide, value, err string,
) encodingSide {
	view := encodingSide{
		ID: argID(spec, arg), Label: translate(argKey(spec, arg, "label")), Hint: translate(argKey(spec, arg, "hint")),
		Err: err, None: translate("task_types.choice.choose"),
	}

	for _, side := range []string{sideASCII, sideEBCDIC} {
		var cases []string

		for _, dir := range encodingDirections() {
			if sides := strings.Split(dir, "-"); sides[position] == side {
				cases = append(cases, dir)
			}
		}

		chosen := ""
		if side == heldSide {
			chosen = value
		}

		view.Lists = append(view.Lists, charsetList{
			Cases: strings.Join(cases, " "), ID: argID(spec, arg+"_"+side), Name: argName(spec, arg+"_"+side),
			Groups: withMissing(translate, charsetGroups(translate, side, chosen), chosen),
		})
	}

	return view
}

// charsetGroups are the character sets of a side, in their groups, each named
// with what it is for; a deprecated one is offered only when it is chosen.
func charsetGroups(translate gui.Translator, side, chosen string) []taskGroup {
	byGroup := map[string][]taskOption{}

	for _, item := range charsetSides()[side] {
		if item.deprecated && item.name != chosen {
			continue
		}

		byGroup[item.group] = append(byGroup[item.group], taskOption{
			Value:    item.name,
			Label:    item.name + " — " + translate(charsetKeyPrefix+charsetKey(item.name)),
			Selected: item.name == chosen,
		})
	}

	var groups []taskGroup

	for _, group := range charsetGroupOrder() {
		if options := byGroup[group]; len(options) > 0 {
			groups = append(groups, taskGroup{Label: translate(charsetGroupKeyPrefix + group), Options: options})
		}
	}

	return groups
}

// readEncoding reads the encoding card of a TRANSCODE task: the direction,
// then the character set chosen in the list of each side.
func readEncoding(translate gui.Translator, r *http.Request, spec *taskSpec, args map[string]string,
	input *taskInput,
) bool {
	held := &input.encoding
	required := translate("rules.tasks.error.required")
	held.direction = r.PostFormValue(argName(spec, encodingDirection))

	from, to, ok := sidesOf(held.direction)
	if !ok {
		held.direction, held.dirErr = "", translate("rules.tasks.error.sens")

		return false
	}

	held.from = strings.TrimSpace(r.PostFormValue(argName(spec, encodingFrom+"_"+from)))
	held.to = strings.TrimSpace(r.PostFormValue(argName(spec, encodingTo+"_"+to)))
	input.values[encodingFrom], input.values[encodingTo] = held.from, held.to
	valid := true

	if held.from == "" {
		held.fromErr, valid = required, false
	}

	if held.to == "" {
		held.toErr, valid = required, false
	}

	if valid {
		args[encodingFrom], args[encodingTo] = held.from, held.to
	}

	return valid
}
