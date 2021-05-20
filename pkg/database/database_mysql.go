package database

import (
	"crypto/tls"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	msql "github.com/go-sql-driver/mysql" // register the mysql driver
	"xorm.io/xorm"
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

func mysqlinfo(config conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return mysqlDriver, mysqlDSN(config), func(db *xorm.Engine) error {
		db.DatabaseTZ = time.UTC
		return nil
	}
}

func mysqlDSN(config conf.DatabaseConfig) string {
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

	//dsn.Params = map[string]string{"time_zone": "'+00:00'"}

	return dsn.FormatDSN()
}
