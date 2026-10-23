package keys

import (
	"cmp"
	"encoding/base64"
	"slices"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The states of a key, as the kit draws them.
const (
	stateOK   = "ok"
	stateWarn = "warn"
	stateKO   = "ko"

	// A PGP key that expires within this delay is marked.
	expiryWarning = 30 * 24 * time.Hour

	// The AES key sizes the tasks can use, in bytes.
	aes128      = 16
	aes192      = 24
	aes256      = 32
	bitsPerByte = 8

	// The causes of an unusable key, completing "keys.cause.".
	causeAES        = "aes"
	causeHMAC       = "hmac"
	causePGP        = "pgp"
	causeNotPrivate = "not_private"
	causeNotPublic  = "not_public"
	causeLocked     = "locked"
	causeEmpty      = "empty"
	causeType       = "type"

	fingerprintGroup = 4
)

// The public key algorithms of OpenPGP (RFC 9580, section 9.1), by their
// identifier, for the summary of a key.
//
//nolint:gochecknoglobals,mnd // reference data of OpenPGP
var pgpAlgorithms = map[uint8]string{
	1: "RSA", 2: "RSA", 3: "RSA", 16: "ElGamal", 17: "DSA", 18: "ECDH", 19: "ECDSA", 22: "EdDSA",
	25: "X25519", 26: "X448", 27: "Ed25519", 28: "Ed448",
}

// keyInfo is what the value of a key says, read as the tasks read it: the
// size of an AES key, the summary of a PGP key; Readable is false, with the
// cause, when the tasks cannot use it.
type keyInfo struct {
	Readable bool
	Cause    string

	Bits int // AES

	Fingerprint string
	KeyID       string
	Identities  []string
	Algorithm   string
	Created     time.Time
	Expires     time.Time // zero: never
	Expired     bool
	Revoked     bool
	Public      string // the armoured public key, for a public key only
}

// infoOf reads the value of a key for its type.
func infoOf(k *model.CryptoKey, now time.Time) keyInfo {
	switch k.Type {
	case model.CryptoKeyTypeAES:
		return aesInfo(k.Key)
	case model.CryptoKeyTypeHMAC:
		if k.Key == "" {
			return keyInfo{Cause: causeEmpty}
		}

		return hmacInfo(k.Key)
	case model.CryptoKeyTypePGPPublic, model.CryptoKeyTypePGPPrivate:
		return pgpInfo(k.Type, k.Key, now)
	default:
		return keyInfo{Cause: causeType}
	}
}

// aesInfo decodes an AES key from base64, as the tasks do: 16, 24 or 32 bytes.
func aesInfo(value string) keyInfo {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || !slices.Contains([]int{aes128, aes192, aes256}, len(decoded)) {
		return keyInfo{Cause: causeAES}
	}

	return keyInfo{Readable: true, Bits: len(decoded) * bitsPerByte}
}

// hmacInfo decodes an HMAC secret from base64, as the tasks do (#736).
func hmacInfo(value string) keyInfo {
	if _, err := base64.StdEncoding.DecodeString(value); err != nil {
		return keyInfo{Cause: causeHMAC}
	}

	return keyInfo{Readable: true}
}

// pgpInfo reads an armoured PGP key: its identities, fingerprint, algorithm
// and dates; a private key protected by a passphrase cannot be used by the
// tasks, the product having no field for the passphrase.
func pgpInfo(typ, armored string, now time.Time) keyInfo {
	key, err := pgp.NewKeyFromArmored(armored)
	if err != nil {
		return keyInfo{Cause: causePGP}
	}

	info := keyInfo{
		Readable:    true,
		Fingerprint: groupHex(key.GetFingerprint()),
		KeyID:       strings.ToUpper(key.GetHexKeyID()),
		Expired:     key.IsExpired(now.Unix()),
		Revoked:     key.IsRevoked(now.Unix()),
	}

	if entity := key.GetEntity(); entity != nil {
		for name := range entity.Identities {
			info.Identities = append(info.Identities, name)
		}

		slices.Sort(info.Identities)

		if pk := entity.PrimaryKey; pk != nil {
			info.Created = pk.CreationTime
			info.Algorithm = pgpAlgorithms[uint8(pk.PubKeyAlgo)]

			if bits, bitsErr := pk.BitLength(); bitsErr == nil && info.Algorithm == "RSA" {
				info.Algorithm += " " + itoa(int(bits))
			}

			if sig, sigErr := entity.PrimarySelfSignature(time.Time{}, nil); sigErr == nil && sig != nil &&
				sig.KeyLifetimeSecs != nil && *sig.KeyLifetimeSecs > 0 {
				info.Expires = pk.CreationTime.Add(time.Duration(*sig.KeyLifetimeSecs) * time.Second)
			}
		}
	}

	switch private := key.IsPrivate(); {
	case typ == model.CryptoKeyTypePGPPrivate && !private:
		info.Readable, info.Cause = false, causeNotPrivate
	case typ == model.CryptoKeyTypePGPPublic && private:
		info.Readable, info.Cause = false, causeNotPublic
	case private:
		if locked, lockErr := key.IsLocked(); lockErr != nil || locked {
			info.Readable, info.Cause = false, causeLocked
		}
	default:
		info.Public = strings.TrimSpace(armored)
	}

	return info
}

// groupHex writes a fingerprint in upper case, by groups of four.
func groupHex(hex string) string {
	hex = strings.ToUpper(hex)

	var b strings.Builder

	for i := 0; i < len(hex); i += fingerprintGroup {
		if i > 0 {
			b.WriteByte(' ')
		}

		b.WriteString(hex[i:min(i+fingerprintGroup, len(hex))])
	}

	return b.String()
}

// state is what the list and the sheet say of a key: usable, expiring
// within 30 days, expired or revoked, unusable and why.
func (i *keyInfo) state(translate gui.Translator, now time.Time) gui.Status {
	date := translate("auth.date")

	switch {
	case !i.Readable:
		return gui.Status{State: stateKO, Label: translate("keys.state.unusable", "cause", translate(causeKeyPrefix+i.Cause))}
	case i.Revoked:
		return gui.Status{State: stateKO, Label: translate("keys.state.revoked")}
	case i.Expired:
		return gui.Status{State: stateKO, Label: translate("keys.state.expired", "date", i.Expires.Format(date))}
	case !i.Expires.IsZero() && i.Expires.Sub(now) < expiryWarning:
		return gui.Status{State: stateWarn, Label: translate("keys.state.warn", "date", i.Expires.Format(date))}
	default:
		return gui.Status{State: stateOK, Label: translate("keys.state.ok")}
	}
}

// use is a rule whose tasks name a key: what the screen shows of it, the
// tasks themselves being always the same ones (encrypt, decrypt, sign,
// verify).
type use struct {
	RuleID int64
	Rule   string
	Send   bool
}

// keyParams are, for each task of the product that reads a cryptographic
// key, the arguments that name it.
//
//nolint:gochecknoglobals // the arguments the tasks of the product read
var keyParams = map[string][]string{
	tasks.Encrypt:          {"keyName"},
	tasks.Decrypt:          {"keyName"},
	tasks.Sign:             {"keyName"},
	tasks.Verify:           {"keyName"},
	tasks.EncryptAndSign:   {"encryptKeyName", "signKeyName"},
	tasks.DecryptAndVerify: {"decryptKeyName", "verifyKeyName"},
}

// usesOf finds, for every key name, the rules whose tasks name it, by rule
// name then direction.
func usesOf(list model.Tasks, rules model.Rules) map[string][]use {
	byID := make(map[int64]*model.Rule, len(rules))
	for _, r := range rules {
		byID[r.ID] = r
	}

	uses := map[string][]use{}

	for _, t := range list {
		r, ok := byID[t.RuleID]
		if !ok {
			continue
		}

		for _, param := range keyParams[t.Type] {
			name := t.Args[param]
			u := use{RuleID: r.ID, Rule: r.Name, Send: r.IsSend}

			if name != "" && !slices.Contains(uses[name], u) {
				uses[name] = append(uses[name], u)
			}
		}
	}

	for name := range uses {
		slices.SortFunc(uses[name], func(a, b use) int {
			return cmp.Or(strings.Compare(a.Rule, b.Rule), directionOrder(a.Send)-directionOrder(b.Send))
		})
	}

	return uses
}

// directionOrder puts the sending rule before the receiving one of the same
// name.
func directionOrder(send bool) int {
	if send {
		return 0
	}

	return 1
}
