package as2

import (
	"encoding/json"
	"errors"
	"fmt"

	"code.waarp.fr/lib/as2"
)

var (
	ErrUnknownSignAlgo    = errors.New("unknown signature algorithm")
	ErrUnknownEncryptAlgo = errors.New("unknown encryption algorithm")
)

type SignAlgo string

const (
	SignAlgoSHA1   = SignAlgo(as2.MICSHA1)
	SignAlgoMD5    = SignAlgo(as2.MICMD5)
	SignAlgoSHA256 = SignAlgo(as2.MICSHA256)
	SignAlgoSHA384 = SignAlgo(as2.MICSHA384)
	SignAlgoSHA512 = SignAlgo(as2.MICSHA512)
)

func SignatureAlgorithms() []string {
	return []string{
		string(SignAlgoSHA1),
		string(SignAlgoMD5),
		string(SignAlgoSHA256),
		string(SignAlgoSHA384),
		string(SignAlgoSHA512),
	}
}

func (a *SignAlgo) as2() as2.MICAlg { return as2.MICAlg(*a) }

//nolint:wrapcheck //no need to wrap here
func (a *SignAlgo) UnmarshalJSON(b []byte) error {
	var algo string
	if err := json.Unmarshal(b, &algo); err != nil {
		return err
	}

	switch SignAlgo(algo) {
	case "":
	case SignAlgoSHA1,
		SignAlgoMD5,
		SignAlgoSHA256,
		SignAlgoSHA384,
		SignAlgoSHA512:
	default:
		return fmt.Errorf("%w: %q", ErrUnknownSignAlgo, algo)
	}

	*a = SignAlgo(algo)

	return nil
}

type EncryptAlgo string

const (
	EncryptAlgoDESCBC    EncryptAlgo = "des-cbc"
	EncryptAlgoAES128CBC EncryptAlgo = "aes128-cbc"
	EncryptAlgoAES128GCM EncryptAlgo = "aes128-gcm"
	EncryptAlgoAES256CBC EncryptAlgo = "aes256-cbc"
	EncryptAlgoAES256GCM EncryptAlgo = "aes256-gcm"
)

func EncryptionAlgorithms() []string {
	return []string{
		string(EncryptAlgoDESCBC),
		string(EncryptAlgoAES128CBC),
		string(EncryptAlgoAES128GCM),
		string(EncryptAlgoAES256CBC),
		string(EncryptAlgoAES256GCM),
	}
}

func (a EncryptAlgo) PKCS7() as2.PKCS7EncryptionAlgorithm {
	switch a {
	case EncryptAlgoDESCBC:
		return as2.PKCS7EncDESCBC
	case EncryptAlgoAES128CBC:
		return as2.PKCS7EncAES128CBC
	case EncryptAlgoAES128GCM:
		return as2.PKCS7EncAES128GCM
	case EncryptAlgoAES256CBC:
		return as2.PKCS7EncAES256CBC
	case EncryptAlgoAES256GCM:
		return as2.PKCS7EncAES256GCM
	default:
		return -1
	}
}

//nolint:wrapcheck //no need to wrap here
func (a *EncryptAlgo) UnmarshalJSON(b []byte) error {
	var algo string
	if err := json.Unmarshal(b, &algo); err != nil {
		return err
	}

	switch EncryptAlgo(algo) {
	case "":
	case EncryptAlgoDESCBC,
		EncryptAlgoAES128CBC,
		EncryptAlgoAES128GCM,
		EncryptAlgoAES256CBC,
		EncryptAlgoAES256GCM:
		*a = EncryptAlgo(algo)
	default:
		return fmt.Errorf("%w: %q", ErrUnknownEncryptAlgo, algo)
	}

	return nil
}
