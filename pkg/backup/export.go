package backup

import (
	"encoding/json"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// ExportData extracts from the database the subsets specified in targets,
// serializes it in JSON, and writes the result in the writer w.
//
// Possible values for targets are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data.
func ExportData(db database.ReadAccess, w io.Writer, targets []string) error {
	logger := log.NewLogger("export")

	var err error
	data := &file.Data{}

	if utils.ContainsStrings(targets, "servers", "all") {
		data.Locals, err = exportLocals(logger, db)
		if err != nil {
			return err
		}
	}
	if utils.ContainsStrings(targets, "partners", "all") {
		data.Remotes, err = exportRemotes(logger, db)
		if err != nil {
			return err
		}
	}
	if utils.ContainsStrings(targets, "rules", "all") {
		data.Rules, err = exportRules(logger, db)
		if err != nil {
			return err
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	return encoder.Encode(data)
}
