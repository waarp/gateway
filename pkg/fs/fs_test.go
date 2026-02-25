package fs

import (
	"runtime"
	"testing"

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
