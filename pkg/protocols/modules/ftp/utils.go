package ftp

import (
	"fmt"
	"math/rand/v2"
	"net"
	"strings"

	"code.waarp.fr/lib/log"
	ftplog "github.com/fclairamb/go-log"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func getPortInRange(addr string, minPort, maxPort uint16) (uint16, *pipeline.Error) {
	if minPort == maxPort {
		return minPort, nil
	} else if minPort > maxPort {
		return 0, pipeline.NewError(types.TeInternal, "invalid port range for active mode")
	}

	const (
		minNbTries = 10
		maxNbTries = 100
	)

	rangeSize := int(maxPort - minPort)
	nbTries := rangeSize
	nbTries = utils.Min(nbTries, maxNbTries)
	nbTries = utils.Max(nbTries, minNbTries)

	for i := range nbTries {
		candidate := minPort + uint16(rand.IntN(rangeSize))

		if list, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, i)); err == nil {
			_ = list.Close() //nolint:errcheck //error is irrelevant

			return candidate, nil
		}
	}

	return 0, pipeline.NewError(types.TeInternal,
		"could not find an available port in the range for active mode")
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

	f.Tracef("%s %s", msg, strings.Join(args, ", "))
}

func (f *ftpServerLogger) Debug(msg string, keyvals ...any) { f.log(msg, keyvals) }
func (f *ftpServerLogger) Info(msg string, keyvals ...any)  { f.log(msg, keyvals) }
func (f *ftpServerLogger) Warn(msg string, keyvals ...any)  { f.log(msg, keyvals) }
func (f *ftpServerLogger) Error(msg string, keyvals ...any) { f.log(msg, keyvals) }
func (f *ftpServerLogger) Panic(msg string, keyvals ...any) { f.log(msg, keyvals) }
func (f *ftpServerLogger) With(...any) ftplog.Logger        { return f }
