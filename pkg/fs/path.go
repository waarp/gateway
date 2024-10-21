package fs

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rclone/rclone/fs/fspath"
)

type parsedPath fspath.Parsed

func (p *parsedPath) String() string {
	if p.Name == "" {
		return p.Path
	}

	return p.Name + ":" + p.Path
}

func (p *parsedPath) dir() *parsedPath {
	return &parsedPath{Name: p.Name, Path: filepath.Dir(p.Path)}
}

func (p *parsedPath) unrooted() string {
	volume := filepath.VolumeName(p.Path)
	if volume == "" {
		return p.Path
	}

	return strings.TrimPrefix(p.Path, volume+"/")
}

func parsePath(path string) (*parsedPath, error) {
	if path == "" {
		return &parsedPath{}, nil
	}

	parsed, err := fspath.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	// For retro-compatibility, "file" => local file system.
	if parsed.Name == "file" {
		parsed.Name = ""
	}

	// For retro-compatibility, remove leading slashes on Windows paths.
	if parsed.Name == "" && runtime.GOOS == "windows" {
		parsed.Path = strings.TrimLeft(parsed.Path, "/")
	}

	p := parsedPath(parsed)

	return &p, nil
}

func ValidPath(path string) error {
	parsed, err := parsePath(path)
	if err != nil {
		return err
	}

	if parsed.Name != "" {
		if _, ok := FileSystems.Load(parsed.Name); !ok {
			return fmt.Errorf("%w %q", ErrUnknownFS, parsed.Name)
		}
	}

	return err
}

func IsAbsPath(path string) bool {
	parsed, err := fspath.Parse(path)
	if err != nil {
		return false
	}

	if parsed.Name != "" {
		return true
	}

	if runtime.GOOS == "windows" {
		parsed.Path = strings.TrimLeft(parsed.Path, "/")
	}

	return filepath.IsAbs(parsed.Path)
}
