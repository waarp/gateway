package wgd

import (
	"database/sql"
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

//nolint:lll //struct tags are long
type SQLCommand struct {
	ConfigFile string `short:"c" long:"config" description:"The configuration file to use"`
	Select     bool   `short:"s" long:"select" description:"Execute the query as a SELECT (i.e a query with an output)"`
	Verbose    []bool `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity"`
	Args       struct {
		Query string `required:"yes" positional-arg-name:"query" description:"The SQL query to execute"`
	} `positional-args:"yes"`
}

func (s *SQLCommand) Execute([]string) error {
	config, confErr := conf.ParseServerConfig(s.ConfigFile)
	if confErr != nil {
		return fmt.Errorf("cannot load server config: %w", confErr)
	}

	db, dbErr := s.OpenDB(&config.Database)
	if dbErr != nil {
		return fmt.Errorf("cannot open database: %w", dbErr)
	}

	return s.Run(db)
}

func (s *SQLCommand) Run(db *sql.DB) error {
	if !s.Select {
		if _, err := db.Exec(s.Args.Query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}

		return nil
	}

	return s.runQuery(db)
}

func (s *SQLCommand) OpenDB(config *conf.DatabaseConfig) (*sql.DB, error) {
	var driver, dsn string

	switch dbKind := config.Type; dbKind {
	case migrations.SQLite:
		driver = migrations.SqliteDriver
		dsn = migrations.SqliteDSN(config)
	case migrations.PostgreSQL:
		driver = migrations.PostgresDriver
		dsn = migrations.PostgresDSN(config)
	case migrations.MySQL:
		driver = migrations.MysqlDriver
		dsn = migrations.MysqlDSN(config)
	default:
		//nolint:err113 //this is a base error
		return nil, fmt.Errorf("unsupported database type %q", dbKind)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}

	return db, nil
}

func (s *SQLCommand) runQuery(db *sql.DB) error {
	rows, queryErr := db.Query(s.Args.Query)
	if queryErr != nil {
		return fmt.Errorf("failed to execute query: %w", queryErr)
	}
	defer rows.Close()

	cols, colErr := rows.Columns()
	if colErr != nil {
		return fmt.Errorf("failed to retrieve columns names: %w", colErr)
	}

	res, scanErr := s.scanStringSlices(cols, rows)
	if scanErr != nil {
		return fmt.Errorf("failed to scan query result: %w", scanErr)
	}

	s.displayResult(cols, res)

	return nil
}

func (s *SQLCommand) scanStringSlices(cols []string, rows *sql.Rows) ([][]string, error) {
	row := make([]any, len(cols))
	rowPtrs := make([]any, len(cols))

	for i := range row {
		rowPtrs[i] = &row[i]
	}

	var res [][]string

	for rows.Next() {
		if err := rows.Scan(rowPtrs...); err != nil {
			return nil, err //nolint:wrapcheck //wrapping adds nothing here
		}

		rowStr := make([]string, len(cols))
		for i, val := range row {
			if b, ok := val.([]byte); ok {
				rowStr[i] = string(b)
			} else {
				rowStr[i] = fmt.Sprintf("%v", val)
			}
		}

		res = append(res, rowStr)
	}

	return res, nil
}

//nolint:forbidigo //we need to output to stdout here
func (s *SQLCommand) displayResult(cols []string, res [][]string) {
	if len(res) == 0 {
		fmt.Println("No result")

		return
	}

	const padding = 2

	colsLen := make([]int, len(cols))

	for i, col := range cols {
		colsLen[i] = len(col)
	}

	for _, row := range res {
		for i, val := range row {
			if len(val) > colsLen[i] {
				colsLen[i] = len(val)
			}
		}
	}

	for i, col := range cols {
		fmt.Printf(fmt.Sprintf("%%-%ds", colsLen[i]+padding), col)
	}

	fmt.Println()

	for _, colLen := range colsLen {
		fmt.Print(strings.Repeat("-", colLen) + "  ")
	}

	fmt.Println()

	for _, row := range res {
		for i, val := range row {
			fmt.Printf(fmt.Sprintf("%%-%ds", colsLen[i]+padding), val)
		}

		fmt.Println()
	}
}
