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

func (s *sign) makeHMACSHA256Signer(cryptoKey *model.CryptoKey) error {
	return s.makeHMACSigner(cryptoKey, sha256.New)
}

func (s *sign) makeHMACSHA384Signer(cryptoKey *model.CryptoKey) error {
	return s.makeHMACSigner(cryptoKey, sha512.New384)
}

func (s *sign) makeHMACSHA512Signer(cryptoKey *model.CryptoKey) error {
	return s.makeHMACSigner(cryptoKey, sha512.New)
}

func (s *sign) makeHMACMD5Signer(cryptoKey *model.CryptoKey) error {
	return s.makeHMACSigner(cryptoKey, md5.New)
}

func (s *sign) makeHMACSigner(cryptoKey *model.CryptoKey, mkHash func() hash.Hash) error {
	if !isHMACKey(cryptoKey) {
		return ErrSignNotHMACKey
	}

	hasher := hmac.New(mkHash, []byte(cryptoKey.Key))

	s.sign = func(file io.Reader) ([]byte, error) {
		if _, err := io.Copy(hasher, file); err != nil {
			return nil, fmt.Errorf("failed to compute file signature: %w", err)
		}

		return hasher.Sum(nil), nil
	}

	return nil
}
