package main

import "code.waarp.fr/lib/log"

func getLogger() *log.Logger {
	//nolint:errcheck //never returns an error
	back, _ := log.NewBackend(log.LevelInfo, log.Stdout, "", "")

	return back.NewLogger(loggerName)
}
