package r66

import (
	"context"
	"errors"
	"io"
	"strings"

	"code.waarp.fr/lib/r66"
	r66utils "code.waarp.fr/lib/r66/utils"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

var errInvalidHandle = errors.New("file handle is no longer valid")

func checkBefore(store *utils.ErrorStorage) error {
	select {
	case iErr, ok := <-store.Wait():
		if ok {
			return internal.ToR66Error(iErr)
		}

		return errInvalidHandle
	default:
		return nil
	}
}

func checkAfter(store *utils.ErrorStorage, tErr *types.TransferError) error {
	select {
	case <-store.Wait():
		return internal.ToR66Error(store.Get())
	default:
		if tErr != nil {
			err := internal.ToR66Error(tErr)
			store.Store(err)

			return err
		}

		return nil
	}
}

type serverStream struct {
	file  pipeline.TransferStream
	store *utils.ErrorStorage
}

func (s *serverStream) ReadAt(p []byte, off int64) (int, error) {
	if err := checkBefore(s.store); err != nil {
		return 0, err
	}

	n, err := s.file.ReadAt(p, off)

	if err == nil {
		return n, nil
	}

	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}

	var tErr *types.TransferError
	if !errors.As(err, &tErr) {
		tErr = types.NewTransferError(types.TeInternal, err.Error())
	}

	return n, checkAfter(s.store, tErr)
}

func (s *serverStream) WriteAt(p []byte, off int64) (int, error) {
	if err := checkBefore(s.store); err != nil {
		return 0, err
	}

	n, err := s.file.WriteAt(p, off)

	if err == nil {
		return n, nil
	}

	var tErr *types.TransferError
	if !errors.As(err, &tErr) {
		tErr = types.NewTransferError(types.TeInternal, err.Error())
	}

	return n, checkAfter(s.store, tErr)
}

type serverTransfer struct {
	conf  *r66.Authent
	pip   *pipeline.Pipeline
	store *utils.ErrorStorage
}

func (t *serverTransfer) Interrupt(ctx context.Context) error {
	sigShutdown := internal.NewR66Error(r66.Shutdown, "service is shutting down")

	t.pip.Interrupt(func() {
		t.store.StoreCtx(ctx, sigShutdown)
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

func (t *serverTransfer) Pause(ctx context.Context) error {
	sigPause := internal.NewR66Error(r66.StoppedTransfer, "transfer paused by user")

	defer t.pip.Pause(func() {
		t.store.StoreCtx(ctx, sigPause)
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

func (t *serverTransfer) Cancel(ctx context.Context) error {
	sigCancel := internal.NewR66Error(r66.CanceledTransfer, "transfer canceled by user")

	defer t.pip.Cancel(func() {
		t.store.StoreCtx(ctx, sigCancel)
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

func (t *serverTransfer) getHash() ([]byte, error) {
	if err := checkBefore(t.store); err != nil {
		return nil, err
	}

	hash, err := t.makeHash()

	return hash, checkAfter(t.store, err)
}

func (t *serverTransfer) updTransInfo(info *r66.UpdateInfo) error {
	if err := checkBefore(t.store); err != nil {
		return err
	}

	err := internal.UpdateInfo(info, t.pip)

	return checkAfter(t.store, err)
}

func (t *serverTransfer) runPreTask() (*r66.UpdateInfo, error) {
	if err := checkBefore(t.store); err != nil {
		return nil, err
	}

	var info *r66.UpdateInfo

	pErr := t.pip.PreTasks()

	if t.pip.TransCtx.Rule.IsSend {
		info = &r66.UpdateInfo{
			Filename: strings.TrimPrefix(t.pip.TransCtx.Transfer.RemotePath, "/"),
			FileSize: t.pip.TransCtx.Transfer.Filesize,
			FileInfo: &r66.TransferData{},
		}
	}

	return info, checkAfter(t.store, pErr)
}

func (t *serverTransfer) getStream() (r66utils.ReadWriterAt, error) {
	if err := checkBefore(t.store); err != nil {
		return nil, err
	}

	file, fErr := t.pip.StartData()
	if err := checkAfter(t.store, fErr); err != nil {
		return nil, err
	}

	return &serverStream{file: file, store: t.store}, nil
}

func (t *serverTransfer) validEndTransfer(end *r66.EndTransfer) error {
	if err := checkBefore(t.store); err != nil {
		return err
	}

	if t.pip.Stream == nil {
		_, dErr := t.pip.StartData()
		if dErr != nil {
			return checkAfter(t.store, dErr)
		}
	}

	if pErr := t.pip.EndData(); pErr != nil {
		return checkAfter(t.store, pErr)
	}

	if sErr := t.checkSize(); sErr != nil {
		return checkAfter(t.store, sErr)
	}

	if len(end.Hash) > 0 || t.pip.TransCtx.Transfer.Filesize > 0 {
		if hErr := t.checkHash(end.Hash); hErr != nil {
			return checkAfter(t.store, hErr)
		}
	}
	return nil
}

func (t *serverTransfer) runPostTask() error {
	if err := checkBefore(t.store); err != nil {
		return err
	}

	pErr := t.pip.PostTasks()

	return checkAfter(t.store, pErr)
}

func (t *serverTransfer) validEndRequest() error {
	defer t.store.Close()

	if tErr := t.pip.EndTransfer(); tErr != nil {
		return internal.ToR66Error(tErr)
	}

	return nil
}

func (t *serverTransfer) runErrorTasks(err error) error {
	tErr := internal.FromR66Error(err, t.pip)
	if tErr != nil {
		t.pip.SetError(tErr)
	}

	return nil
}
