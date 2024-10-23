package wg

import (
	"fmt"
	"io"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
)

func displayPGPKeyInfo(w io.Writer, style *style, content string) error {
	pgpKey, err := pgp.NewKeyFromArmored(content)
	if err != nil {
		return fmt.Errorf("could not parse PGP key: %w", err)
	}

	kind := "Public"
	if pgpKey.IsPrivate() {
		kind = "Private"
	}

	subStyle := nextStyle(style)
	subSubStyle := nextStyle(subStyle)

	style.printf(w, "%s key:", kind)
	subStyle.printf(w, "Entity:")

	for _, identity := range pgpKey.GetEntity().Identities {
		subSubStyle.printf(w, identity.Name)
	}

	subStyle.printL(w, "Fingerprint", pgpKey.GetFingerprint())

	return nil
}
