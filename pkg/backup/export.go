package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

//nolint:gochecknoglobals //global variable needed here for Waarp Transfer
var ExportFileHeaderAppName = "waarp-gatewayd"

func printExportFileHeader(f io.Writer, prefix string) {
	fmt.Fprintf(f,
		"%sCreated on %s by %s version %s\n",
		prefix,
		time.Now().Format(time.UnixDate),
		ExportFileHeaderAppName,
		version.Num,
	)
}

// ExportData extracts from the database the subsets specified in targets,
// serializes it in JSON, and writes the result in the writer w.
//
// Possible values for targets are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data.
//
//nolint:funlen,gocognit,cyclop //function cannot be easily split
func ExportData(db database.ReadAccess, w *os.File, targets []string) error {
	logger := logging.NewLogger("export")

	var expErr error

	data := &file.Data{}

	if utils.ContainsOneOf(targets, "servers", "all") {
		data.Locals, expErr = exportLocals(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "clients", "all") {
		data.Clients, expErr = exportClients(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "partners", "all") {
		data.Remotes, expErr = exportRemotes(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "rules", "all") {
		data.Rules, expErr = exportRules(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "users", "all") {
		data.Users, expErr = exportUsers(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "clouds", "all") {
		data.Clouds, expErr = exportClouds(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "snmp", "all") {
		data.SNMPConfig, expErr = exportSNMPConfig(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "authorities", "all") {
		data.Authorities, expErr = exportAuthorities(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "keys", "all") {
		data.CryptoKeys, expErr = exportCryptoKeys(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if utils.ContainsOneOf(targets, "email", "all") {
		data.EmailConfig, expErr = exportEmailConf(logger, db)
		if expErr != nil {
			return expErr
		}
	}

	if err := serializeFile(data, w); err != nil {
		return fmt.Errorf("cannot encode data: %w", err)
	}

	return nil
}

func serializeFile(data *file.Data, f *os.File) error {
	var serErr error

	ext := filepath.Ext(f.Name())
	switch ext {
	case ".yaml", ".yml":
		const yamlIndent = 2

		printExportFileHeader(f, "# ")
		encoder := yaml.NewEncoder(f)
		encoder.SetIndent(yamlIndent)
		serErr = encoder.Encode(data)
	default:
		encoder := json.NewEncoder(f)
		encoder.SetIndent("", "  ")
		serErr = encoder.Encode(&data)
	}

	if serErr != nil {
		return fmt.Errorf("failed to serialize file %q: %w", f.Name(), serErr)
	}

	return nil
}
