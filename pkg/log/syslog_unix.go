//+build !windows,!nacl,!plan9

package log

import "code.bcarlin.xyz/go/logging"

func NewSyslogBackend() (logging.Backend, error) {
	return logging.NewSyslogBackend(conf.SyslogFacility, "waarp-manager-ng")
}
