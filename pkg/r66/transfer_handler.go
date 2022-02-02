package r66

import (
	"code.waarp.fr/lib/r66"
	r66utils "code.waarp.fr/lib/r66/utils"
)

type transferHandler struct {
	*sessionHandler
	trans *serverTransfer
}

func (t *transferHandler) GetHash() ([]byte, error) {
	return t.trans.getHash()
}

func (t *transferHandler) UpdateTransferInfo(info *r66.UpdateInfo) error {
	return t.trans.updTransInfo(info)
}

func (t *transferHandler) RunPreTask() (*r66.UpdateInfo, error) {
	return t.trans.runPreTask()
}

func (t *transferHandler) GetStream() (r66utils.ReadWriterAt, error) {
	return t.trans.getStream()
}

func (t *transferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	return t.trans.validEndTransfer(end)
}

func (t *transferHandler) RunPostTask() error {
	return t.trans.runPostTask()
}

func (t *transferHandler) ValidEndRequest() error {
	defer t.runningTransfers.Delete(t.trans.pip.TransCtx.Transfer.ID)

	return t.trans.validEndRequest()
}

func (t *transferHandler) RunErrorTask(err error) error {
	defer t.runningTransfers.Delete(t.trans.pip.TransCtx.Transfer.ID)

	return t.trans.runErrorTasks(err)
}
