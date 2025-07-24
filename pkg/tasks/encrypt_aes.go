package tasks

import (
	"crypto/aes"
	"crypto/cipher"
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
	return makeAESEncryptor(cryptoKey, cipher.NewCFBEncrypter)
}

func makeAESOFBEncryptor(cryptoKey *model.CryptoKey) (encryptFunc, error) {
	return makeAESEncryptor(cryptoKey, cipher.NewOFB)
}

func makeAESEncryptor(cryptoKey *model.CryptoKey,
	mkStream func(cipher.Block, []byte) cipher.Stream,
) (encryptFunc, error) {
	if !isAESKey(cryptoKey) {
		return nil, ErrEncryptNotAESKey
	}

	block, aesErr := aes.NewCipher([]byte(cryptoKey.Key))
	if aesErr != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", aesErr)
	}

	return func(src io.Reader, dst io.Writer) error {
		return encryptStream(src, dst, block, mkStream)
	}, nil
}
