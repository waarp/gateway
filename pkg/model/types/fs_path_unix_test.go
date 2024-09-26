//go:build !windows

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePath(t *testing.T) {
	for _, test := range []struct {
		fullPath   string
		wantParsed FSPath
		wantErr    error
	}{
		{`a/b/c`, FSPath{``, "a/b/c"}, nil},
		{`file:a/b/c`, FSPath{``, "a/b/c"}, nil},
		{`/a/b/c`, FSPath{``, `/a/b/c`}, nil},
		{`file:/a/b/c`, FSPath{``, `/a/b/c`}, nil},
	} {
		t.Run(test.fullPath, func(t *testing.T) {
			gotParsed, gotErr := ParsePath(test.fullPath)

			if test.wantErr != nil {
				assert.ErrorIs(t, gotErr, test.wantErr)
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, test.wantParsed, *gotParsed, test.fullPath)
			}
		})
	}
}

func TestToFSPath(t *testing.T) {
	for _, test := range []struct {
		parsed     FSPath
		wantFSPath string
	}{
		{FSPath{``, `a/b/c`}, `a/b/c`},
		{FSPath{``, `/a/b/c`}, `a/b/c`},
	} {
		t.Run(test.parsed.String(), func(t *testing.T) {
			gotFSPath := test.parsed.FSPath()
			assert.Equal(t, test.wantFSPath, gotFSPath)
		})
	}
}
