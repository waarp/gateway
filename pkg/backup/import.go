// Package backup provides two methods too generate export of the database for
// backup or migration purpose, and to import a previous dump in order to
// restore the database.
package backup

import (
	"encoding/json"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

var errDry = database.NewValidationError("dry run")

// ImportData reads the content of the reader r, parses it as json and imports
// the subsets specified in targets.
// If dry is true, then the data is not really imported, but a simulation of
// the import is performed.
//
// Possible values for targets are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data.
func ImportData(db *database.DB, r io.Reader, targets []string, dry bool) error {
	logger := log.NewLogger("import")

	data := &file.Data{}
	err := json.NewDecoder(r).Decode(data)
	if err != nil {
		return err
	}

	err = db.Transaction(func(ses *database.Session) database.Error {
		if utils.ContainsStrings(targets, "partners", "all") {
			if err := importRemoteAgents(logger, ses, data.Remotes); err != nil {
				return err
			}
		}
		if utils.ContainsStrings(targets, "servers", "all") {
			if err := importLocalAgents(logger, ses, data.Locals); err != nil {
				return err
			}
		}
		if utils.ContainsStrings(targets, "rules", "all") {
			if err := importRules(logger, ses, data.Rules); err != nil {
				return err
			}
		}

		if dry {
			return errDry
		}
		return nil
	})

	if err != nil {
		if dry && err == errDry {
			return nil
		}
		return err
	}

	return nil
}
