package sftp

import (
	"errors"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"

	"golang.org/x/crypto/ssh"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

type errorHandler struct {
	ch ssh.Channel
}

func (e *errorHandler) SendError(*types.TransferError) {
	_ = e.ch.CloseWrite()
}

type stream struct {
	list     *SSHListener
	trans    *model.Transfer
	pipeline *pipeline.ServerPipeline
	file     pipeline.TransferStream
}

// newStream initialises a special kind of TransferStream tailored for
// the SFTP server. This constructor initialises a TransferStream, opens the
// local file and executes the pre-tasks.
func (l *SSHListener) newStream(pip *pipeline.ServerPipeline, trans *model.Transfer) (*stream, *types.TransferError) {
	l.runningTransfers.Add(trans.ID, pip)
	str := &stream{list: l, pipeline: pip, trans: trans}

	if err := pip.PreTasks(); err != nil {
		l.runningTransfers.Delete(trans.ID)
		return nil, err
	}

	file, err := pip.StartData()
	if err != nil {
		l.runningTransfers.Delete(trans.ID)
		return nil, err
	}
	str.file = file

	return str, nil
}

func (s *stream) TransferError(err error) {
	if err == io.EOF {
		s.pipeline.SetError(types.NewTransferError(types.TeUnknownRemote,
			"session closed by remote host"))
	} else {
		s.pipeline.SetError(types.NewTransferError(types.TeUnknownRemote, err.Error()))
	}
}

func (s *stream) ReadAt(p []byte, off int64) (int, error) {
	n, err := s.file.ReadAt(p, off)
	var tErr *types.TransferError
	if err != nil && errors.As(err, &tErr) {
		return n, modelToSFTP(tErr)
	}
	return n, err
}

func (s *stream) WriteAt(p []byte, off int64) (int, error) {
	n, err := s.file.WriteAt(p, off)
	var tErr *types.TransferError
	if err != nil && errors.As(err, &tErr) {
		return n, modelToSFTP(tErr)
	}
	return n, nil
}

func (s *stream) close() *types.TransferError {
	defer s.list.runningTransfers.Delete(s.trans.ID)
	if err := s.pipeline.EndData(); err != nil {
		return err
	}
	if err := s.pipeline.PostTasks(); err != nil {
		return err
	}
	if err := s.pipeline.EndTransfer(); err != nil {
		return err
	}
	return nil
}

func (s *stream) Close() error {
	if err := s.close(); err != nil {
		return modelToSFTP(err)
	}
	return nil
}
