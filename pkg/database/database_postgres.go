package database

import (
	"fmt"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	_ "github.com/lib/pq" // register the postgres driver
)

const (
	// Configuration option for using the PostgreSQL RDBMS
	postgres = "postgresql"

	// Name of the PostgreSQL database driver
	postgresDriver = "postgres"
)

func init() {
	supportedRBMS[postgres] = postgresinfo
}

func postgresinfo(config conf.DatabaseConfig) (string, string) {
	return postgresDriver, postgresDSN(config)
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
		dns = append(dns, fmt.Sprintf("host='%s'", config.Address))
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
