package sftp

import (
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

type sftpStream struct {
	*pipeline.TransferStream
	errors chan model.TransferError
}

func (s *sftpStream) TransferError(err error) {
	if err == io.EOF {
		s.errors <- model.NewTransferError(model.TeConnectionReset,
			"SFTP connection closed unexpectedly")
	}
}

func (s *sftpStream) Close() error {
	close(s.errors)
	err, ok := <-s.errors
	if !ok || err.Code == model.TeOk {
		err = s.Exit()
	}

	if err.Code != model.TeOk {
		s.ExitError(err)
		return err
	}
	return nil
}
