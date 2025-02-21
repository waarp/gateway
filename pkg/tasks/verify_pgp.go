package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/lib/log"
	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	pgpprofile "github.com/ProtonMail/gopenpgp/v3/profile"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrVerifyPGPNoSigFile   = errors.New("missing PGP signature file")
	ErrVerifyPGPNoKeyName   = errors.New("missing PGP public key")
	ErrVerifyPGPKeyNotFound = errors.New("cryptographic key not found")
	ErrVerifyPGPNoPubKey    = errors.New("cryptographic key does not contain a public key")
)

type verifyPGP struct {
	PGPKeyName    string `json:"pgpKeyName"`
	SignatureFile string `json:"signatureFile"`

	verifyKey *pgp.Key
}

func (v *verifyPGP) checkParams(params map[string]string) error {
	if err := utils.JSONConvert(params, v); err != nil {
		return fmt.Errorf("failed to parse the PGP verification parameters: %w", err)
	}

	if v.SignatureFile == "" {
		return ErrVerifyPGPNoSigFile
	}

	if v.PGPKeyName == "" {
		return ErrVerifyPGPNoKeyName
	}

	return nil
}

func (v *verifyPGP) parseParams(db database.ReadAccess, params map[string]string) error {
	if err := v.checkParams(params); err != nil {
		return err
	}

	var pgpKey model.CryptoKey
	if err := db.Get(&pgpKey, "name = ?", v.PGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrVerifyPGPKeyNotFound, v.PGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if !isPGPPrivateKey(&pgpKey) && !isPGPPublicKey(&pgpKey) {
		return fmt.Errorf("%q: %w", pgpKey.Name, ErrVerifyPGPNoPubKey)
	}

	var err error
	if v.verifyKey, err = pgp.NewKeyFromArmored(pgpKey.Key.String()); err != nil {
		return fmt.Errorf("failed to parse PGP verification key: %w", err)
	}

	if v.verifyKey.IsPrivate() {
		if v.verifyKey, err = v.verifyKey.ToPublic(); err != nil {
			return fmt.Errorf("failed to parse PGP verification key: %w", err)
		}
	}

	return nil
}

func (v *verifyPGP) Validate(params map[string]string) error {
	return v.checkParams(params)
}

func (v *verifyPGP) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := v.parseParams(db, params); err != nil {
		logger.Error("Failed to parse PGP verification parameters: %v", err)

		return err
	}

	sigFile, sigErr := fs.Open(v.SignatureFile)
	if sigErr != nil {
		logger.Error("Failed to open PGP signature file: %v", sigErr)

		return fmt.Errorf("failed to open PGP signature file: %w", sigErr)
	}

	defer sigFile.Close() //nolint:errcheck //this error is inconsequential

	file, openErr := fs.Open(transCtx.Transfer.LocalPath)
	if openErr != nil {
		logger.Error("Failed to open file to sign: %v", openErr)

		return fmt.Errorf("failed to open file to sign: %w", openErr)
	}

	defer file.Close() //nolint:errcheck //this error is inconsequential

	if signErr := v.verify(logger, file, sigFile); signErr != nil {
		return signErr
	}

	return nil
}

func (v *verifyPGP) verify(logger *log.Logger, file, sigFile io.Reader) error {
	builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Verify()

	verifyHandler, handlerErr := builder.VerificationKey(v.verifyKey).New()
	if handlerErr != nil {
		logger.Error("Failed to create PGP verification handler: %v", handlerErr)

		return fmt.Errorf("failed to create PGP verification handler: %w", handlerErr)
	}

	verifier, verifyErr := verifyHandler.VerifyingReader(file, sigFile, pgp.Auto)
	if verifyErr != nil {
		logger.Error("Failed to initialize PGP verifier: %v", verifyErr)

		return fmt.Errorf("failed to initialize PGP verifier: %w", verifyErr)
	}

	if _, err := verifier.DiscardAllAndVerifySignature(); err != nil {
		logger.Error("Failed to verify file signature: %v", err)

		return fmt.Errorf("failed to sign file signature: %w", err)
	}

	return nil
}
