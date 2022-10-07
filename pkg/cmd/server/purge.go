package wgd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var ErrTransfersNotEmpty = errors.New("the transfer table must be empty to reset the transfer ID increment")

//nolint:lll // tags can be long for flags
type PurgeCommand struct {
	ConfigFile string `short:"c" long:"config" required:"yes" description:"The configuration file to use"`
	Reset      bool   `short:"r" long:"reset" description:"Reset the transfer ID auto-increment after the purge."`
	Verbose    []bool `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity"`
}

func (p *PurgeCommand) Execute([]string) error {
	db, _, initErr := initImportExport(p.ConfigFile, p.Verbose)
	if initErr != nil {
		return initErr
	}

	return p.run(db, os.Stdin, os.Stdout)
}

func (p *PurgeCommand) run(db *database.DB, in io.Reader, out io.Writer) error {
	var proceed string

	if p.Reset {
		nTrans, err := db.Count(&model.Transfer{}).Run()
		if err != nil {
			return fmt.Errorf("failed to count the transfer entries: %w", err)
		}

		if nTrans != 0 {
			return ErrTransfersNotEmpty
		}
	}

	nHist, err := db.Count(&model.HistoryEntry{}).Run()
	if err != nil {
		return fmt.Errorf("failed to count the history entries to purge: %w", err)
	}

	fmt.Fprintf(out, "You are about to purge %d history entries.\n", nHist)
	fmt.Fprintln(out, "This operation cannot be undone. Do you wish to proceed anyway ?")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "(Type 'YES' in all caps to proceed): ")
	_, scanErr := fmt.Fscanf(in, "%s", &proceed)
	fmt.Fprintln(out)

	if scanErr != nil || proceed != "YES" {
		fmt.Fprintln(out, "Purge aborted.")

		return nil
	}

	if err := db.Transaction(func(ses *database.Session) database.Error {
		if err := ses.DeleteAll(&model.HistoryEntry{}).Run(); err != nil {
			return err
		}

		if p.Reset {
			if err := ses.ResetIncrement(&model.Transfer{}); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to purge the transfer history: %w", err)
	}

	if p.Reset {
		fmt.Fprintln(out, "The transfer history has been purged successfully,")
		fmt.Fprintln(out, "and the transfer ID increment has been reset.")
	} else {
		fmt.Fprintln(out, "The transfer history has been purged successfully.")
	}

	return nil
}
