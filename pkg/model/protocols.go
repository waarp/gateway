// Package model contains all the definitions of the database models. Each
// model instance represents an entry in one of the database's tables.
package model

var validProtocols = []string{"sftp", "r66"}

// IsValidProtocol returns whether the given protocol is a valid protocol for
// the gateway.
func IsValidProtocol(proto string) bool {
	for _, protocol := range validProtocols {
		if proto == protocol {
			return true
		}
	}
	return false
}
