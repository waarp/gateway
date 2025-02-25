package fs

import (
	"io/fs"
	"path"
)

var ErrBadPattern = path.ErrBadPattern

func Glob(pattern string) ([]string, error) {
	parsed, srcFs, fsErr := parseFs(pattern)
	if fsErr != nil {
		return nil, fsErr
	}

	matches, err := fs.Glob(srcFs, parsed.Path)
	if err != nil {
		return nil, err //nolint:wrapcheck //wrapping adds nothing here
	}

	for i, match := range matches {
		if parsed.Name != "" {
			matches[i] = parsed.Name + ":" + match
		}
	}

	return matches, nil
}
