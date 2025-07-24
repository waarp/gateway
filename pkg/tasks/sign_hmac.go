package tasks

import (
	"crypto/hmac"
	"crypto/md5" //nolint:gosec //MD5 is needed for compatibility
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var ErrSignNotHMACKey = errors.New("the provided cryptographic key does not contain an HMAC key")

const (
	SignMethodHMACSHA256 string = "HMAC-SHA256"
	SignMethodHMACSHA384 string = "HMAC-SHA384"
	SignMethodHMACSHA512 string = "HMAC-SHA512"
	SignMethodHMACMD5    string = "HMAC-MD5"
)

func isHMACKey(cryptoKey *model.CryptoKey) bool {
	return cryptoKey.Type == model.CryptoKeyTypeHMAC
}

func makeHMACSHA256Signer(cryptoKey *model.CryptoKey) (signFunc, error) {
	return makeHMACSigner(cryptoKey, sha256.New)
}

func makeHMACSHA384Signer(cryptoKey *model.CryptoKey) (signFunc, error) {
	return makeHMACSigner(cryptoKey, sha512.New384)
}

func makeHMACSHA512Signer(cryptoKey *model.CryptoKey) (signFunc, error) {
	return makeHMACSigner(cryptoKey, sha512.New)
}

func makeHMACMD5Signer(cryptoKey *model.CryptoKey) (signFunc, error) {
	return makeHMACSigner(cryptoKey, md5.New)
}

func makeHMACSigner(cryptoKey *model.CryptoKey, mkHash func() hash.Hash) (signFunc, error) {
	if !isHMACKey(cryptoKey) {
		return nil, ErrSignNotHMACKey
	}

	hasher := hmac.New(mkHash, []byte(cryptoKey.Key))

	return func(file io.Reader) ([]byte, error) {
		if _, err := io.Copy(hasher, file); err != nil {
			return nil, fmt.Errorf("failed to compute file signature: %w", err)
		}

		return hasher.Sum(nil), nil
	}, nil
}
