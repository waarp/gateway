package backup

import (
	"encoding/json"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// ExportData extracts from the database the subsets specified in targets,
// serializes it in JSON, and writes the result in the writer w.
//
// Possible values for targets are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data.
func ExportData(db *database.DB, w io.Writer, targets []string) error {
	logger := log.NewLogger("export")

	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}
	defer ses.Rollback()

	data := &data{}

	if utils.ContainsStrings(targets, "servers", "all") {
		data.Locals, err = exportLocals(logger, ses)
		if err != nil {
			return err
		}
	}
	if utils.ContainsStrings(targets, "partners", "all") {
		data.Remotes, err = exportRemotes(logger, ses)
		if err != nil {
			return err
		}
	}
	if utils.ContainsStrings(targets, "rules", "all") {
		data.Rules, err = exportRules(logger, ses)
		if err != nil {
			return err
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	return encoder.Encode(data)
}
