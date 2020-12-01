package r66

import (
	"code.waarp.fr/waarp-r66/r66"
)

type testAuthHandler struct {
	r66.AuthentHandler
}

func (t *testAuthHandler) ValidAuth(authent *r66.Authent) (r66.SessionHandler, error) {
	s, err := t.AuthentHandler.ValidAuth(authent)
	if err != nil {
		serverCheckChannel <- err.Error()
	}
	return &testSessionHandler{s}, err
}

type testSessionHandler struct {
	r66.SessionHandler
}

func (t *testSessionHandler) ValidRequest(request *r66.Request) (r66.TransferHandler, error) {
	h, err := t.SessionHandler.ValidRequest(request)
	if err != nil {
		serverCheckChannel <- err.Error()
	}
	return &testTransferHandler{h}, err
}

type testTransferHandler struct {
	r66.TransferHandler
}

func (t *testTransferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	err := t.TransferHandler.ValidEndTransfer(end)
	if err != nil {
		serverCheckChannel <- err.Error()
	}
	return err
}

func (t *testTransferHandler) ValidEndRequest() error {
	err := t.TransferHandler.ValidEndRequest()
	serverCheckChannel <- "SERVER END TRANSFER OK"
	return err
}

func (t *testTransferHandler) RunErrorTask(protoErr error) error {
	err := t.TransferHandler.RunErrorTask(protoErr)
	serverCheckChannel <- "SERVER END TRANSFER ERROR"
	return err
}
