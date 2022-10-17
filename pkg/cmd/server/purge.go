package wgd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/karrick/tparse/v2"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:lll // error messages are long
var (
	ErrResetTransfersNotEmpty = errors.New(`the transfer table must be empty to reset the transfer ID increment`)
	ErrResetOlderThan         = errors.New(`the "--reset" option cannot be usedin combination with the "--older-than" option`)
	ErrInvalidOlderThan       = errors.New(`failed to parse the "--older-than" options, must be either a date or a duration`)
)

const untilFormat = "2006/01/02 15:04:05"

//nolint:lll // tags can be long for flags
type PurgeCommand struct {
	ConfigFile string `short:"c" long:"config" required:"yes" description:"The configuration file to use"`
	Reset      bool   `short:"r" long:"reset" description:"Reset the transfer ID auto-increment after the purge (NOTE: cannot be used in combination with -o)."`
	OlderThan  string `short:"o" long:"older-than" description:"Limit the purge to transfers older than the given time (can be either a date or a duration)."`
	Verbose    []bool `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity"`
}

func (p *PurgeCommand) Execute([]string) error {
	db, _, initErr := initImportExport(p.ConfigFile, p.Verbose)
	if initErr != nil {
		return initErr
	}

	return p.run(db, time.Now(), os.Stdin, os.Stdout)
}

func (p *PurgeCommand) run(db *database.DB, now time.Time, in io.Reader,
	out io.Writer,
) error {
	if err := p.checkArguments(db, now); err != nil {
		return err
	}

	proceed, err := p.userConfirm(db, in, out)
	if err != nil {
		return err
	} else if !proceed {
		return nil
	}

	if err := db.Transaction(func(ses *database.Session) database.Error {
		delQuery := ses.DeleteAll(&model.HistoryEntry{})

		if p.OlderThan != "" {
			delQuery.Where("stop <= ? OR (stop IS NULL AND start <= ?)", p.OlderThan,
				p.OlderThan)
		}

		if err := delQuery.Run(); err != nil {
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

func (p *PurgeCommand) checkArguments(db database.ReadAccess, now time.Time) error {
	if p.Reset {
		if p.OlderThan != "" {
			return ErrResetOlderThan
		}

		nTrans, err := db.Count(&model.Transfer{}).Run()
		if err != nil {
			return fmt.Errorf("failed to count the transfer entries: %w", err)
		}

		if nTrans != 0 {
			return ErrResetTransfersNotEmpty
		}
	}

	if p.OlderThan != "" {
		date, timeErr := time.ParseInLocation(untilFormat, p.OlderThan, time.Local)
		if timeErr != nil {
			date, timeErr = tparse.AddDuration(now, "-"+p.OlderThan)
			if timeErr != nil {
				return ErrInvalidOlderThan
			}
		}

		p.OlderThan = date.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
	}

	return nil
}

func (p *PurgeCommand) userConfirm(db database.ReadAccess, in io.Reader,
	out io.Writer,
) (bool, error) {
	countQuery := db.Count(&model.HistoryEntry{})

	if p.OlderThan != "" {
		countQuery.Where("stop <= ? OR (stop IS NULL AND start <= ?)", p.OlderThan,
			p.OlderThan)
	}

	nHist, err := countQuery.Run()
	if err != nil {
		return false, fmt.Errorf("failed to count the history entries to purge: %w", err)
	}

	fmt.Fprintf(out, "You are about to purge %d history entries.\n", nHist)
	fmt.Fprintln(out, "This operation cannot be undone. Do you wish to proceed anyway ?")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "(Type 'YES' in all caps to proceed): ")

	var proceed string
	_, scanErr := fmt.Fscanf(in, "%s", &proceed)

	fmt.Fprintln(out)

	if scanErr != nil || proceed != "YES" {
		fmt.Fprintln(out, "Purge aborted.")

		return false, nil
	}

	return true, nil
}
