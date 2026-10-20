package clouds

import (
	"slices"
	"strings"

	rclonefs "github.com/rclone/rclone/fs"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

// What the storage library says of the options of a storage type, read in
// its registry through the description the backend gives of itself
// (fs.LibraryOf): the options it knows, the ones it masks (secrets and
// identifiers), what it applies when one is absent (the product applies it
// too, #701), the values it proposes. The "other options" zone is checked
// against it, and lists the options it offers.

const (
	// The library filters the S3 options by provider; the gateway sets
	// none, the list shows the options of Amazon S3.
	listedProvider = "AWS"

	// A row of the list shows the values the library proposes when there
	// are few enough of them to be read in a cell.
	maxValues = 12

	// The name of the library in its help texts gives way to libraryWord:
	// the interface names no third-party product.
	libraryName = "rclone"
	libraryWord = "the storage library"
)

// libOption is an option of the storage library.
type libOption struct {
	Key       string
	Default   string   // what the library applies when the option is absent, "" for nothing
	Help      string   // the first line of the help of the library, in English
	Values    []string // the values the library proposes, when they are few
	Sensitive bool     // the library masks it: a secret or an identifier
	Listed    bool     // offered in the list of the "other options"
}

// library is what the storage library says of a storage type.
type library struct {
	options map[string]*libOption
}

// libraryOf reads the library of a storage type; false when the backend did
// not describe it or the library does not know it (the screen then checks
// nothing more than the form fields).
func libraryOf(s *storage) (*library, bool) {
	desc, ok := fs.LibraryOf(s.Key)
	if !ok {
		return nil, false
	}

	info, err := rclonefs.Find(desc.Name)
	if err != nil {
		return nil, false
	}

	lib := &library{options: map[string]*libOption{}}

	for i := range info.Options {
		o := &info.Options[i]
		if _, seen := lib.options[o.Name]; seen {
			continue
		}

		opt := &libOption{
			Key: o.Name, Default: o.String(), Help: helpLine(o.Help), Values: proposed(o),
			Sensitive: o.IsPassword || o.Sensitive,
		}
		opt.Listed = !opt.Sensitive && !obsolete(o.Help) && s.offers(o.Name) &&
			(o.Provider == "" || rclonefs.MatchProvider(o.Provider, listedProvider))
		lib.options[o.Name] = opt
	}

	return lib, true
}

// offers tells whether an option of the library belongs to the "other
// options": not one the form names, nor a credential, nor the token the
// product manages.
func (s *storage) offers(key string) bool {
	return !s.knows(key) && key != optEnvAuth && !isSecret(key)
}

// knows tells whether the library knows the option.
func (l *library) knows(key string) bool {
	_, ok := l.options[key]

	return ok
}

// sensitive tells whether the library masks the option.
func (l *library) sensitive(key string) bool {
	o, ok := l.options[key]

	return ok && o.Sensitive
}

// listed returns the options offered in the list, by key.
func (l *library) listed() []*libOption {
	var list []*libOption

	for _, o := range l.options {
		if o.Listed {
			list = append(list, o)
		}
	}

	slices.SortFunc(list, func(a, b *libOption) int { return strings.Compare(a.Key, b.Key) })

	return list
}

// helpLine is the first line of a help text of the library, its own name
// replaced.
func helpLine(help string) string {
	line, _, _ := strings.Cut(help, "\n")

	return strings.ReplaceAll(strings.TrimSpace(line), libraryName, libraryWord)
}

// obsolete tells whether the library says the option is no longer used.
func obsolete(help string) bool {
	help = strings.ToLower(help)

	return strings.Contains(help, "no longer used") || strings.Contains(help, "deprecated")
}

// proposed returns the values the library proposes for the option (for the
// listed provider), when they are few enough; nil otherwise.
func proposed(o *rclonefs.Option) []string {
	var values []string

	for _, ex := range o.Examples {
		if ex.Value == "" || slices.Contains(values, ex.Value) {
			continue
		}

		if ex.Provider == "" || rclonefs.MatchProvider(ex.Provider, listedProvider) {
			values = append(values, ex.Value)
		}
	}

	if len(values) > maxValues {
		return nil
	}

	return values
}
