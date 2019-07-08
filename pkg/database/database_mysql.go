package database

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	_ "github.com/go-sql-driver/mysql" // register the mysql driver
)

const (
	// Configuration option for using the MySQL RDBMS
	mysql = "mysql"

	// Name of the MySQL database driver
	mysqlDriver = "mysql"
)

func init() {
	supportedRBMS[mysql] = mysqlinfo
}

func mysqlinfo(config conf.DatabaseConfig) (string, string) {
	return mysqlDriver, mysqlDSN(config)
}

func mysqlDSN(config conf.DatabaseConfig) string {
	var pass, port string
	if config.Password != "" {
		pass = fmt.Sprintf(":%s", config.Password)
	}
	if config.Port != 0 {
		port = fmt.Sprintf(":%v", config.Port)
	}

	return fmt.Sprintf("%s%s@(%s%s)/%s", config.User, pass, config.Address, port, config.Name)
}
