//go:build unix

package types

func (u *URL) toOSPath() string {
	return u.Path
}
