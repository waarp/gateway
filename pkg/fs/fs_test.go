package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsLocal(t *testing.T) {
	t.Parallel()

	type testCase struct {
		path     string
		expected bool
	}

	testCases := []testCase{
		{`/absolute/local/path.txt`, true},
		{`/absolute_local_file.txt`, true},
		{`relative/local/path.txt`, true},
		{`relative_local_file.txt`, true},
		{`file:/absolute/local/path.txt`, true},
		{`file:/absolute/local/path.txt`, true},
		{`file:relative/local/path.txt`, true},
		{`file:relative_local_file.txt`, true},
		{`remote:/absolute/remote/path.txt`, false},
		{`remote:/absolute/remote_file.txt`, false},
		{`remote:relative/remote/path.txt`, false},
		{`remote:relative/remote_file.txt`, false},
	}

	if runtime.GOOS == `windows` {
		testCases = append(testCases,
			testCase{`C:\absolute\local\path.txt`, true},
			testCase{`C:/absolute/local/path.txt`, true},
			testCase{`C:\absolute_local_file.txt`, true},
			testCase{`C:/absolute_local_file.txt`, true},
			testCase{`relative\local\path.txt`, true},
		)
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, IsLocalPath(tc.path))
		})
	}
}

func TestIsAbs(t *testing.T) {
	t.Parallel()

	type testCase struct {
		path     string
		expected bool
	}

	testCases := []testCase{
		{`relative/local/path.txt`, false},
		{`relative_local_file.txt`, false},
		{`file:/absolute/local/path.txt`, true},
		{`file:/absolute/local/path.txt`, true},
		{`file:relative/local/path.txt`, true},
		{`file:relative_local_file.txt`, true},
		{`remote:/absolute/remote/path.txt`, true},
		{`remote:/absolute/remote_file.txt`, true},
		{`remote:relative/remote/path.txt`, true},
		{`remote:relative/remote_file.txt`, true},
	}

	if runtime.GOOS != `windows` {
		testCases = append(testCases,
			testCase{`/absolute/local/path.txt`, true},
			testCase{`/absolute_local_file.txt`, true},
		)
	} else {
		testCases = append(testCases,
			testCase{`C:\absolute\local\path.txt`, true},
			testCase{`C:/absolute/local/path.txt`, true},
			testCase{`C:\absolute_local_file.txt`, true},
			testCase{`C:/absolute_local_file.txt`, true},
			testCase{`relative\local\path.txt`, false},
			testCase{`/unix/absolute/path.txt`, false},
			testCase{`\unix\absolute\path.txt`, false},
			testCase{`\\unc\absolute\path.txt`, true},
			testCase{`//unc/absolute/path.txt`, true},
		)
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, IsAbsPath(tc.path))
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	type testCase struct {
		path    string
		expName string
		expPath string
	}

	testCases := []testCase{
		{`/absolute/local/path.txt`, ``, `/absolute/local/path.txt`},
		{`relative/local/path.txt`, ``, `relative/local/path.txt`},
		{`file:/absolute/local/path.txt`, ``, `/absolute/local/path.txt`},
		{`file:relative/local/path.txt`, ``, `relative/local/path.txt`},
		{`remote:/absolute/remote/path.txt`, `remote`, `/absolute/remote/path.txt`},
		{`remote:relative/remote/path.txt`, `remote`, `relative/remote/path.txt`},
	}

	if runtime.GOOS == `windows` {
		testCases = append(testCases,
			testCase{`C:\\absolute\local\path.txt`, ``, `C://absolute/local/path.txt`},
			testCase{`C://absolute/local/path.txt`, ``, `C://absolute/local/path.txt`},
			testCase{`relative\local\path.txt`, ``, `relative/local/path.txt`},
			testCase{`\\unc\absolute\path.txt`, ``, `//unc/absolute/path.txt`},
			testCase{`//unc/absolute/path.txt`, ``, `//unc/absolute/path.txt`},
		)
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			parsed, err := parsePath(tc.path)
			require.NoError(t, err)
			assert.Equal(t, tc.expName, parsed.Name)
			assert.Equal(t, tc.expPath, parsed.Path)
		})
	}
}

func TestGetFileTime(t *testing.T) {
	t.Parallel()

	const timeDelta = 10 * time.Millisecond

	readFile := func(path string) error {
		_, err := os.ReadFile(path)
		return err
	}
	writeFile := func(path string) error {
		return os.WriteFile(path, []byte(`hello world`), 0644)
	}

	doesChangeTime := func(tb testing.TB, path string, op func(string) error,
		getTime func(string) (time.Time, error)) {
		tb.Helper()

		before, err := getTime(path)
		require.NoError(tb, err)

		<-time.After(timeDelta)
		require.NoError(tb, op(path))
		after := time.Now()

		ts, err := getTime(path)
		require.NoError(tb, err)
		assert.NotEqual(tb, before, ts)
		assert.WithinRange(tb, ts, before, after)
	}

	doesNotChangeTime := func(tb testing.TB, path string, op func(string) error,
		getTime func(string) (time.Time, error)) {
		tb.Helper()

		before, err := getTime(path)
		require.NoError(tb, err)

		<-time.After(timeDelta)
		require.NoError(tb, op(path))

		ts, err := getTime(path)
		require.NoError(tb, err)
		assert.Equal(tb, before, ts)
	}

	t.Run("GetFileATime", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "test.txt")
		require.NoError(t, writeFile(path))
		doesChangeTime(t, path, readFile, GetFileATime)
	})

	t.Run("GetFileMTime", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "test.txt")
		require.NoError(t, writeFile(path))
		doesChangeTime(t, path, writeFile, GetFileMTime)
		doesNotChangeTime(t, path, readFile, GetFileMTime)
	})

	t.Run("GetFileBTime", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "test.txt")
		require.NoError(t, writeFile(path))
		doesNotChangeTime(t, path, writeFile, GetFileBTime)
		doesNotChangeTime(t, path, readFile, GetFileBTime)
	})
}
