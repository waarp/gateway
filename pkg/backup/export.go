package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
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

// ExportFile is where an export is written: its name gives the format (YAML
// for .yaml and .yml, JSON otherwise).
type ExportFile interface {
	io.Writer
	Name() string
}

// ExportData extracts from the database the subsets specified in targets,
// serializes it in JSON, and writes the result in the writer w.
//
// Possible values for targets are 'rules' for the transfer rules, 'servers' for
// local servers and accounts, 'partners' for remote partners and accounts, or
// 'all' for all data.
func ExportData(db database.ReadAccess, w ExportFile, targets []string) error {
	logger := logging.NewLogger("export")

	return Export(db, logger, w, targets)
}

// Export collects the subsets specified in targets and writes them to w:
// Collect, then Serialize.
func Export(db database.ReadAccess, logger *log.Logger, w ExportFile, targets []string) error {
	data, err := Collect(db, logger, targets)
	if err != nil {
		return err
	}

	return Serialize(data, w)
}

// Collect extracts from the database the subsets specified in targets, as
// Export writes them, without serializing them: what a caller may read or
// amend before Serialize.
//
//nolint:funlen,gocognit,cyclop //function cannot be easily split
func Collect(db database.ReadAccess, logger *log.Logger, targets []string) (*file.Data, error) {
	var expErr error

	data := &file.Data{}

	if utils.ContainsOneOf(targets, "servers", "all") {
		data.Locals, expErr = exportLocals(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "clients", "all") {
		data.Clients, expErr = exportClients(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "partners", "all") {
		data.Remotes, expErr = exportRemotes(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "rules", "all") {
		data.Rules, expErr = exportRules(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "users", "all") {
		data.Users, expErr = exportUsers(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "clouds", "all") {
		data.Clouds, expErr = exportClouds(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "snmp", "all") {
		data.SNMPConfig, expErr = exportSNMPConfig(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "authorities", "all") {
		data.Authorities, expErr = exportAuthorities(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "keys", "all") {
		data.CryptoKeys, expErr = exportCryptoKeys(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "email", "all") {
		data.EmailConfig, expErr = exportEmailConf(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	if utils.ContainsOneOf(targets, "filewatchers", "all") {
		data.Filewatchers, expErr = exportFilewatchers(logger, db)
		if expErr != nil {
			return nil, expErr
		}
	}

	return data, nil
}

// Serialize writes the data to the file, in the format its name gives.
func Serialize(data *file.Data, w ExportFile) error {
	if err := serializeFile(data, w); err != nil {
		return fmt.Errorf("cannot encode data: %w", err)
	}

	return nil
}

func serializeFile(data *file.Data, f ExportFile) error {
	var serErr error

	ext := filepath.Ext(f.Name())
	switch ext {
	case ".yaml", ".yml":
		const yamlIndent = 2

		printExportFileHeader(f, "# ")
		encoder := yaml.NewEncoder(f,
			yaml.Indent(yamlIndent),
			yaml.CustomMarshaler(func(num json.Number) ([]byte, error) {
				return []byte(num), nil
			}),
		)
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
