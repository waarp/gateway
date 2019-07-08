package database

import (
	"fmt"

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
	user := fmt.Sprintf("user='%s'", config.User)
	pass := fmt.Sprintf("password='%s'", config.Password)
	host := fmt.Sprintf("host='%s'", config.Address)
	db := fmt.Sprintf("dbname='%s'", config.Name)
	var port string
	if config.Port != 0 {
		port = fmt.Sprintf(" port=%v", config.Port)
	}

	return fmt.Sprintf("%s %s %s%s %s", user, pass, host, port, db)
}
