package api

// DateHeader is the name of the header used by the REST API to provide the
// server's local date, since the standard "Date" header does not allow any
// time zone other than GMT. The date is still in standard RFC1123 format, but
// other time zones allowed.
const DateHeader = "Waarp-Gateway-Date"

// Service represents a gateway service (core, server or client) along with its
// status.
type Service struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}
