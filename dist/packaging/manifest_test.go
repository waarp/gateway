package packaging_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/dist/packaging"
)

const (
	// manifestPath and repoRoot are relative to this package's directory,
	// which is the working directory "go test" uses.
	manifestPath = "../nfpm.yaml"
	repoRoot     = "../.."
)

func loadManifest(tb testing.TB) *packaging.Manifest {
	tb.Helper()

	manifest, err := packaging.Load(manifestPath)
	require.NoError(tb, err)

	return manifest
}

// An "overrides.<format>.contents" block REPLACES the top-level "contents"
// entirely instead of extending it. Keeping a single list and tagging the
// format-specific entries with "packager" is the only layout that cannot
// silently drop a file, which is how example_config.yaml went unshipped for
// years (issue #387).
func TestOverridesDoNotRedefineContents(t *testing.T) {
	t.Parallel()

	assert.Empty(t, loadManifest(t).ContentOverrides(),
		"overrides must not redefine 'contents': tag entries with 'packager' instead")
}

// nFPM reads an undecorated "755" as decimal, yielding mode 0o1363, and exits 0
// while doing it. Only a leading-zero octal literal is safe.
func TestEveryFileModeIsAnOctalLiteral(t *testing.T) {
	t.Parallel()

	assert.Empty(t, loadManifest(t).NonOctalModes(),
		"file_info.mode must be an unquoted octal literal with a leading zero, e.g. 0644")
}

// A mode inherited from the source file's on-disk mode is whatever the build
// machine happened to have: gatewayd.ini shipped 0600 in 0.12.9 and 0644 in
// 0.16.0 from the very same manifest.
func TestEveryEntryDeclaresItsMode(t *testing.T) {
	t.Parallel()

	assert.Empty(t, loadManifest(t).EntriesWithoutMode(),
		"every content entry must declare file_info.mode rather than inherit it")
}

// Catches the class of typo fixed by a7a9a09e, where a src pointed at
// ./build/dist/example_config.yaml instead of ./dist/example_config.yaml.
// Build artefacts are skipped: they only exist after "make.sh build dist".
func TestEveryCheckedInSourceExists(t *testing.T) {
	t.Parallel()

	assert.Empty(t, loadManifest(t).MissingSources(repoRoot),
		"a content 'src' points at a file that is not in the repository")
}

// /lib/systemd/system is an aliased path on merged-/usr systems and lintian
// rejects it (DEP-17). Both packagers must use /usr/lib/systemd/system.
func TestSystemdUnitsUseTheCanonicalDirectory(t *testing.T) {
	t.Parallel()

	assert.Empty(t, loadManifest(t).AliasedUnitPaths(),
		"systemd units must be installed under /usr/lib/systemd/system")
}

// /usr/share is for architecture-independent data. The helper binaries belong
// in the packager's libexec directory: /usr/libexec on RPM distributions,
// /usr/lib on Debian, where /usr/libexec is only allowed from Policy 4.7.0.
func TestNoArchDependentBinaryUnderUsrShare(t *testing.T) {
	t.Parallel()

	assert.Empty(t, loadManifest(t).BinariesUnderUsrShare(),
		"compiled helpers must not be installed under /usr/share")
}
