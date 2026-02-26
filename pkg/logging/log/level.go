package log

import "code.waarp.fr/lib/log/v2"

type Level = log.Level

const (
	LevelTrace    = log.LevelTrace
	LevelDebug    = log.LevelDebug
	LevelInfo     = log.LevelInfo
	LevelNotice   = log.LevelNotice
	LevelWarning  = log.LevelWarning
	LevelError    = log.LevelError
	LevelCritical = log.LevelCritical
	LevelAlert    = log.LevelAlert
	LevelFatal    = log.LevelFatal
)

func LevelByName(name string) (Level, error) {
	//nolint:wrapcheck //no need to wrap here
	return log.LevelByName(name)
}
