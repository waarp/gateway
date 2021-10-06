//go:build !windows && !nacl && !plan9
// +build !windows,!nacl,!plan9

package log

import (
	"fmt"

	"code.bcarlin.xyz/go/logging"
)

func newSyslogBackend(facility string) (logging.Backend, error) {
	logger, err := logging.NewSyslogBackend(facility, "waarp-manager")
	if err != nil {
		return nil, fmt.Errorf("cannot initialize syslog backend: %w", err)
	}

	return logger, nil
}
