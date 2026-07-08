package protoutils

import (
	"context"
	"log/slog"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

type leveledHandler struct {
	slog.Handler

	from, to slog.Level
}

//nolint:gocritic //cannot change method signature
func (l *leveledHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level == l.from {
		record.Level = l.to
	}

	//nolint:wrapcheck //no need to wrap here
	return l.Handler.Handle(ctx, record)
}

func LibSLogger(output *log.Logger, from, to log.Level) *slog.Logger {
	handler := &leveledHandler{
		Handler: output.Slogger().Handler(),
		from:    slog.Level(from),
		to:      slog.Level(to),
	}

	return slog.New(handler)
}
