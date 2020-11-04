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

// boolPtr takes a boolean and returns a pointer to that bool. Useful for
// putting a boolean literal where a boolean pointer is required.
func boolPtr(b bool) *bool {
	return &b
}

var (
	// TruePtr is a boolean pointer to a true constant.
	TruePtr *bool = boolPtr(true)
	// FalsePtr is a boolean pointer to a false constant.
	FalsePtr *bool = boolPtr(false)
)
