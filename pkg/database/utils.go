package database

import (
	"errors"
	"strings"

	"code.waarp.fr/lib/log"
	"xorm.io/xorm"
	xLog "xorm.io/xorm/log"
	"xorm.io/xorm/names"

	vers "code.waarp.fr/apps/gateway/gateway/pkg/version"
)

var errBadVersion = errors.New("database version mismatch")

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

func checkExists(db Access, bean exister) error {
	logger := db.GetLogger()

	exist, err := db.getUnderlying().NoAutoCondition().
		Where("id=?", bean.GetID()).Exist(bean)
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

type inCond struct {
	*strings.Builder
	args []interface{}
}

func (i *inCond) Append(args ...interface{}) {
	i.args = append(i.args, args...)
}

func exec(ses *xorm.Session, logger *log.Logger, query string, args ...interface{}) error {
	elems := append([]interface{}{query}, args...)

	if _, err := ses.Exec(elems...); err != nil {
		logger.Error("Failed to execute the query: %s", err)

		return NewInternalError(err)
	}

	return nil
}

func (db *DB) setLogger(engine *xorm.Engine) {
	xormLogger := xLog.NewSimpleLogger2(db.logger.AsStdLogger(log.LevelTrace).
		Writer(), "", 0)

	xormLogger.ERR.SetPrefix("xorm: ")
	xormLogger.WARN.SetPrefix("xorm: ")
	xormLogger.INFO.SetPrefix("xorm: ")
	xormLogger.ERR.SetPrefix("xorm: ")

	engine.SetLogger(xormLogger)
	engine.SetMapper(names.GonicMapper{})
	engine.ShowSQL(true)
}
