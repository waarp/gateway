package database

import (
	"fmt"
	"regexp"
	"strings"

	"code.waarp.fr/lib/log"
	"xorm.io/builder"
	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	vers "code.waarp.fr/apps/gateway/gateway/pkg/version"
)

var errBadVersion = fmt.Errorf("database version mismatch")

type exister interface {
	Table
	Identifier
}

func (db *DB) checkVersion() error {
	dbVer := &version{}

	ok, err := db.engine.IsTableExist(dbVer.TableName())
	if err != nil {
		db.logger.Error("Failed to query database version table: %v", err)

		return NewInternalError(err)
	}

	if !ok {
		return nil
	}

	if err := db.Get(dbVer, "").Run(); err != nil {
		db.logger.Error("Failed to retrieve database version: %v", err)

		return err
	}

	if dbVer.Current != vers.Num {
		db.logger.Critical("Mismatch between database (%s) and program (%s) versions.",
			dbVer.Current, vers.Num)

		return errBadVersion
	}

	return nil
}

func checkExists(db Access, bean exister) Error {
	logger := db.GetLogger()

	exist, err := db.getUnderlying().NoAutoCondition().ID(bean.GetID()).Exist(bean)
	if err != nil {
		logger.Error("Failed to check if the %s exists: %s", bean.Appellation(), err)

		return NewInternalError(err)
	}

	if !exist {
		logger.Debug("No %s found with ID %d", bean.Appellation(), bean.GetID())

		return NewNotFoundError(bean)
	}

	return nil
}

// logSQL logs the last executed SQL command.
func logSQL(query *xorm.Session, logger *log.Logger) {
	sql, args := query.LastSQL()
	if len(args) == 0 {
		logger.Trace("[SQL] %s", sql)

		return
	}

	for i := range args {
		if args[i] == nil {
			args[i] = "<nil>"
		}
	}

	if !strings.Contains(sql, "?") {
		reg := regexp.MustCompile(`\$\d`)
		sql = reg.ReplaceAllLiteralString(sql, "?")
	}

	sqlMsg, err := builder.ConvertToBoundSQL(sql, args)
	if err == nil {
		logger.Trace("[SQL] %s", sqlMsg)

		return
	}
}

// ping checks if the database is reachable and updates the service state accordingly.
// If an error occurs while contacting the database, that error is returned.
func ping(dbState *state.State, ses *xorm.Session, logger *log.Logger) Error {
	if err := ses.Ping(); err != nil {
		logger.Critical("Could not reach database: %s", err.Error())
		dbState.Set(state.Error, err.Error())

		return NewInternalError(err)
	}

	dbState.Set(state.Running, "")

	return nil
}

type inCond struct {
	*strings.Builder
	args []interface{}
}

func (i *inCond) Append(args ...interface{}) {
	i.args = append(i.args, args...)
}
