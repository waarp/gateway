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

func (e *encrypt) makeAESCTREncryptor(cryptoKey *model.CryptoKey) error {
	return e.makeAESEncryptor(cryptoKey, cipher.NewCTR)
}

func (e *encrypt) makeAESCFBEncryptor(cryptoKey *model.CryptoKey) error {
	return e.makeAESEncryptor(cryptoKey, cipher.NewCFBEncrypter)
}

func (e *encrypt) makeAESOFBEncryptor(cryptoKey *model.CryptoKey) error {
	return e.makeAESEncryptor(cryptoKey, cipher.NewOFB)
}

func (e *encrypt) makeAESEncryptor(cryptoKey *model.CryptoKey,
	mkStream func(cipher.Block, []byte) cipher.Stream,
) error {
	if !isAESKey(cryptoKey) {
		return ErrEncryptNotAESKey
	}

	block, aesErr := aes.NewCipher([]byte(cryptoKey.Key))
	if aesErr != nil {
		return fmt.Errorf("failed to create AES cipher: %w", aesErr)
	}

	e.encrypt = func(src io.Reader, dst io.Writer) error {
		return encryptStream(src, dst, block, mkStream)
	}

	return nil
}
