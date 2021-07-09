package database

import (
	"crypto/tls"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	msql "github.com/go-sql-driver/mysql" // register the mysql driver
	"xorm.io/xorm"
)

const (
	// Configuration option for using the MySQL RDBMS
	mysql = migration.MySQL

	// MysqlDriver is the name of the MySQL database driver
	MysqlDriver = "mysql"
)

func init() {
	supportedRBMS[mysql] = mysqlinfo
}

func mysqlinfo(config *conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return MysqlDriver, MysqlDSN(config), func(db *xorm.Engine) error {
		db.DatabaseTZ = time.UTC
		return nil
	}
}

// MysqlDSN takes a database configuration and returns the corresponding MySQL
// DSN necessary to connect to the database.
func MysqlDSN(config *conf.DatabaseConfig) string {
	dsn := msql.NewConfig()
	dsn.Addr = config.Address
	dsn.DBName = config.Name
	dsn.User = config.User
	dsn.Passwd = config.Password

	if config.TLSCert != "" && config.TLSKey != "" {
		cert, _ := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
		tlsConf := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
		_ = msql.RegisterTLSConfig("db", tlsConf)

		dsn.TLSConfig = "db"
	}

	return dsn.FormatDSN()
}
