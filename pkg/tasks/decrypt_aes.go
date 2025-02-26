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

func (d *decrypt) makeAESCTRDecryptor(cryptoKey *model.CryptoKey) error {
	return d.makeAESDecryptor(cryptoKey, cipher.NewCTR)
}

func (d *decrypt) makeAESCFBDecryptor(cryptoKey *model.CryptoKey) error {
	return d.makeAESDecryptor(cryptoKey, cipher.NewCFBDecrypter)
}

func (d *decrypt) makeAESOFBDecryptor(cryptoKey *model.CryptoKey) error {
	return d.makeAESDecryptor(cryptoKey, cipher.NewOFB)
}

func (d *decrypt) makeAESDecryptor(cryptoKey *model.CryptoKey,
	mkStream func(cipher.Block, []byte) cipher.Stream,
) error {
	if !isAESKey(cryptoKey) {
		return ErrDecryptNotAESKey
	}

	block, aesErr := aes.NewCipher([]byte(cryptoKey.Key))
	if aesErr != nil {
		return fmt.Errorf("failed to create AES cipher: %w", aesErr)
	}

	d.decrypt = func(src io.Reader, dst io.Writer) error {
		return decryptStream(src, dst, block, mkStream)
	}

	return nil
}
