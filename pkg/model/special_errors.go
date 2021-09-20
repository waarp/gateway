package model

// PauseError is the error returned when a transfer is paused by the user.
type PauseError struct{}

func (p *PauseError) Error() string {
	return "transfer paused"
}

// CancelError is the error returned when a transfer is canceled by the user.
type CancelError struct{}

func (c *CancelError) Error() string {
	return "transfer canceled"
}

// ShutdownError is the error returned when a transfer is interrupted by a
// service shutdown.
type ShutdownError struct{}

func (s *ShutdownError) Error() string {
	return "service is shutting down"
}
