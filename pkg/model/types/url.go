package types

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

type URL url.URL

// ParseURL parses the given raw url into a URL structure. If the scheme is
// missing, it will be assumed to be the file scheme. In any case, with or
// without scheme, the path MUST be absolute.
func ParseURL(rawURL string) (*URL, error) {
	if filepath.VolumeName(rawURL) != "" {
		rawURL = "/" + rawURL
	}

	rawURL = filepath.ToSlash(rawURL)

	u, err := url.Parse(rawURL)
	if err != nil {
		//nolint:wrapcheck //wrapping the error adds nothing here
		return nil, err
	}

	if u.Scheme == "" || u.Scheme == "file" {
		return &URL{
			Scheme:   "file",
			OmitHost: true,
			Path:     path.Join("/", u.Path),
		}, nil
	}

	return &URL{
		Scheme:   u.Scheme,
		Host:     u.Host,
		Path:     u.Path,
		OmitHost: u.Host == "",
	}, nil
}

// GetScheme parses the given raw url, and returns its scheme. If the given URL
// does not have a scheme, or if the parsing fails, the function returns an
// empty string.
func GetScheme(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	return u.Scheme
}

// String returns the string representation of the URL.
func (u *URL) String() string { return (*url.URL)(u).String() }

// JoinPath returns a new URL with the given elements joined to the already
// existing path of the URL.
func (u *URL) JoinPath(elem ...string) *URL {
	return (*URL)((*url.URL)(u).JoinPath(elem...))
}

// Dir returns a new URL similar to the existing one, but without the trailing
// file name. The new URL will thus point to the parent directory of the old URL.
func (u *URL) Dir() *URL {
	newURL := *u
	newURL.Path = path.Dir(newURL.Path)

	return &newURL
}

func (u *URL) FromDB(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	raw, err := ParseURL(string(bytes))
	if err != nil {
		return err
	}

	*u = *raw

	return nil
}

func (u *URL) ToDB() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *URL) Value() (driver.Value, error) {
	return u.String(), nil
}

func (u *URL) Scan(src any) error {
	switch v := src.(type) {
	case string:
		return u.FromDB([]byte(v))
	case []byte:
		return u.FromDB(v)
	default:
		//nolint:goerr113 // too specific to have a base error
		return fmt.Errorf("cannot scan %+v of type %T into a URL", v, v)
	}
}

func (u *URL) OSPath() string {
	if u.Scheme != "file" {
		return u.String()
	}

	return u.toOSPath()
}

func (u *URL) FSPath() string {
	fsPath := strings.TrimLeft(u.Path, "/")
	if vol := filepath.VolumeName(fsPath); vol != "" {
		fsPath = strings.TrimPrefix(fsPath, vol+"/")
	}

	return path.Clean(fsPath)
}
