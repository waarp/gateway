package filewatcher

import (
	"fmt"
	"io/fs"
	"path"
)

const TestProtocol = "test"

//nolint:gochecknoglobals //global var needed here for tests
var getTestLister func() Lister

func EnableTestProtocol(files ...fs.FileInfo) {
	getTestLister = func() Lister {
		return &testLister{files}
	}
}

type testLister struct {
	files []fs.FileInfo
}

func (t *testLister) List(pattern string) ([]fs.FileInfo, error) {
	res := make([]fs.FileInfo, 0, len(t.files))
	for _, f := range t.files {
		ok, err := path.Match(pattern, f.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to match pattern: %w", err)
		} else if ok {
			res = append(res, f)
		}
	}

	return res, nil
}
