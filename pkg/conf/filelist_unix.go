//go:build !windows

package conf

const (
	DefaultFilePermissions = 0o640
	DefaultDirPermissions  = 0o750
)

func getDefaultConfFiles() []string {
	return []string{
		"gatewayd.ini",
		"etc/gatewayd.ini",
		"/etc/waarp-gateway/gatewayd.ini",
	}
}
