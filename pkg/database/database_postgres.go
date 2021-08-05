package database

import (
	"fmt"
	"net"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	_ "github.com/jackc/pgx/v4/stdlib" // register the postgres driver
	"xorm.io/xorm"
)

const (
	// PostgreSQL is the configuration option for using the PostgreSQL RDBMS
	PostgreSQL = "postgresql"

	// PostgresDriver is the name of the PostgreSQL database driver
	PostgresDriver = "pgx"
)

func init() {
	supportedRBMS[PostgreSQL] = postgresinfo
}

func postgresinfo() (string, string, func(*xorm.Engine) error) {
	return PostgresDriver, PostgresDSN(), func(*xorm.Engine) error {
		return nil
	}
}

// PostgresDSN takes a database configuration and returns the corresponding
// PostgreSQL DSN necessary to connect to the database.
func PostgresDSN() string {
	config := &conf.GlobalConfig.ServerConf.Database
	dns := []string{}
	if config.User != "" {
		dns = append(dns, fmt.Sprintf("user='%s'", config.User))
	}
	if config.Password != "" {
		dns = append(dns, fmt.Sprintf("password='%s'", config.Password))
	}
	if config.Address != "" {
		host, port, err := net.SplitHostPort(config.Address)
		if err != nil {
			dns = append(dns, fmt.Sprintf("host='%s'", config.Address))
		} else {
			dns = append(dns, fmt.Sprintf("host='%s'", host))
			dns = append(dns, fmt.Sprintf("port='%s'", port))
		}
	}
	if config.Name != "" {
		dns = append(dns, fmt.Sprintf("dbname='%s'", config.Name))
	}
	if config.TLSCert != "" && config.TLSKey != "" {
		dns = append(dns, "sslmode=verify-full")
		dns = append(dns, fmt.Sprintf("sslcert='%s'", config.TLSCert))
		dns = append(dns, fmt.Sprintf("sslkey='%s'", config.TLSKey))
	} else {
		dns = append(dns, "sslmode=disable")
	}

	return strings.Join(dns, " ")
}
