package migrations

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	// Register the SQL drivers.
	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "github.com/mattn/go-sqlite3"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

const (
	// SqliteDriver is the name of the SQLite database driver.
	SqliteDriver = "sqlite3"

	// PostgresDriver is the name of the PostgreSQL database driver.
	PostgresDriver = "pgx"

	// MysqlDriver is the name of the MySQL database driver.
	MysqlDriver = "mysql"
)

type dbInfo struct {
	driver  string
	makeDSN func(*conf.DatabaseConfig) string
}

//nolint:gochecknoglobals // global var is used by design
var rdbms = map[string]dbInfo{
	migration.SQLite: {
		driver:  SqliteDriver,
		makeDSN: SqliteDSN,
	},
	migration.PostgreSQL: {
		driver:  PostgresDriver,
		makeDSN: PostgresDSN,
	},
	migration.MySQL: {
		driver:  MysqlDriver,
		makeDSN: MysqlDSN,
	},
}

// SqliteDSN takes a database configuration and returns the corresponding
// SQLite DSN necessary to connect to the database.
func SqliteDSN(config *conf.DatabaseConfig) string {
	var user, pass string
	if config.User != "" {
		user = fmt.Sprintf("&_auth_user=%s", config.User)
	}

	if config.Password != "" {
		pass = fmt.Sprintf("&_auth_pass=%s", config.Password)
	}

	return fmt.Sprintf("file:%s?mode=rwc&_busy_timeout=10000%s%s",
		config.Address, user, pass)
}

// PostgresDSN takes a database configuration and returns the corresponding
// PostgreSQL DSN necessary to connect to the database.
func PostgresDSN(config *conf.DatabaseConfig) string {
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
			dns = append(dns, fmt.Sprintf("host='%s'", host),
				fmt.Sprintf("port='%s'", port))
		}
	}

	if config.Name != "" {
		dns = append(dns, fmt.Sprintf("dbname='%s'", config.Name))
	}

	if config.TLSCert != "" && config.TLSKey != "" {
		dns = append(dns, "sslmode=verify-full",
			fmt.Sprintf("sslcert='%s'", config.TLSCert),
			fmt.Sprintf("sslkey='%s'", config.TLSKey))
	} else {
		dns = append(dns, "sslmode=disable")
	}

	return strings.Join(dns, " ")
}

// MysqlDSN takes a database configuration and returns the corresponding MySQL
// DSN necessary to connect to the database.
func MysqlDSN(config *conf.DatabaseConfig) string {
	dsn := mysql.NewConfig()
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

		_ = mysql.RegisterTLSConfig("db", tlsConf) //nolint:errcheck // nothing to handle the errors

		dsn.TLSConfig = "db"
	}

	return dsn.FormatDSN()
}
