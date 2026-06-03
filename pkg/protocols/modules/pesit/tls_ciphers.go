package pesit

import (
	"crypto/tls"
	"fmt"
	"strings"
)

// tlsCipherSuiteByName builds a map of cipher suite name → ID from the
// Go standard library. Both secure and insecure suites are included so
// that admins can configure legacy suites for mainframe interop.
func tlsCipherSuiteByName() map[string]uint16 {
	m := make(map[string]uint16)

	for _, cs := range tls.CipherSuites() {
		m[cs.Name] = cs.ID
	}

	for _, cs := range tls.InsecureCipherSuites() {
		m[cs.Name] = cs.ID
	}

	return m
}

// resolveCipherSuites converts a list of cipher suite names (from config)
// to a list of uint16 IDs suitable for tls.Config.CipherSuites.
// Unknown names are logged and skipped.
func resolveCipherSuites(names []string) ([]uint16, error) {
	if len(names) == 0 {
		return nil, nil // use Go defaults
	}

	lookup := tlsCipherSuiteByName()
	ids := make([]uint16, 0, len(names))

	var unknown []string

	for _, name := range names {
		id, ok := lookup[name]
		if !ok {
			unknown = append(unknown, name)

			continue
		}

		ids = append(ids, id)
	}

	if len(unknown) > 0 {
		return ids, fmt.Errorf("unknown TLS cipher suites: %s", strings.Join(unknown, ", "))
	}

	return ids, nil
}

// validTLSCipherSuiteNames returns all known cipher suite names for
// validation and documentation purposes.
func validTLSCipherSuiteNames() []string {
	var names []string

	for _, cs := range tls.CipherSuites() {
		names = append(names, cs.Name)
	}

	for _, cs := range tls.InsecureCipherSuites() {
		names = append(names, cs.Name)
	}

	return names
}
