package model

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"

	"github.com/bwmarrin/snowflake"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// getCryptos fetch from the database then return the associated Cryptos if they exist.
func getCryptos(db database.ReadAccess, owner CryptoOwner) (Cryptos, error) {
	var certs Cryptos
	query := db.Select(&certs).Where(owner.GenCryptoSelectCond())

	if err := query.Run(); err != nil {
		return nil, err
	}

	// TODO: get only validate certificates
	return certs, nil
}

func getAuthorizedRules(db database.ReadAccess, ownerCol string, ownerID int64,
) (Rules, error) {
	tbl := TableRuleAccesses

	var rules Rules
	if err := db.Select(&rules).Where(
		"(id IN (SELECT DISTINCT rule_id FROM "+tbl+" WHERE "+ownerCol+"=?))"+
			" OR "+
			"(SELECT COUNT(*) FROM "+tbl+" WHERE rule_id = id) = 0",
		ownerID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the authorized rules: %w", err)
	}

	return rules, nil
}

// CheckClientAuthent checks whether the given certificate chain is valid as a
// client certificate for the given login, using the target cryptos as root CAs.
func (c *Cryptos) CheckClientAuthent(login string, certs []*x509.Certificate) error {
	return c.checkAuthent(login, certs, x509.ExtKeyUsageClientAuth)
}

// CheckServerAuthent checks whether the given certificate chain is valid as a
// server certificate for the given host, using the target cryptos as root CAs.
func (c *Cryptos) CheckServerAuthent(host string, certs []*x509.Certificate) error {
	return c.checkAuthent(host, certs, x509.ExtKeyUsageServerAuth)
}

func (c *Cryptos) checkAuthent(name string, certs []*x509.Certificate,
	usages ...x509.ExtKeyUsage,
) error {
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}

	for _, crypto := range *c {
		chain, err := utils.ParsePEMCertChain(crypto.Certificate)
		if err != nil {
			return fmt.Errorf("failed to parse trusted certificate: %w", err)
		}

		if isLegacyR66Cert(chain[0]) && isLegacyR66Cert(certs[0]) {
			return nil
		}

		roots.AppendCertsFromPEM([]byte(crypto.Certificate))
	}

	intermediate := x509.NewCertPool()
	for _, cert := range certs {
		intermediate.AddCert(cert)
	}

	opt := x509.VerifyOptions{
		DNSName:       name,
		Roots:         roots,
		Intermediates: intermediate,
		KeyUsages:     usages,
	}

	if _, err := certs[0].Verify(opt); err != nil {
		return fmt.Errorf("invalid certificate: %w", err)
	}

	return nil
}

func getTransferInfo(db database.ReadAccess, id int64) (map[string]interface{}, database.Error) {
	var infoList TransferInfoList
	if err := db.Select(&infoList).Where("transfer_id=?", id).Run(); err != nil {
		return nil, err
	}

	infoMap := map[string]interface{}{}

	for _, info := range infoList {
		var val interface{}
		if err := json.Unmarshal([]byte(info.Value), &val); err != nil {
			return nil, database.NewValidationError("invalid transfer info value '%s': %s", info.Value, err)
		}

		infoMap[info.Name] = val
	}

	return infoMap, nil
}

// SetTransferInfo replaces all the TransferInfo in the database of the given
// transfer by those given in the map parameter.
func setTransferInfo(db *database.DB, info map[string]interface{}, transID int64,
	isHistory bool,
) database.Error {
	return db.Transaction(func(ses *database.Session) database.Error {
		if err := ses.DeleteAll(&TransferInfo{}).Where("transfer_id=?", transID).Run(); err != nil {
			return err
		}
		for name, val := range info {
			str, err := json.Marshal(val)
			if err != nil {
				return database.NewValidationError("invalid transfer info value '%v': %s", val, err)
			}

			i := &TransferInfo{
				Name:  name,
				Value: string(str),
			}

			if isHistory {
				i.HistoryID = utils.NewNullInt64(transID)
			} else {
				i.TransferID = utils.NewNullInt64(transID)
			}

			if err := ses.Insert(i).Run(); err != nil {
				return err
			}
		}

		return nil
	})
}

func makeIDGenerator() (*snowflake.Node, error) {
	var nodeID, mod, machineID big.Int

	nodeID.SetBytes([]byte(conf.GlobalConfig.NodeID))
	mod.SetInt64(math.MaxInt64)

	machineID.Mod(&nodeID, &mod)

	generator, err := snowflake.NewNode(machineID.Int64())
	if err != nil {
		return nil, fmt.Errorf("failed to create the ID generator: %w", err)
	}

	return generator, nil
}

const AllowLegacyR66CertificateVar = "WAARP_GATEWAY_ALLOW_LEGACY_CERT"

//nolint:gochecknoglobals //global vars are required here
var (
	IsLegacyR66CertificateAllowed = os.Getenv(AllowLegacyR66CertificateVar) == "1"
	waarpR66LegacyCertSignature   = []byte{
		72, 215, 182, 129, 207, 52, 89, 161, 3, 252, 5, 34, 211, 8, 242, 9, 128, 10,
		99, 160, 61, 3, 88, 159, 19, 200, 166, 251, 234, 226, 192, 13, 196, 213, 170,
		215, 56, 7, 89, 187, 118, 45, 1, 232, 244, 246, 116, 132, 251, 248, 78, 4,
		147, 75, 112, 58, 247, 233, 83, 35, 220, 128, 213, 51, 209, 171, 128, 124,
		17, 118, 236, 242, 76, 74, 237, 161, 186, 15, 71, 117, 41, 221, 188, 80,
		113, 104, 48, 13, 9, 88, 245, 31, 180, 190, 66, 4, 41, 197, 5, 205, 179, 167,
		125, 155, 7, 233, 200, 228, 191, 72, 34, 132, 237, 124, 182, 235, 249, 10,
		109, 13, 104, 90, 138, 118, 129, 94, 240, 255, 237, 11, 28, 175, 64, 1, 219,
		15, 14, 74, 19, 196, 246, 69, 112, 244, 187, 145, 156, 32, 249, 224, 40, 191,
		224, 196, 58, 75, 14, 145, 103, 135, 27, 42, 93, 20, 75, 39, 225, 26, 147,
		235, 180, 97, 120, 39, 142, 102, 200, 132, 15, 140, 225, 0, 60, 29, 93, 220,
		110, 219, 228, 24, 149, 44, 55, 167, 251, 238, 174, 32, 186, 20, 69, 137,
		224, 78, 204, 60, 198, 197, 77, 70, 218, 199, 118, 113, 237, 232, 239, 179,
		199, 191, 14, 7, 227, 101, 145, 228, 194, 65, 93, 73, 18, 115, 244, 33, 122,
		208, 234, 62, 126, 172, 169, 253, 59, 223, 51, 250, 119, 74, 142, 86, 230,
		64, 44, 244, 194, 236,
	}
)

func isLegacyR66Cert(cert *x509.Certificate) bool {
	if IsLegacyR66CertificateAllowed &&
		cert.SignatureAlgorithm == x509.SHA256WithRSA &&
		bytes.Equal(cert.Signature, waarpR66LegacyCertSignature) {
		return true
	}

	return false
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// Can be used to redefine column when the definition generated by Xorm is not
// good enough.
func redefineColumn(db *database.Executor, table, col, colDef string) database.Error {
	if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, col)); err != nil {
		db.Logger.Error("Failed to drop column %s.%s: %s", table, col, err)

		return database.NewInternalError(err)
	}

	if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, col, colDef)); err != nil {
		db.Logger.Error("Failed to re-add column %s.%s: %s", table, col, err)

		return database.NewInternalError(err)
	}

	return nil
}

func addTableConstraint(db *database.Executor, table, cons string) database.Error {
	if db.Dialect != database.SQLite {
		if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD %s`, table, cons)); err != nil {
			db.Logger.Error("Failed to add new constraint to table %s: %s", table, err)

			return database.NewInternalError(err)
		}
	}

	return addSqliteTableConstraint(db, table, cons)
}

//nolint:funlen //splitting the function would hurt
func addSqliteTableConstraint(db *database.Executor, table, cons string) database.Error {
	// SQLite alter table procedure
	res, err1 := db.Query("PRAGMA schema_version") // query the schema version
	if err1 != nil {
		db.Logger.Error("Failed to add new constraint to table %s: "+
			"failed to retrieve schema version: %s", table, err1)

		return database.NewInternalError(err1)
	}

	//nolint:errcheck,forcetypeassert //schema version is always an int64, no need to check
	vers := res[0]["schema_version"].(int64) // save the schema version

	if _, err := db.Exec("PRAGMA writable_schema=ON"); err != nil { // allow schema editing
		db.Logger.Error("Failed to add new constraint to table %s: "+
			"failed to allow schema editing: %s", table, err)

		return database.NewInternalError(err)
	}

	res2, err2 := db.Query(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table)
	if err2 != nil {
		db.Logger.Error("Failed to add new constraint to table %s: "+
			"failed to retrieve the table schema: %s", table, err2)

		return database.NewInternalError(err2)
	}

	//nolint:errcheck,forcetypeassert //sql script is always a string, no need to check
	script := res2[0]["sql"].(string)
	script = strings.TrimSuffix(strings.TrimSpace(script), ")") + ", " + cons + ")"

	if _, err := db.Exec(`UPDATE sqlite_schema SET sql=? WHERE type='table' AND name=?`,
		script, cons, table); err != nil { // add the new constraint to the table schema
		db.Logger.Error("Failed to add new constraint to table %s: "+
			"failed to edit the table schema: %s", table, err)

		return database.NewInternalError(err)
	}

	vers++
	if _, err := db.Exec(fmt.Sprintf("PRAGMA schema_version=%d", vers)); err != nil { // increment the schema version
		db.Logger.Error("Failed to add new constraint to table %s: "+
			"failed to increment schema version: %s", table, err)

		return database.NewInternalError(err)
	}

	if _, err := db.Exec("PRAGMA writable_schema=OFF"); err != nil { // disable schema editing
		db.Logger.Error("Failed to add new constraint to table %s: "+
			"failed to disable schema editing: %s", table, err)

		return database.NewInternalError(err)
	}

	if _, err := db.Exec("PRAGMA integrity_check"); err != nil { // run an integrity check on the new schema
		db.Logger.Error("Failed to add new constraint to table %s: "+
			"integrity check failed: %s", table, err)

		return database.NewInternalError(err)
	}

	return nil
}
