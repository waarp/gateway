package backup

import (
	"encoding/json"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

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

	data := &data{}
	err := json.NewDecoder(r).Decode(data)
	if err != nil {
		return err
	}

	trans, err := db.BeginTransaction()
	if err != nil {
		return err
	}
	defer trans.Rollback()

	if utils.ContainsStrings(targets, "partners", "all") {
		err = importRemoteAgents(logger, trans, data.Remotes)
		if err != nil {
			return err
		}
	}
	if utils.ContainsStrings(targets, "servers", "all") {
		err = importLocalAgents(logger, trans, data.Locals)
		if err != nil {
			return err
		}
	}
	if utils.ContainsStrings(targets, "rules", "all") {
		err = importRules(logger, trans, data.Rules)
		if err != nil {
			return err
		}
	}

	if !dry {
		return trans.Commit()
	}

	return nil
}
