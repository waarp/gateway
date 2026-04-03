package ebics

import (
	"context"
	"time"

	libebics "code.waarp.fr/lib/ebics/ebics"
	"code.waarp.fr/lib/ebics/ebics/returncode"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

type serverObserver struct {
	logger *log.Logger
}

func newServerObserver(logger *log.Logger) *serverObserver {
	return &serverObserver{logger: logger}
}

//nolint:gocritic // lib-ebics observer interface imposes value semantics here.
func (o *serverObserver) OnRequestStart(_ context.Context, _ libebics.OrderContext) {}

//nolint:gocritic // lib-ebics observer interface imposes value semantics here.
func (o *serverObserver) OnRequestEnd(
	_ context.Context,
	req libebics.OrderContext,
	code returncode.Code,
	duration time.Duration,
	err error,
) {
	if o == nil || o.logger == nil {
		return
	}
	if err == nil {
		return
	}

	o.logger.Warningf(
		"EBICS server request failed order=%s host=%s partner=%s user=%s request=%s code=%s dur=%s err=%v",
		req.OrderType,
		req.HostID,
		req.PartnerID,
		req.UserID,
		req.OrderID,
		code.Value,
		duration,
		err,
	)
}
