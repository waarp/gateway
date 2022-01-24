package migration

import (
	"database/sql"
	"fmt"
	"io"
)

// Engine is an object which can execute a series of migrations, using the Run
// function. It requires an *sql.DB, an SQL dialect to initiate.
type Engine struct {
	DB     *sql.DB
	out    io.Writer
	constr func(*queryWriter) Actions
}

// NewEngine returns a nex Engine instantiated with the correct dialect translator.
func NewEngine(db *sql.DB, dialect string, out io.Writer) (*Engine, error) {
	constr, ok := dialects[dialect]
	if !ok {
		return nil, fmt.Errorf("unknown SQL dialect %s", dialect) //nolint:goerr113 // base error
	}

	return &Engine{DB: db, constr: constr, out: out}, nil
}

// Upgrade takes a slice of migrations, and executes them sequentially by calling
// all their Up functions in order. All the migrations are run inside a single
// transaction, and if any of them fails, the whole transaction will be canceled.
func (e *Engine) Upgrade(migrations []Migration) (txErr error) {
	tx, err := e.DB.Begin()
	if err != nil {
		return fmt.Errorf("cannot open transaction: %w", err)
	}

	defer func() {
		if txErr != nil {
			_ = tx.Rollback() //nolint:errcheck // no logger to handle the error
		}
	}()

	dialect := e.constr(&queryWriter{db: tx, writer: e.out})

	for i := range migrations {
		if txErr = migrations[i].Script.Up(dialect); txErr != nil {
			return
		}
	}

	txErr = tx.Commit()

	return
}

// Downgrade takes a slice of migrations, and "reverts" them by calling all their
// Down functions in reverse order, starting from the last one. All the migrations
// are run inside a single transaction, and if any of them fails, the whole
// transaction will be canceled.
func (e *Engine) Downgrade(migrations []Migration) (txErr error) {
	tx, err := e.DB.Begin()
	if err != nil {
		return fmt.Errorf("cannot open transaction: %w", err)
	}

	defer func() {
		if txErr != nil {
			_ = tx.Rollback() //nolint:errcheck // no logger to handle the error
		}
	}()

	dialect := e.constr(&queryWriter{db: tx, writer: e.out})

	for i := len(migrations) - 1; i >= 0; i-- {
		if txErr = migrations[i].Script.Down(dialect); txErr != nil {
			return
		}
	}

	txErr = tx.Commit()

	return
}
