package main

import (
	"fmt"
	"os"

	"code.waarp.fr/lib/log"
)

func getLogger() *log.Logger {
	//nolint:errcheck //never returns an error
	level := log.LevelInfo
	if os.Getenv("WAARP_GATEWAY_DEBUG") != "" {
		level = log.LevelDebug
	}

	back, err := log.NewBackend(level, log.Stdout, "", "")
	if err != nil {
		fmt.Printf("ERROR: cannot get log backend: %v\n", err)
		os.Exit(exitLogBackendError)
	}

	return back.NewLogger(loggerName)
}
