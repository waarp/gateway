package sftp

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

type sftpStream struct {
	*pipeline.TransferStream

	transErr model.TransferError
}

// newSftpStream initialises a special kind of TransferStream tailored for
// the SFTP server. This constructor initialises a TransferStream, opens the
// local file and executes the pre-tasks.
func newSftpStream(logger *log.Logger, db *database.Db, root string,
	trans model.Transfer) (*sftpStream, error) {

	s, err := pipeline.NewTransferStream(logger, db, root, trans)
	if err != nil {
		return nil, err
	}
	stream := &sftpStream{TransferStream: s}

	if te := s.Start(); te.Code != model.TeOk {
		s.ErrorTasks(te)
		s.Exit()
		return nil, te
	}

	if pe := s.PreTasks(); pe.Code != model.TeOk {
		s.ErrorTasks(pe)
		s.Exit()
		return nil, pe
	}

	return stream, nil
}

func (s *sftpStream) TransferError(err error) {
	if te, ok := err.(model.TransferError); ok {
		s.transErr = te
		return
	}
	s.transErr = model.NewTransferError(model.TeConnectionReset,
		"SFTP connection closed unexpectedly")
}

func (s *sftpStream) Close() error {
	defer s.Exit()

	if err := s.TransferStream.Close(); err != nil {
		s.Logger.Warningf("Failed to close local file: %s", err.Error())
	}

	if s.transErr.Code == model.TeOk {
		s.transErr = s.PostTasks()
		if s.transErr.Code == model.TeOk {
			return nil
		}
	}

	s.ErrorTasks(s.transErr)
	return s.transErr
}
