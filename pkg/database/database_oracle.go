// +build  oracle

package database

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	_ "github.com/mattn/go-oci8" // register the oracledb driver
)

const (
	// Configuration option for using the OracleDB RDBMS
	oracle = "oracledb"

	// Name of the OracleDB database driver
	oracleDriver = "oci8"
)

func init() {
	supportedRBMS[oracle] = oracleinfo
}

func oracleinfo(config conf.DatabaseConfig) (string, string) {
	return oracleDriver, oracleDSN(config)
}

func oracleDSN(config conf.DatabaseConfig) string {
	var pass, port string
	if config.Password != "" {
		pass = fmt.Sprintf("/%s", config.Password)
	}
	if config.Port != 0 {
		port = fmt.Sprintf(":%v", config.Port)
	}

	return fmt.Sprintf("%s%s@%s%s/%s", config.User, pass, config.Address, port, config.Name)
}
