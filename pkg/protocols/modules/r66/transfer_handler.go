package r66

import (
	"context"

	"code.waarp.fr/lib/r66"
	r66utils "code.waarp.fr/lib/r66/utils"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type transferHandler struct {
	*sessionHandler
	trans  *serverTransfer
	cancel func(cause error)
}

func (t *transferHandler) GetHash() ([]byte, error) {
	var hash []byte

	err := utils.RunWithCtx(t.trans.ctx, func() error {
		var err error
		hash, err = t.trans.getHash()

		return err
	})

	return hash, err
}

func (t *transferHandler) UpdateTransferInfo(info *r66.UpdateInfo) error {
	return utils.RunWithCtx(t.trans.ctx, func() error {
		return t.trans.updTransInfo(info)
	})
}

func (t *transferHandler) RunPreTask() (*r66.UpdateInfo, error) {
	var info *r66.UpdateInfo

	err := utils.RunWithCtx(t.trans.ctx, func() error {
		var err error
		info, err = t.trans.runPreTask()

		return err
	})

	return info, err
}

func (t *transferHandler) GetStream() (r66utils.ReadWriterAt, error) {
	var stream r66utils.ReadWriterAt

	err := utils.RunWithCtx(t.trans.ctx, func() error {
		var err error
		stream, err = t.trans.getStream(t.trans.ctx)

		return err
	})

	return stream, err
}

func (t *transferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	return utils.RunWithCtx(t.trans.ctx, func() error {
		return t.trans.validEndTransfer(end)
	})
}

func (t *transferHandler) RunPostTask() error {
	return utils.RunWithCtx(t.trans.ctx, func() error {
		return t.trans.runPostTask()
	})
}

func (t *transferHandler) ValidEndRequest() error {
	return utils.RunWithCtx(t.trans.ctx, func() error {
		return t.trans.validEndRequest()
	})
}

func (t *transferHandler) RunErrorTask(origErr error) error {
	return utils.RunWithCtx(t.trans.ctx, func() error {
		return t.trans.runErrorTasks(origErr)
	})
}

func (t *transferHandler) Interrupt(context.Context) error {
	sigShutdown := internal.NewR66Error(r66.Shutdown, "service is shutting down")
	t.cancel(sigShutdown)

	return nil
}

func (t *transferHandler) Pause(context.Context) error {
	sigPause := internal.NewR66Error(r66.StoppedTransfer, "transfer paused by user")
	t.cancel(sigPause)

	return nil
}

func (t *transferHandler) Cancel(context.Context) error {
	sigCancel := internal.NewR66Error(r66.CanceledTransfer, "transfer canceled by user")
	t.cancel(sigCancel)

	return nil
}
