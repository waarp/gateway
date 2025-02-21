package tasks

import (
	"bytes"
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
	ErrSignPGPNoOutFile    = errors.New("missing PGP signature output file")
	ErrSignPGPNoKeyName    = errors.New("missing PGP signature key")
	ErrSignPGPKeyNotFound  = errors.New("cryptographic key not found")
	ErrSignPGPNoPrivateKey = errors.New("cryptographic key does not contain a private PGP key")
)

type signPGP struct {
	PGPKeyName string `json:"pgpKeyName"`
	OutputFile string `json:"outputFile"`

	signKey *pgp.Key
}

func (s *signPGP) ValidateDB(db database.ReadAccess, params map[string]string) error {
	if err := utils.JSONConvert(params, s); err != nil {
		return fmt.Errorf("failed to parse the PGP signature parameters: %w", err)
	}

	if s.OutputFile == "" {
		return ErrSignPGPNoOutFile
	}

	if s.PGPKeyName == "" {
		return ErrSignPGPNoKeyName
	}

	var pgpKey model.CryptoKey
	if err := db.Get(&pgpKey, "name = ?", s.PGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrSignPGPKeyNotFound, s.PGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if !isPGPPrivateKey(&pgpKey) {
		return fmt.Errorf("%q: %w", pgpKey.Name, ErrSignPGPNoPrivateKey)
	}

	var err error
	if s.signKey, err = pgp.NewKeyFromArmored(pgpKey.Key.String()); err != nil {
		return fmt.Errorf("failed to parse PGP signature key: %w", err)
	}

	return nil
}

func (s *signPGP) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := s.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	file, openErr := fs.Open(transCtx.Transfer.LocalPath)
	if openErr != nil {
		logger.Error("Failed to open file to sign: %v", openErr)

		return fmt.Errorf("failed to open file to sign: %w", openErr)
	}

	defer file.Close() //nolint:errcheck //this error is inconsequential

	signature, signErr := s.sign(logger, file)
	if signErr != nil {
		return signErr
	}

	if err := fs.WriteFullFile(s.OutputFile, signature); err != nil {
		logger.Error("Failed to write PGP signature file: %v", err)

		return fmt.Errorf("failed to write PGP signature file: %w", err)
	}

	return nil
}

func (s *signPGP) sign(logger *log.Logger, file io.Reader) ([]byte, error) {
	builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Sign().Detached()

	signHandler, handlerErr := builder.SigningKey(s.signKey).New()
	if handlerErr != nil {
		logger.Error("Failed to create PGP signature handler: %v", handlerErr)

		return nil, fmt.Errorf("failed to create PGP signature handler: %w", handlerErr)
	}

	signature := &bytes.Buffer{}

	signer, signErr := signHandler.SigningWriter(signature, pgp.Armor)
	if signErr != nil {
		logger.Error("Failed to initialize PGP signer: %v", signErr)

		return nil, fmt.Errorf("failed to initialize PGP signer: %w", signErr)
	}

	if _, err := io.Copy(signer, file); err != nil {
		logger.Error("Failed to sign file: %v", err)

		return nil, fmt.Errorf("failed to sign file: %w", err)
	}

	if err := signer.Close(); err != nil {
		logger.Error("Failed to sign file: %v", err)

		return nil, fmt.Errorf("failed to sign file: %w", err)
	}

	return signature.Bytes(), nil
}
