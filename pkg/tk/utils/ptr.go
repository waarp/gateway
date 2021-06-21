package utils

// StringPtr takes a string and returns a pointer to that string. Useful for
// putting a string literal where a string pointer is required.
func StringPtr(s string) *string {
	return &s
}

// String takes a string pointer and returns the string it is pointing to. If
// the pointer is nil, returns an empty string. Useful to avoid panics because
// of nil pointer dereference.
func String(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
