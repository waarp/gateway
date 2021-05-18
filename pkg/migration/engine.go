package migration

import (
	"database/sql"
	"fmt"
	"io"
)

// Engine is an object which can execute a series of migrations, using the Run
// function. It requires an *sql.DB, an SQL dialect to initiate.
type Engine struct {
	db     *sql.DB
	out    io.Writer
	constr func(*queryWriter) Dialect
}

// NewEngine returns a nex Engine instantiated with the correct dialect translator.
func NewEngine(db *sql.DB, dialect string, out io.Writer) (*Engine, error) {
	constr, ok := dialects[dialect]
	if !ok {
		return nil, fmt.Errorf("unknown SQL dialect %s", dialect)
	}

	return &Engine{db: db, constr: constr, out: out}, nil
}

// Upgrade takes a slice of migrations, and executes them sequentially by calling
// all their Up functions in order. All the migrations are run inside a single
// transaction, and if any of them fails, the whole transaction will be cancelled.
func (e *Engine) Upgrade(migrations []Migration) (txErr error) {
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if txErr != nil {
			_ = tx.Rollback()
		}
	}()
	dialect := e.constr(&queryWriter{db: tx, writer: e.out})

	for _, m := range migrations {
		if txErr = m.Script.Up(dialect); txErr != nil {
			return
		}
	}
	txErr = tx.Commit()
	return
}

// Downgrade takes a slice of migrations, and "reverts" them by calling all their
// Down functions in reverse order, starting from the last one. All the migrations
// are run inside a single transaction, and if any of them fails, the whole
// transaction will be cancelled.
func (e *Engine) Downgrade(migrations []Migration) (txErr error) {
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if txErr != nil {
			_ = tx.Rollback()
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
