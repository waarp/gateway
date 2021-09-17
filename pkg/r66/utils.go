// Package r66 adds implementations of both a r66 client & server, which can be
// used in conjunction with a TransferStream to make r66 transfers with the
// gateway.
package r66

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"code.waarp.fr/waarp-r66/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

var errIncorrectHash = fmt.Errorf("file hash does not match expected value")

func makeHash(filepath string) ([]byte, error) {
	f, err := os.Open(utils.DenormalizePath(filepath))
	if err != nil {
		var err2 *os.PathError

		errors.As(err, &err2)

		return nil, err2.Err
	}

	//nolint:errcheck // no logger to handle the error
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return nil, fmt.Errorf("cannot generate hash: %w", err)
	}

	hash := hasher.Sum(nil)

	return hash, nil
}

func checkHash(filepath string, expHash []byte) error {
	hash, err := makeHash(filepath)
	if err != nil {
		return err
	}

	if !bytes.Equal(hash, expHash) {
		return errIncorrectHash
	}

	return nil
}

func setProgress(trans *model.Transfer, request *r66.Request) {
	curBlock := uint32(trans.Progress / uint64(request.Block))
	if request.Rank < curBlock {
		curBlock = request.Rank
	}

	request.Rank = curBlock

	if trans.Step == types.StepData {
		trans.Progress = uint64(curBlock) * uint64(request.Block)
	}
}
