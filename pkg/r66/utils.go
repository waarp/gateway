// Package r66 adds implementations of both a r66 client & server, which can be
// used in conjunction with a TransferStream to make r66 transfers with the
// gateway.
package r66

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-r66/r66"
)

var errIncorrectHash = fmt.Errorf("file hash does not match expected value")

func makeHash(filepath string) ([]byte, error) {
	f, err := os.Open(utils.DenormalizePath(filepath))
	if err != nil {
		return nil, err.(*os.PathError).Err
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return nil, err
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
