package tasks

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var ErrDecryptNotAESKey = errors.New("the provided cryptographic key does not contain an AES key")

func makeAESCTRDecryptor(cryptoKey *model.CryptoKey) (decryptFunc, error) {
	return makeAESDecryptor(cryptoKey, cipher.NewCTR)
}

func makeAESCFBDecryptor(cryptoKey *model.CryptoKey) (decryptFunc, error) {
	return makeAESDecryptor(cryptoKey, cipher.NewCFBDecrypter)
}

func makeAESOFBDecryptor(cryptoKey *model.CryptoKey) (decryptFunc, error) {
	return makeAESDecryptor(cryptoKey, cipher.NewOFB)
}

func makeAESDecryptor(cryptoKey *model.CryptoKey,
	mkStream func(cipher.Block, []byte) cipher.Stream,
) (decryptFunc, error) {
	if !isAESKey(cryptoKey) {
		return nil, ErrDecryptNotAESKey
	}

	block, aesErr := aes.NewCipher([]byte(cryptoKey.Key))
	if aesErr != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", aesErr)
	}

	return func(src io.Reader, dst io.Writer) error {
		return decryptStream(src, dst, block, mkStream)
	}, nil
}
