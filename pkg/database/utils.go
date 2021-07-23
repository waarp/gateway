package database

import (
	"regexp"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type exister interface {
	Table
	Identifier
}

func checkExists(db Access, bean exister) Error {
	logger := db.GetLogger()
	exist, err := db.getUnderlying().NoAutoCondition().ID(bean.GetID()).Exist(bean)
	if err != nil {
		logger.Errorf("Failed to check if the %s exists: %s", bean.Appellation(), err)
		return NewInternalError(err)
	}
	if !exist {
		logger.Debugf("No %s found with ID %d", bean.Appellation(), bean.GetID())
		return NewNotFoundError(bean)
	}
	return nil
}

// logSQL logs the last executed SQL command.
func logSQL(query *xorm.Session, logger *log.Logger) {
	sql, args := query.LastSQL()
	if len(args) == 0 {
		logger.Debugf("[SQL] %s", sql)
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
		logger.Debugf("[SQL] %s", sqlMsg)
		return
	}
}

// ping checks if the database is reachable and updates the service state accordingly.
// If an error occurs while contacting the database, that error is returned.
func ping(state *service.State, ses *xorm.Session, logger *log.Logger) Error {
	if err := ses.Ping(); err != nil {
		logger.Criticalf("Could not reach database: %s", err.Error())
		state.Set(service.Error, err.Error())
		return NewInternalError(err)
	}
	state.Set(service.Running, "")
	return nil
}

type inCond struct {
	*strings.Builder
	args []interface{}
}

func (i *inCond) Append(args ...interface{}) {
	i.args = append(i.args, args...)
}
