// Package packaging validates the nFPM manifest from which the Debian and RPM
// packages are built. The checks it exposes exist because each of them maps to
// a defect that shipped to users and that no linter would have caught: see
// issue #387.
package packaging

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	canonicalUnitDir = "/usr/lib/systemd/system"
	usrSharePrefix   = "/usr/share/"
	buildPrefix      = "./build/"

	// Octal literals run from 0644 to 07777.
	minOctalLen = 4
	maxOctalLen = 5
)

// aliasedPrefixes are the top-level directories that are symlinks into /usr on
// a merged-/usr system. Shipping a file through one of them is a DEP-17
// violation and lintian reports it as "aliased-location".
//
//nolint:gochecknoglobals // a package-level lookup table, never mutated
var aliasedPrefixes = []string{"/lib/", "/bin/", "/sbin/", "/lib64/"}

//nolint:gochecknoglobals // a package-level lookup table, never mutated
var unitSuffixes = []string{".service", ".timer", ".socket", ".path", ".target"}

// Manifest is the subset of the nFPM configuration that the packaging checks
// need. Fields nFPM understands but that no check looks at are left out.
type Manifest struct {
	Contents  []Content           `yaml:"contents"`
	Overrides map[string]Override `yaml:"overrides"`
}

// Override is a per-packager section. A section that never mentions "contents"
// leaves the slice nil, whereas "contents: []" yields a non-nil empty one --
// and only the latter shadows the top-level entries.
type Override struct {
	Contents []Content `yaml:"contents"`
}

// Content is a single packaged file, directory or configuration entry.
type Content struct {
	Src      string `yaml:"src"`
	Dst      string `yaml:"dst"`
	Type     string `yaml:"type"`
	Packager string `yaml:"packager"`
	//nolint:tagliatelle // "file_info" is nFPM's own key and cannot be renamed
	FileInfo *FileInfo `yaml:"file_info"`
}

// FileInfo holds the ownership and mode of an entry. Mode is kept as a raw
// node because the defect being guarded against is one of notation, not of
// value: nFPM reads an undecorated "755" as decimal and silently produces mode
// 0o1363, which an int field would hide.
type FileInfo struct {
	Mode yaml.Node `yaml:"mode"`
}

// Load reads and parses the nFPM manifest at path.
func Load(path string) (*Manifest, error) {
	raw, readErr := os.ReadFile(filepath.Clean(path))
	if readErr != nil {
		return nil, fmt.Errorf("cannot read the nFPM manifest: %w", readErr)
	}

	manifest := &Manifest{}
	if parseErr := yaml.Unmarshal(raw, manifest); parseErr != nil {
		return nil, fmt.Errorf("cannot parse the nFPM manifest: %w", parseErr)
	}

	return manifest, nil
}

// ContentOverrides returns the packagers whose override section redefines
// "contents", shadowing the top-level list rather than extending it.
func (m *Manifest) ContentOverrides() []string {
	found := []string{}

	for packager, override := range m.Overrides {
		if override.Contents != nil {
			found = append(found, packager)
		}
	}

	slices.Sort(found)

	return found
}

// NonOctalModes returns the destinations whose mode is not written as an
// unquoted octal literal with a leading zero.
func (m *Manifest) NonOctalModes() []string {
	return m.destinations(func(entry Content) bool {
		if entry.FileInfo == nil || entry.FileInfo.Mode.Value == "" {
			return false // reported by EntriesWithoutMode instead
		}

		return entry.FileInfo.Mode.Tag != "!!int" ||
			!isOctalLiteral(entry.FileInfo.Mode.Value)
	})
}

// EntriesWithoutMode returns the destinations that declare no mode and would
// therefore inherit whatever mode the source file happens to carry on the
// build machine.
func (m *Manifest) EntriesWithoutMode() []string {
	return m.destinations(func(entry Content) bool {
		return entry.FileInfo == nil || entry.FileInfo.Mode.Value == ""
	})
}

// AliasedUnitPaths returns the destinations installed through a /usr-merge
// alias, plus any systemd unit installed outside the canonical unit directory.
func (m *Manifest) AliasedUnitPaths() []string {
	return m.destinations(func(entry Content) bool {
		for _, prefix := range aliasedPrefixes {
			if strings.HasPrefix(entry.Dst, prefix) {
				return true
			}
		}

		return isSystemdUnit(entry.Dst) && !strings.HasPrefix(entry.Dst, canonicalUnitDir)
	})
}

// isSystemdUnit covers every unit suffix, not just ".service": the timer was
// the one shipped through an aliased path that a ".service"-only test would
// have caught by accident rather than by design.
func isSystemdUnit(dst string) bool {
	for _, suffix := range unitSuffixes {
		if strings.HasSuffix(dst, suffix) {
			return true
		}
	}

	return false
}

// BinariesUnderUsrShare returns the compiled artefacts installed under
// /usr/share, which is reserved for architecture-independent data.
func (m *Manifest) BinariesUnderUsrShare() []string {
	return m.destinations(func(entry Content) bool {
		return strings.HasPrefix(entry.Dst, usrSharePrefix) && isCompiledArtefact(entry.Src)
	})
}

// MissingSources returns the sources that do not exist under root. Build
// artefacts are skipped: they only exist once "make.sh build dist" has run.
func (m *Manifest) MissingSources(root string) []string {
	missing := []string{}

	for _, entry := range m.allContents() {
		if entry.Src == "" || strings.HasPrefix(entry.Src, buildPrefix) {
			continue
		}

		if _, err := os.Stat(filepath.Join(root, entry.Src)); err != nil {
			missing = append(missing, entry.Src)
		}
	}

	slices.Sort(missing)

	return slices.Compact(missing)
}

// destinations returns the sorted destinations of every entry matching keep.
func (m *Manifest) destinations(keep func(Content) bool) []string {
	found := []string{}

	for _, entry := range m.allContents() {
		if keep(entry) {
			found = append(found, entry.Dst)
		}
	}

	slices.Sort(found)

	return slices.Compact(found)
}

// allContents flattens the top-level entries and every override section, so
// that a check cannot pass simply because the offending entry was hidden in an
// override.
func (m *Manifest) allContents() []Content {
	all := slices.Clone(m.Contents)

	for _, override := range m.Overrides {
		all = append(all, override.Contents...)
	}

	return all
}

// isCompiledArtefact reports whether src is one of the binaries make.sh builds
// into build/. Those have no file extension, which is what tells them apart
// from the generated build/waarp-gatewayd.ini.
func isCompiledArtefact(src string) bool {
	return strings.HasPrefix(src, buildPrefix) && filepath.Ext(src) == ""
}

// isOctalLiteral reports whether value is written the way nFPM needs it: a
// leading zero followed by octal digits only.
func isOctalLiteral(value string) bool {
	if len(value) < minOctalLen || len(value) > maxOctalLen || value[0] != '0' {
		return false
	}

	for _, digit := range value[1:] {
		if digit < '0' || digit > '7' {
			return false
		}
	}

	return true
}
