package r66

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
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
	if !t.conf.FinalHash || (len(exp) == 0 && t.pip.TransCtx.Transfer.Filesize <= 0) {
		return nil
	}

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

	var (
		hash []byte
		tErr *types.TransferError
	)

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
