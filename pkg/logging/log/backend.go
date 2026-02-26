package log

import "code.waarp.fr/lib/log/v2"

const (
	Stdout  = log.Stdout
	Stderr  = log.Stderr
	Discard = log.Discard
	Syslog  = log.Syslog
)

type (
	Backend     = log.Backend
	BackendPool = log.BackendPool
)

func NewBackend(level Level, logTo, facility, tag string) (*Backend, error) {
	//nolint:wrapcheck //no need to wrap here
	return log.NewBackend(level, logTo, facility, tag)
}
