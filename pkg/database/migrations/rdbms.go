package migrations

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"

	// Register the SQL drivers.
	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "modernc.org/sqlite"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

const (
	// SqliteDriver is the name of the SQLite database driver.
	SqliteDriver = "sqlite"

	// PostgresDriver is the name of the PostgreSQL database driver.
	PostgresDriver = "pgx"

	// MysqlDriver is the name of the MySQL database driver.
	MysqlDriver = "mysql"
)

type dbInfo struct {
	driver  string
	makeDSN func() string
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
func SqliteDSN() string {
	config := conf.GlobalConfig.Database
	values := url.Values{}

	values.Set("mode", "rwc")
	values.Set("_txlock", "immediate")
	values.Add("_pragma", "busy_timeout=5000")
	values.Add("_pragma", "foreign_keys=ON")
	values.Add("_pragma", "journal_mode=WAL")
	values.Add("_pragma", "synchronous=NORMAL")

	return fmt.Sprintf("%s?%s", config.Address, values.Encode())
}

// PostgresDSN takes a database configuration and returns the corresponding
// PostgreSQL DSN necessary to connect to the database.
func PostgresDSN() string {
	config := &conf.GlobalConfig.Database

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
func MysqlDSN() string {
	config := &conf.GlobalConfig.Database

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
