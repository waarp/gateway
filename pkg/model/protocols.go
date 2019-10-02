package model

var validProtocols = []string{"sftp"}

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
