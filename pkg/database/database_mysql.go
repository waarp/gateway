package database

import (
	"crypto/tls"
	"time"

	// Register the mysql driver.
	msql "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

const (
	// MySQL is the configuration option for using the MySQL RDBMS.
	MySQL = "mysql"

	// MysqlDriver is the name of the MySQL database driver.
	MysqlDriver = "mysql"
)

//nolint:gochecknoinits // init is used by design
func init() {
	supportedRBMS[MySQL] = mysqlinfo
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
		cert, _ := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey) //nolint:errcheck // nothing to handle the errors
		tlsConf := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}

		_ = msql.RegisterTLSConfig("db", tlsConf) //nolint:errcheck // nothing to handle the errors

		dsn.TLSConfig = "db"
	}

	return dsn.FormatDSN()
}
