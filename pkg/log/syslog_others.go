// +build windows nacl plan9

package log

import (
	"fmt"

	"code.bcarlin.xyz/go/logging"
)

func newSyslogBackend(string) (logging.Backend, error) {
	return nil, fmt.Errorf("syslog logging is not available on this system")
}
