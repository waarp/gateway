package r66

import (
	"context"
	"errors"
	"io"

	"code.waarp.fr/lib/r66"
	r66utils "code.waarp.fr/lib/r66/utils"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type serverStream struct {
	file *pipeline.FileStream
	ctx  context.Context
}

func (s *serverStream) ReadAt(p []byte, off int64) (int, error) {
	if err := utils.CheckCtx(s.ctx); err != nil {
		return 0, err
	}

	n, rErr := s.file.ReadAt(p, off)
	if rErr != nil && !errors.Is(rErr, io.EOF) {
		return n, internal.ToR66Error(rErr)
	}

	if err := utils.CheckCtx(s.ctx); err != nil {
		return 0, err
	}

	return n, rErr //nolint:wrapcheck //error is either nil or io.EOF, do not wrap
}

func (s *serverStream) WriteAt(p []byte, off int64) (int, error) {
	if err := utils.CheckCtx(s.ctx); err != nil {
		return 0, err
	}

	n, err := s.file.WriteAt(p, off)
	if err != nil {
		return n, internal.ToR66Error(err)
	}

	return n, utils.CheckCtx(s.ctx)
}

type serverTransfer struct {
	r66Conf *serverConfig
	conf    *r66.Authent
	pip     *pipeline.Pipeline
	ctx     context.Context
}

func (t *serverTransfer) getHash() ([]byte, error) {
	hash, tErr := internal.MakeHash(t.ctx, t.pip.TransCtx.FS, t.pip.Logger,
		&t.pip.TransCtx.Transfer.LocalPath)
	if tErr != nil {
		return nil, internal.ToR66Error(tErr)
	}

	return hash, nil
}

func (t *serverTransfer) updTransInfo(info *r66.UpdateInfo) error {
	if !t.pip.TransCtx.Rule.IsSend {
		if err := internal.UpdateFileInfo(info, t.pip); err != nil {
			return internal.ToR66Error(err)
		}
	}

	if fID := info.FileInfo.SystemData.FollowID; fID != 0 {
		t.pip.TransCtx.TransInfo[internal.FollowID] = fID
	}

	if tErr := internal.UpdateTransferInfo(info.FileInfo.UserContent, t.pip); tErr != nil {
		return internal.ToR66Error(tErr)
	}

	return nil
}

func (t *serverTransfer) runPreTask() (*r66.UpdateInfo, error) {
	if pErr := t.pip.PreTasks(); pErr != nil {
		return nil, internal.ToR66Error(pErr)
	}

	if t.pip.TransCtx.Rule.IsSend {
		return &r66.UpdateInfo{
			Filename: t.pip.TransCtx.Transfer.SrcFilename,
			FileSize: t.pip.TransCtx.Transfer.Filesize,
			FileInfo: &r66.TransferData{},
		}, nil
	}

	return nil, nil //nolint:nilnil //library requires us to return nil in this case
}

func (t *serverTransfer) getStream(ctx context.Context) (r66utils.ReadWriterAt, error) {
	file, fErr := t.pip.StartData() //nolint:contextcheck //no need to pass context here
	if fErr != nil {
		return nil, internal.ToR66Error(fErr)
	}

	return &serverStream{file: file, ctx: ctx}, nil
}

func (t *serverTransfer) validEndTransfer(end *r66.EndTransfer) error {
	if t.pip.Stream == nil {
		_, dErr := t.pip.StartData()
		if dErr != nil {
			return internal.ToR66Error(dErr)
		}
	}

	if pErr := t.pip.EndData(); pErr != nil {
		return internal.ToR66Error(pErr)
	}

	if sErr := t.checkSize(); sErr != nil {
		return internal.ToR66Error(sErr)
	}

	if hErr := t.checkHash(end.Hash); hErr != nil {
		return internal.ToR66Error(hErr)
	}

	return nil
}

func (t *serverTransfer) runPostTask() error {
	if pErr := t.pip.PostTasks(); pErr != nil {
		return internal.ToR66Error(pErr)
	}

	return nil
}

func (t *serverTransfer) validEndRequest() error {
	if tErr := t.pip.EndTransfer(); tErr != nil {
		return internal.ToR66Error(tErr)
	}

	return nil
}

func (t *serverTransfer) runErrorTasks(err error) error {
	if tErr := internal.FromR66Error(err, t.pip); tErr != nil {
		t.pip.SetError(tErr.Code(), tErr.Details())
	}

	return nil
}
