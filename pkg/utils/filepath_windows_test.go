package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPath(t *testing.T) {
	testCases := []struct {
		path                  string
		roots                 []Elem
		wantBackend, wantPath string
		wantErr               error
	}{
		{`ressource`, []Elem{Branch(`C:\in`), Branch(`C:\out\test`)}, ``, `C:/in/ressource`, nil},
		{`ressource`, []Elem{Branch(`in`), Branch(`out\test`)}, ``, `out/test/in/ressource`, nil},
		{`ressource`, []Elem{Branch(`work`), Leaf(`out\test`)}, ``, `work/ressource`, nil},
		{`ressource`, []Elem{Branch(``), Branch(`out\test`)}, ``, `out/test/ressource`, nil},
		{`ressource`, []Elem{Branch(``), Branch(``)}, ``, `ressource`, nil},
		{`ressource`, []Elem{}, ``, `ressource`, nil},
		{`C:\ressource`, []Elem{Branch(`in`), Branch(`out\test`)}, ``, `C:/ressource`, nil},
		{`C:\ressource`, []Elem{Branch(``), Branch(`out\test`)}, ``, `C:/ressource`, nil},
		{`C:\ressource`, []Elem{Branch(``), Branch(``)}, ``, `C:/ressource`, nil},
		{``, []Elem{Branch(`C:\in`), Branch(`out\test`)}, ``, `C:/in`, nil},
		{`ressource`, []Elem{Leaf(`file:out`)}, `file`, `out/ressource`, nil},
		{`ressource`, []Elem{Leaf(`file:`)}, `file`, `ressource`, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.wantBackend+":"+tc.wantPath, func(t *testing.T) {
			gotBackent, gotPath, gotErr := GetPath(tc.path, tc.roots...)
			if tc.wantErr != nil {
				assert.ErrorContains(t, gotErr, tc.wantErr.Error())
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, tc.wantBackend, gotBackent)
				assert.Equal(t, tc.wantPath, gotPath)
			}
		})
	}
}
