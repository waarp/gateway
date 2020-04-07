package backup

import (
	"encoding/json"
	"io"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

// ImportData reads the content of the reader r, parses it as json and imports
// the subsets specified in targets.
// If dry is true, then the data is not really imported, but a simulation of
// the import is performed.
//
// Possible values for target are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data. Several groups can be given separated by ','.
func ImportData(db *database.DB, r io.Reader, target string, dry bool) error {
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

	if strings.Contains(target, "partners") || strings.Contains(target, "all") {
		err = importRemoteAgents(trans, data.Remotes)
		if err != nil {
			return err
		}
	}
	if strings.Contains(target, "servers") || strings.Contains(target, "all") {
		err = importLocalAgents(trans, data.Locals)
		if err != nil {
			return err
		}
	}
	if strings.Contains(target, "rules") || strings.Contains(target, "all") {
		err = importRules(trans, data.Rules)
		if err != nil {
			return err
		}
	}

	if !dry {
		return trans.Commit()
	}

	return nil
}
