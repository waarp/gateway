package types

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"path"
	"path/filepath"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

type URL url.URL

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

// ParseURL parses the given raw url into a URL structure. If the scheme is
// missing, it will be assumed to be the file scheme. In any case, with or
// without scheme, the path MUST be absolute. If it isn't, it will be converted
// to an absolute path.
func ParseURL(rawURL string) (*URL, error) {
	rawURL = filepath.ToSlash(rawURL)

	if filepath.VolumeName(rawURL) != "" {
		rawURL = "/" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		//nolint:wrapcheck //wrapping the error adds nothing here
		return nil, err
	}

	if !path.IsAbs(u.Path) {
		u.Path = "/" + u.Path
	}

	if u.Scheme == "" {
		u.Scheme = "file"
	}

	if u.Host == "" {
		u.OmitHost = true
	}

	return (*URL)(u), nil
}

// String returns the string representation of the URL.
func (u *URL) String() string {
	return (*url.URL)(u).Redacted()
}

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

// Query parses RawQuery and returns the corresponding values.
func (u *URL) Query() url.Values {
	return (*url.URL)(u).Query()
}

func (u *URL) FromDB(bytes []byte) error {
	return u.Scan(bytes)
}

func (u *URL) ToDB() ([]byte, error) {
	val, err := u.Value()
	if err != nil {
		return nil, err
	}

	//nolint:forcetypeassert //Value() always returns a string, no need to check
	return []byte(val.(string)), nil
}

func (u *URL) Value() (driver.Value, error) {
	if pwd, _ := u.User.Password(); pwd != "" {
		crypt, err := utils.AESCrypt(database.GCM, pwd)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt url password: %w", err)
		}

		u.User = url.UserPassword(u.User.Username(), crypt)
	}

	return u.String(), nil
}

func parseFromDB(rawURL string) (*URL, error) {
	if rawURL == "" {
		return &URL{}, nil
	}

	return ParseURL(rawURL)
}

func (u *URL) Scan(src any) error {
	var (
		raw      *URL
		parseErr error
	)

	switch v := src.(type) {
	case string:
		raw, parseErr = parseFromDB(v)
	case []byte:
		raw, parseErr = parseFromDB(string(v))
	default:
		//nolint:goerr113 // too specific to have a base error
		return fmt.Errorf("cannot scan %+v of type %T into a URL", v, v)
	}

	if parseErr != nil {
		return fmt.Errorf(`failed to parse url "%v": %w`, src, parseErr)
	}

	if crypt, _ := raw.User.Password(); crypt != "" {
		pwd, err := utils.AESDecrypt(database.GCM, crypt)
		if err != nil {
			return fmt.Errorf("failed to decrypt url password: %w", err)
		}

		raw.User = url.UserPassword(raw.User.Username(), pwd)
	}

	*u = *raw

	return nil
}
