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

var (
	ErrVerifyNotHMACKey           = errors.New("the provided cryptographic key does not contain an HMAC key")
	ErrVerifyHMACInvalidSignature = errors.New("the provided HMAC signature does not match the transfer file")
)

func (v *verify) makeHMACSHA256Verifier(cryptoKey *model.CryptoKey) error {
	return v.makeHMACVerifier(cryptoKey, sha256.New)
}

func (v *verify) makeHMACSHA384Verifier(cryptoKey *model.CryptoKey) error {
	return v.makeHMACVerifier(cryptoKey, sha512.New384)
}

func (v *verify) makeHMACSHA512Verifier(cryptoKey *model.CryptoKey) error {
	return v.makeHMACVerifier(cryptoKey, sha512.New)
}

func (v *verify) makeHMACMD5Verifier(cryptoKey *model.CryptoKey) error {
	return v.makeHMACVerifier(cryptoKey, md5.New)
}

func (v *verify) makeHMACVerifier(cryptoKey *model.CryptoKey, mkHash func() hash.Hash) error {
	if !isHMACKey(cryptoKey) {
		return ErrVerifyNotHMACKey
	}

	hasher := hmac.New(mkHash, []byte(cryptoKey.Key))

	v.verify = func(file io.Reader, expected []byte) error {
		if _, err := io.Copy(hasher, file); err != nil {
			return fmt.Errorf("failed to compute file signature: %w", err)
		}

		actual := hasher.Sum(nil)
		if !hmac.Equal(expected, actual) {
			return ErrVerifyHMACInvalidSignature
		}

		return nil
	}

	return nil
}
