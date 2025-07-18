package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"

	wgd "code.waarp.fr/apps/gateway/gateway/pkg/cmd/server"
)

const (
	memLimitVar = "WAARP_GATEWAYD_MEMORY_LIMIT"
	cpuLimitVar = "WAARP_GATEWAYD_CPU_LIMIT"
)

//nolint:lll // tags can be long for flags
type commands struct {
	Server      wgd.ServerCommand       `command:"server" description:"Start/Create the gateway"`
	Import      wgd.ImportCommand       `command:"import" description:"Imports the data of source file into the gateway database"`
	Export      wgd.ExportCommand       `command:"export" description:"Exports the data of the gateway database to the destination file"`
	Purge       wgd.PurgeCommand        `command:"purge" description:"Purge the transfer history"`
	RestoreHist wgd.RestoreHistCommand  `command:"restore-history" description:"Restore the given transfer history dump"`
	Version     wgd.VersionCommand      `command:"version" description:"Print version and exit"`
	Migrate     wgd.MigrateCommand      `command:"migrate" description:"Migrate the gateway database to a different version"`
	ChangeAES   wgd.ChangeAESPassphrase `command:"change-aes-passphrase" description:"Change the AES passphrase of the gateway database"`
	SQL         wgd.SQLCommand          `command:"sql" description:"Execute a custom SQL query on the gateway database"`
}

func main() {
	setResourcesLimits()
	parser := flags.NewNamedParser("waarp-gatewayd", flags.Default)

	if err := wgd.InitParser(parser, &commands{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := wgd.Main(parser, os.Args[1:]); err != nil {
		// fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setResourcesLimits() {
	memLimit := os.Getenv(memLimitVar)
	cpuLimit := os.Getenv(cpuLimitVar)

	if memLimit != "" {
		if bytes, err := humanize.ParseBytes(memLimit); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse memory limit: %v\n", err)
		} else {
			//nolint:gosec //conversion from uint64 to int64 is fine in most cases
			debug.SetMemoryLimit(int64(bytes))
		}
	}

	if cpuLimit != "" {
		if cpus, err := strconv.Atoi(cpuLimit); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse CPU limit: %v\n", err)
		} else {
			runtime.GOMAXPROCS(cpus)
		}
	}
}
