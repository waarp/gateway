package r66

import (
	"bytes"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
)

func (t *serverTransfer) checkSize() *types.TransferError {
	if t.pip.TransCtx.Rule.IsSend || !t.conf.Filesize || t.pip.TransCtx.Transfer.Step > types.StepData {
		return nil
	}

	stat, err := fs.Stat(t.pip.TransCtx.FS, &t.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		t.pip.Logger.Error("Failed to retrieve file info: %s", err)

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

func (t *serverTransfer) checkHash(exp []byte) error {
	if t.r66Conf.NoFinalHash || !t.conf.FinalHash || (len(exp) == 0 &&
		t.pip.TransCtx.Transfer.Filesize <= 0) {
		return nil
	}

	hash, err := internal.MakeHash(t.ctx, t.pip.TransCtx.FS, t.pip.Logger,
		&t.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		return err
	}

	if !bytes.Equal(hash, exp) {
		t.pip.Logger.Error("File hash verification failed: hashes do not match")

		return types.NewTransferError(types.TeIntegrity, "file hash does not match expected value")
	}

	return nil
}
