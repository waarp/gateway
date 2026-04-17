package as2

import (
	"log/slog"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

func slogs(logger *log.Logger) *slog.Logger {
	return protoutils.LibSLogger(logger, log.LevelInfo, log.LevelDebug)
}
