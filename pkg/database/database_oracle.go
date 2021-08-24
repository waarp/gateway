// +build  oracle

package database

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	_ "github.com/mattn/go-oci8" // register the oracledb driver
	"xorm.io/xorm"
)

const (
	// Configuration option for using the OracleDB RDBMS
	oracle = "oracledb"

	// OracleDriver is the name of the OracleDB database driver
	OracleDriver = "oci8"
)

func init() {
	supportedRBMS[oracle] = oracleinfo
}

func oracleinfo() (string, string, func(*xorm.Engine) error) {
	return OracleDriver, oracleDSN(), func(*xorm.Engine) error {
		return nil
	}
}

func oracleDSN() string {
	config := &conf.GlobalConfig.Database
	var pass string
	if config.Password != "" {
		pass = fmt.Sprintf("/%s", config.Password)
	}

	return fmt.Sprintf("%s%s@%s/%s", config.User, pass, config.Address, config.Name)
}
