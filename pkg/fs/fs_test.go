package fs

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocal(t *testing.T) {
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
			assert.Equal(t, tc.expected, IsLocalPath(tc.path))
		})
	}
}
