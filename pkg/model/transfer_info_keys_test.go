package model

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The registry is the only place that spells a reserved key: every other
// package of the gateway names it from here (#816).
func TestReservedInfoKeysAreTheOnlySource(t *testing.T) {
	t.Parallel()

	keys := ReservedInfoKeys()
	names := make([]string, 0, keys.Len())

	for fam, key := range keys.Iter() {
		assert.Regexp(t, `^__[a-z][A-Za-z0-9]*__$`, key.Name)
		assert.NotContainsf(t, names, key.Name, "%s is listed twice", key.Name)
		assert.NotZerof(t, fam, "%s has no family", key.Name)
		assert.NotZerof(t, key.SetBy, "%s says not who sets it", key.Name)

		names = append(names, key.Name)
	}

	// The markers of the redacted secrets of an export (__PWD__…) have an
	// upper-case first letter: they are not keys of the transfer information.
	literal := regexp.MustCompile(`"(__[a-z][A-Za-z0-9]*__)"`)
	spelled := map[string][]string{}

	require.NoError(t, filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		for _, m := range literal.FindAllStringSubmatch(string(content), -1) {
			spelled[m[1]] = append(spelled[m[1]], filepath.ToSlash(path))
		}

		return nil
	}))

	for name, files := range spelled {
		assert.Containsf(t, names, name, "%s is spelled in %v but not in the registry", name, files)
		assert.Equalf(t, []string{"../model/transfer_info_keys.go"}, files,
			"%s must be spelled in the registry only, not in %v", name, files)
	}

	for _, name := range names {
		assert.Containsf(t, slices.Collect(mapKeys(spelled)), name, "%s is in the registry but spelled nowhere", name)
	}
}

func mapKeys(m map[string][]string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func TestReservedInfoKeyUserSets(t *testing.T) {
	t.Parallel()

	assert.True(t, ReservedInfoKey{SetBy: InfoSetByUser}.UserSets())
	assert.True(t, ReservedInfoKey{SetBy: InfoSetByBoth}.UserSets())
	assert.False(t, ReservedInfoKey{SetBy: InfoSetByGateway}.UserSets())
}
