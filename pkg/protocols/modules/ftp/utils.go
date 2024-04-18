package ftp

import (
	"fmt"
	"math/rand"
	"net"
	"strings"

	"code.waarp.fr/lib/log"
	ftplog "github.com/fclairamb/go-log"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func getPortInRange(addr string, minPort, maxPort uint16) uint16 {
	const (
		minNbTries = 10
		maxNbTries = 100
	)

	rangeSize := int(maxPort - minPort)
	nbTries := rangeSize
	nbTries = utils.Min(nbTries, maxNbTries)
	nbTries = utils.Max(nbTries, minNbTries)

	for i := 0; i < nbTries; i++ {
		//nolint:gosec //we don't need to be secure, we just need a random port
		candidate := minPort + uint16(rand.Intn(rangeSize))

		if _, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, i)); err == nil {
			return candidate
		}
	}

	return 0
}

type ftpServerLogger struct {
	*log.Logger
}

func (f *ftpServerLogger) log(msg string, keyvals []any) {
	if len(keyvals) == 0 {
		f.Trace(msg)

		return
	}

	var args []string

	for i := 0; i < len(keyvals); i += 2 {
		if i+1 >= len(keyvals) {
			args = append(args, fmt.Sprint(keyvals[i]))
		} else {
			args = append(args, fmt.Sprintf(`%v="%v"`, keyvals[i], keyvals[i+1]))
		}
	}

	f.Trace("%s %s", msg, strings.Join(args, ", "))
}

func (f *ftpServerLogger) Debug(msg string, keyvals ...any) { f.log(msg, keyvals) }
func (f *ftpServerLogger) Info(msg string, keyvals ...any)  { f.log(msg, keyvals) }
func (f *ftpServerLogger) Warn(msg string, keyvals ...any)  { f.log(msg, keyvals) }
func (f *ftpServerLogger) Error(msg string, keyvals ...any) { f.log(msg, keyvals) }
func (f *ftpServerLogger) Panic(msg string, keyvals ...any) { f.log(msg, keyvals) }
func (f *ftpServerLogger) With(args ...any) ftplog.Logger   { return f }
