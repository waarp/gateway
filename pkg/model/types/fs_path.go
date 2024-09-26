package types

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rclone/rclone/fs/fspath"
)

type FSPath struct {
	Backend string
	Path    string
}

var ErrNoVolumeName = errors.New("missing windows volume name in path")

// ParsePath parses the given file path into an FSPath structure. The path must
// have the form `backend:path`. If no backend is given, the local file system
// will be assumed.
// For windows paths, the path MUST be absolute, meaning it must start with a
// drive letter.
// For other file systems, the path must be either absolute, or relative to the
// file system's root.
func ParsePath(fullPath string) (*FSPath, error) {
	parsed, pathErr := fspath.Parse(fullPath)
	if pathErr != nil {
		return nil, fmt.Errorf("failed to parse file path: %w", pathErr)
	}

	if parsed.Name == "file" {
		parsed.Name = ""
	}

	if runtime.GOOS == "windows" && parsed.Name == "" {
		parsed.Path = strings.TrimLeft(parsed.Path, "/")

		if filepath.VolumeName(parsed.Path) == "" {
			return nil, ErrNoVolumeName
		}
	}

	return &FSPath{
		Backend: parsed.Name,
		Path:    parsed.Path,
	}, nil
}

func (p *FSPath) IsBlank() bool { return p.Path == "" && p.Backend == "" }

// String returns the string representation of the URL.
func (p *FSPath) String() string {
	if p.Backend == "" {
		return p.Path
	}

	return p.Backend + ":" + p.Path
}

// JoinPath returns a new URL with the given elements joined to the already
// existing path of the URL.
func (p *FSPath) JoinPath(elem ...string) *FSPath {
	return &FSPath{
		Backend: p.Backend,
		Path:    path.Join(append([]string{p.Path}, elem...)...),
	}
}

// Dir returns a new URL similar to the existing one, but without the trailing
// file name. The new URL will thus point to the parent directory of the old URL.
func (p *FSPath) Dir() *FSPath {
	return &FSPath{
		Backend: p.Backend,
		Path:    path.Dir(p.Path),
	}
}

func (p *FSPath) FromDB(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	parsed, err := ParsePath(string(bytes))
	if err != nil {
		return err
	}

	*p = *parsed

	return nil
}

func (p *FSPath) ToDB() ([]byte, error)        { return []byte(p.String()), nil }
func (p *FSPath) Value() (driver.Value, error) { return p.String(), nil }

func (p *FSPath) Scan(src any) error {
	switch v := src.(type) {
	case string:
		return p.FromDB([]byte(v))
	case []byte:
		return p.FromDB(v)
	default:
		//nolint:goerr113 // too specific to have a base error
		return fmt.Errorf("cannot scan %+v of type %T into a URL", v, v)
	}
}

func (p *FSPath) FSPath() string {
	fsPath := strings.TrimLeft(p.Path, "/")
	if vol := filepath.VolumeName(fsPath); vol != "" {
		fsPath = strings.TrimPrefix(fsPath, vol+"/")
	}

	return path.Clean(fsPath)
}
