package snmp

import (
	"fmt"
	golog "log"

	"github.com/gosnmp/gosnmp"
	snmplib "github.com/slayercat/GoSNMPServer"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

type clientLogger struct {
	*golog.Logger
}

func clientLog(gwLogger *log.Logger) gosnmp.Logger {
	return gosnmp.NewLogger(&clientLogger{Logger: gwLogger.AsStdLogger(log.LevelTrace)})
}

func (l *clientLogger) Printf(msg string, v ...any) { l.Print(fmt.Sprintf(msg, v...)) }

type serverLogger struct {
	*log.Logger
}

func serverLog(gwLogger *log.Logger) snmplib.ILogger {
	return &serverLogger{Logger: gwLogger}
}

func (s *serverLogger) Debug(args ...any)                { s.Logger.Debug(fmt.Sprint(args...)) }
func (s *serverLogger) Debugf(msg string, args ...any)   { s.Logger.Debugf(msg, args...) }
func (s *serverLogger) Debugln(args ...any)              { s.Logger.Debug(fmt.Sprintln(args...)) }
func (s *serverLogger) Error(args ...any)                { s.Logger.Error(fmt.Sprint(args...)) }
func (s *serverLogger) Errorf(msg string, args ...any)   { s.Logger.Errorf(msg, args...) }
func (s *serverLogger) Errorln(args ...any)              { s.Logger.Error(fmt.Sprintln(args...)) }
func (s *serverLogger) Fatal(args ...any)                { s.Logger.Fatal(fmt.Sprint(args...)) }
func (s *serverLogger) Fatalf(msg string, args ...any)   { s.Logger.Fatalf(msg, args...) }
func (s *serverLogger) Fatalln(args ...any)              { s.Logger.Fatal(fmt.Sprintln(args...)) }
func (s *serverLogger) Info(args ...any)                 { s.Logger.Info(fmt.Sprint(args...)) }
func (s *serverLogger) Infof(msg string, args ...any)    { s.Logger.Infof(msg, args...) }
func (s *serverLogger) Infoln(args ...any)               { s.Logger.Info(fmt.Sprintln(args...)) }
func (s *serverLogger) Trace(args ...any)                { s.Logger.Trace(fmt.Sprint(args...)) }
func (s *serverLogger) Tracef(msg string, args ...any)   { s.Logger.Tracef(msg, args...) }
func (s *serverLogger) Traceln(args ...any)              { s.Logger.Trace(fmt.Sprintln(args...)) }
func (s *serverLogger) Warn(args ...any)                 { s.Logger.Warning(fmt.Sprint(args...)) }
func (s *serverLogger) Warnf(msg string, args ...any)    { s.Logger.Warningf(msg, args...) }
func (s *serverLogger) Warnln(args ...any)               { s.Logger.Warning(fmt.Sprintln(args...)) }
func (s *serverLogger) Warning(args ...any)              { s.Logger.Warning(fmt.Sprint(args...)) }
func (s *serverLogger) Warningf(msg string, args ...any) { s.Logger.Warningf(msg, args...) }
func (s *serverLogger) Warningln(args ...any)            { s.Logger.Warning(fmt.Sprintln(args...)) }
