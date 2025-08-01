package backup

import (
	"encoding/json"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// ExportData extracts from the database the subsets specified in targets,
// serializes it in JSON, and writes the result in the writer w.
//
// Possible values for targets are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data.
//
//nolint:funlen,gocognit,cyclop //function cannot be easily split
func ExportData(db database.ReadAccess, w io.Writer, targets []string) error {
	logger := logging.NewLogger("export")

	var err error

	data := &file.Data{}

	if utils.ContainsOneOf(targets, "servers", "all") {
		data.Locals, err = exportLocals(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "clients", "all") {
		data.Clients, err = exportClients(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "partners", "all") {
		data.Remotes, err = exportRemotes(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "rules", "all") {
		data.Rules, err = exportRules(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "users", "all") {
		data.Users, err = exportUsers(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "clouds", "all") {
		data.Clouds, err = exportClouds(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "snmp", "all") {
		data.SNMPConfig, err = exportSNMPConfig(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "authorities", "all") {
		data.Authorities, err = exportAuthorities(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "keys", "all") {
		data.CryptoKeys, err = exportCryptoKeys(logger, db)
		if err != nil {
			return err
		}
	}

	if utils.ContainsOneOf(targets, "email", "all") {
		data.EmailConfig, err = exportEmailConf(logger, db)
		if err != nil {
			return err
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if encErr := encoder.Encode(data); encErr != nil {
		return fmt.Errorf("cannot encode data: %w", encErr)
	}

	return nil
}
