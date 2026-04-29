package tasks

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	EncryptMethodAESCFB = "AES-CFB"
	EncryptMethodAESCTR = "AES-CTR"
	EncryptMethodAESOFB = "AES-OFB"
)

var ErrEncryptNotAESKey = errors.New("the provided cryptographic key does not contain an AES key")

func isAESKey(key *model.CryptoKey) bool {
	return key.Type == model.CryptoKeyTypeAES
}

func makeAESCTREncryptor(cryptoKey *model.CryptoKey) (encryptFunc, error) {
	return makeAESEncryptor(cryptoKey, cipher.NewCTR)
}

func makeAESCFBEncryptor(cryptoKey *model.CryptoKey) (encryptFunc, error) {
	//nolint:staticcheck // CFB is needed here
	return makeAESEncryptor(cryptoKey, cipher.NewCFBEncrypter)
}

func makeAESOFBEncryptor(cryptoKey *model.CryptoKey) (encryptFunc, error) {
	//nolint:staticcheck // OFB is needed here
	return makeAESEncryptor(cryptoKey, cipher.NewOFB)
}

func makeAESEncryptor(cryptoKey *model.CryptoKey,
	mkStream func(cipher.Block, []byte) cipher.Stream,
) (encryptFunc, error) {
	if !isAESKey(cryptoKey) {
		return nil, ErrEncryptNotAESKey
	}

	key, err := base64.StdEncoding.DecodeString(string(cryptoKey.Key))
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	block, aesErr := aes.NewCipher(key)
	if aesErr != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", aesErr)
	}

	return func(src io.Reader, dst io.Writer) error {
		return encryptStream(src, dst, block, mkStream)
	}, nil
}
