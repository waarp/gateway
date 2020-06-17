//+build !windows,!nacl,!plan9

package log

import "code.bcarlin.xyz/go/logging"

func newSyslogBackend(facility string) (logging.Backend, error) {
	return logging.NewSyslogBackend(facility, "waarp-manager-ng")
}
