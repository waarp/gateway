package r66

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-r66/r66"
)

var (
	errDatabase    = types.NewTransferError(types.TeInternal, "database error")
	r66ErrDatabase = internal.NewR66Error(r66.Internal, "database error")
)

func (t *serverTransfer) checkSize() *types.TransferError {
	if t.pip.TransCtx.Rule.IsSend || !t.conf.Filesize || t.pip.TransCtx.Transfer.Step > types.StepData {
		return nil
	}

	stat, err := os.Stat(t.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		t.pip.Logger.Errorf("Failed to retrieve file info: %s", err)
		return types.NewTransferError(types.TeInternal, "failed to retrieve file info")
	}
	if stat.Size() != t.pip.TransCtx.Transfer.Filesize {
		msg := fmt.Sprintf("incorrect file size (expected %d, got %d)",
			t.pip.TransCtx.Transfer.Filesize, stat.Size())
		t.pip.Logger.Error(msg)
		return types.NewTransferError(types.TeBadSize, msg)
	}
	return nil
}

func (t *serverTransfer) checkHash(exp []byte) *types.TransferError {
	hash, err := t.makeHash()
	if err != nil {
		return err
	}

	if !bytes.Equal(hash, exp) {
		t.pip.Logger.Errorf("File hash verification failed: hashes do not match")
		return types.NewTransferError(types.TeIntegrity, "file hash does not match expected value")
	}
	return nil
}

func (t *serverTransfer) makeHash() ([]byte, *types.TransferError) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var hash []byte
	var tErr *types.TransferError
	done := make(chan struct{})
	go func() {
		defer close(done)
		hash, tErr = internal.MakeHash(ctx, t.pip.Logger, t.pip.TransCtx.Transfer.LocalPath)
	}()
	select {
	case <-t.store.Wait():
		cancel()
		<-done
	case <-done:
	}
	if tErr != nil {
		return nil, tErr
	}
	return hash, nil
}
