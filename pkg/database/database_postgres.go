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
	// Configuration option for using the PostgreSQL RDBMS
	postgres = "postgresql"

	// Name of the PostgreSQL database driver
	postgresDriver = "pgx"
)

func init() {
	supportedRBMS[postgres] = postgresinfo
}

func postgresinfo(config conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return postgresDriver, postgresDSN(config), func(*xorm.Engine) error {
		return nil
	}
}

func postgresDSN(config conf.DatabaseConfig) string {
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
