//go:build !windows

package conf

func getDefaultConfFiles() []string {
	return []string{
		"gatewayd.ini",
		"etc/gatewayd.ini",
		"/etc/waarp-gateway/gatewayd.ini",
	}
}
