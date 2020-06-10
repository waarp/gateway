package backup

import (
	"encoding/json"
	"io"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

// ExportData extracts from the database the subsets specified in targets,
// serializes it in JSON, and writes the result in the writer w.
//
// Possible values for target are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data. Several groups can be given separated by ','.
func ExportData(db *database.DB, w io.Writer, target string) error {
	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}
	defer ses.Rollback()

	data := &data{}

	if strings.Contains(target, "servers") || strings.Contains(target, "all") {
		data.Locals, err = exportLocals(ses)
		if err != nil {
			return err
		}
	}
	if strings.Contains(target, "partners") || strings.Contains(target, "all") {
		data.Remotes, err = exportRemotes(ses)
		if err != nil {
			return err
		}
	}
	if strings.Contains(target, "rules") || strings.Contains(target, "all") {
		data.Rules, err = exportRules(ses)
		if err != nil {
			return err
		}
	}

	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		return err
	}

	return nil
}
