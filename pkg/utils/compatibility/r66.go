package compatibility

// IsTLS checks whether the given R66 proto config map contains an "isTLS"
// property, and whether that property is true. If the property does not exist,
// this returns false.
//
// The "isTLS" property has been replaced by the "r66-tls" protocol, but to
// maintain backwards compatibility, the property has been kept. This function
// can be used to check its presence.
func IsTLS(mapConf map[string]any) bool {
	if isTLSany, hasTLS := mapConf["isTLS"]; hasTLS {
		if isTLS, isBool := isTLSany.(bool); isBool && isTLS {
			return true
		}
	}

	return false
}
