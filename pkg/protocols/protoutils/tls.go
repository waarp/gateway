// Package protoutils provides utility functions for protocol implementations.
package protoutils

import "crypto/tls"

const (
	TLSv10 = "v1.0"
	TLSv11 = "v1.1"
	TLSv12 = "v1.2"
	TLSv13 = "v1.3"
)

func ParseTLSVersion(version string) uint16 {
	switch version {
	case TLSv10:
		return tls.VersionTLS10
	case TLSv11:
		return tls.VersionTLS11
	case TLSv12:
		return tls.VersionTLS12
	case TLSv13:
		return tls.VersionTLS13
	default:
		return 0
	}
}
