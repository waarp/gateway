package r66

import (
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
)

func (t *serverTransfer) checkSize() *pipeline.Error {
	if t.pip.TransCtx.Rule.IsSend || !t.conf.Filesize || t.pip.TransCtx.Transfer.Step > types.StepData {
		return nil
	}

	stat, err := t.pip.Stream.Stat()
	if err != nil {
		t.pip.Logger.Errorf("Failed to retrieve file info: %v", err)

		return pipeline.NewError(types.TeInternal, "failed to retrieve file info")
	}

	if stat.Size() != t.pip.TransCtx.Transfer.Filesize {
		msg := fmt.Sprintf("incorrect file size (expected %d, got %d)",
			t.pip.TransCtx.Transfer.Filesize, stat.Size())
		t.pip.Logger.Error(msg)

		return pipeline.NewError(types.TeBadSize, msg)
	}

	return nil
}

func (t *serverTransfer) getHash() ([]byte, error) {
	file := t.pip.Stream

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, internal.ToR66Error(err)
	}

	hash, tErr := internal.ComputeHash(t.ctx, t.conf.Digest, t.pip.Logger, file)
	if tErr != nil {
		return nil, internal.ToR66Error(tErr)
	}

	return hash, nil
}

func (t *serverTransfer) checkHash(exp []byte) error {
	if t.pip.TransCtx.Rule.IsSend || t.r66Conf.NoFinalHash || !t.conf.FinalHash ||
		(len(exp) == 0 && t.pip.TransCtx.Transfer.Filesize <= 0) {
		return nil
	}

	hasher, hashErr := internal.GetHasher(t.conf.Digest)
	if hashErr != nil {
		return internal.ToR66Error(hashErr)
	}

	if err := t.pip.Stream.CheckHash(hasher, exp); err != nil {
		return internal.ToR66Error(err)
	}

	return nil
}
