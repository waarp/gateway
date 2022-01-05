// Package backup provides two methods to generate export of the database for
// backup or migration purpose, and to import a previous dump in order to
// restore the database.
package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

var errDry = database.NewValidationError("dry run")

const (
	ImportReset int8 = iota + 1
	ImportForceReset
)

// ImportData reads the content of the reader r, parses it as json and imports
// the subsets specified in targets.
// If dry is true, then the data is not really imported, but a simulation of
// the import is performed.
//
// Possible values for targets are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data.
//
// The reset parameter states whether the database should be reset before
// importing. A value of 1 means 'reset', a value of 2 means
// 'reset with no confirmation prompt', and any other value means 'no reset'.
func ImportData(db *database.DB, r io.Reader, targets []string, dry, reset bool) error {
	logger := conf.GetLogger("import")
	data := &file.Data{}

	if err := json.NewDecoder(r).Decode(data); err != nil {
		return fmt.Errorf("cannot read data: %w", err)
	}

	transErr := db.Transaction(func(ses *database.Session) database.Error {
		if utils.ContainsStrings(targets, "partners", "all") {
			if err := importRemoteAgents(logger, ses, data.Remotes, reset); err != nil {
				return err
			}
		}
		if utils.ContainsStrings(targets, "servers", "all") {
			if err := importLocalAgents(logger, ses, data.Locals, reset); err != nil {
				return err
			}
		}
		if utils.ContainsStrings(targets, "rules", "all") {
			if err := importRules(logger, ses, data.Rules, reset); err != nil {
				return err
			}
		}
		if utils.ContainsStrings(targets, "users", "all") {
			if err := importUsers(logger, ses, data.Users); err != nil {
				return err
			}
		}

		if dry {
			return errDry
		}

		return nil
	})

	if transErr != nil {
		if dry && errors.Is(transErr, errDry) {
			return nil
		}

		return fmt.Errorf("cannot import file: %w", transErr)
	}

	return nil
}

func ImportHistory(db *database.DB, r io.Reader, dry bool) error {
	tErr := db.Transaction(func(ses *database.Session) database.Error {
		if err := ses.DeleteAll(&model.HistoryEntry{}).Run(); err != nil {
			return err
		}

		if err := ses.DeleteAll(&model.Transfer{}).Run(); err != nil {
			return err
		}

		maxID, impErr := importHistory(ses, r)
		if impErr != nil {
			var dbErr database.Error
			if errors.As(impErr, &dbErr) {
				return dbErr
			}

			return database.NewInternalError(impErr)
		}

		// We advance the transfer ID auto-increment to the highest value imported
		// to avoid ID conflicts with new future transfers.
		// We don't advance the auto-increment when doing a dry run because it's
		// useless, and it causes problems with MySQL because it auto-commits the
		// transaction.
		if !dry {
			if err := ses.AdvanceIncrement(&model.Transfer{}, maxID); err != nil {
				return err
			}
		}

		if dry {
			return errDry
		}

		return nil
	})

	if tErr != nil {
		if dry && errors.Is(tErr, errDry) {
			return nil
		}

		return fmt.Errorf("cannot import file: %w", tErr)
	}

	return nil
}
