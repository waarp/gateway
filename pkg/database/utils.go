package database

import (
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/go-xorm/builder"
	"github.com/go-xorm/xorm"
)

// logSQL logs the last executed SQL command.
func logSQL(query *xorm.Session, logger *log.Logger) {
	msg, err := builder.ConvertToBoundSQL(query.LastSQL())
	if err != nil {
		logger.Warningf("Failed to log SQL command: %s", err)
	} else {
		logger.Debug(msg)
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
