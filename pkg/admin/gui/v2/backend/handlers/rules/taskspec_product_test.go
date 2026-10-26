package rules

import (
	"maps"
	"testing"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/model/compatibility" // the default client of TRANSFER, as in the gateway
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// What the samples of the arguments need besides their kind: a value the
// product accepts for one type (an ICAP address, an archive extension, a key
// of the type the first method needs), and, for TRANSFER, the partner the
// product requires on one side or the other, which the form cannot require.
//
//nolint:gochecknoglobals // reference data of the test
var (
	sampleOverrides = map[string]map[string]string{
		tasks.Icap:             {"uploadURL": "icap://127.0.0.1:1344/avscan"},
		tasks.Archive:          {"outputPath": "/tmp/archive.zip"},
		tasks.Email:            {"recipients": "exploitation@example.org"},
		tasks.Transcode:        {"toCharset": "UTF-16BE"},
		tasks.Sign:             {"keyName": "hmac"},
		tasks.Verify:           {"keyName": "hmac"},
		tasks.EncryptAndSign:   {"encryptKeyName": "pgp-public", "signKeyName": "pgp-private"},
		tasks.DecryptAndVerify: {"decryptKeyName": "pgp-private", "verifyKeyName": "pgp-public"},
	}
	sampleExtras = map[string]map[string]string{
		tasks.Transfer: {"to": "banque-demo"},
	}

	// The arguments the form requires and the product does not, each with
	// its reason.
	formStricter = map[string]string{
		tasks.Preregister + ".isSend":   "the form always says the direction; the product reads none as receiving",
		tasks.Verify + ".signatureFile": "the product checks it only when the task runs, which then fails",
	}
	sampleRefs = map[refKind]string{
		refKey: "aes", refSMTP: sender, refTemplate: template, refPartner: "banque-demo", refRemoteAccount: "prod",
		refClient: "cli-sftp", refServer: "srv-sftp", refLocalAccount: "filiale-sud", refRule: "releves",
	}
	sampleKinds = map[argKind]string{
		argText: "/tmp/fichier", argBool: "true", argInt: "1", argDecimal: "1.5",
		argDuration: "1s", argOctal: "640", argJSON: "{}",
	}
)

// sampleArgs are the required arguments of a type, each with a value the
// product accepts.
func sampleArgs(spec *taskSpec) map[string]string {
	args := map[string]string{}

	for i := range spec.Args {
		arg := &spec.Args[i]
		if !arg.Required || arg.When != nil {
			continue
		}

		switch arg.Kind {
		case argChoice:
			args[arg.Name] = arg.Choices()[0].Value
		case argRef:
			args[arg.Name] = sampleRefs[arg.Ref]
		default:
			args[arg.Name] = sampleKinds[arg.Kind]
		}

		if value, ok := sampleOverrides[spec.Type][arg.Name]; ok {
			args[arg.Name] = value
		}
	}

	return args
}

// validate checks the arguments of a task as the product does when it saves
// one.
func validate(db database.ReadAccess, typ string, args map[string]string) error {
	switch validator := model.ValidTasks[typ]().(type) {
	case model.TaskValidator:
		return validator.Validate(args) //nolint:wrapcheck // the error of the product, as it is
	case model.TaskValidatorDB:
		return validator.ValidateDB(db, args) //nolint:wrapcheck // the error of the product, as it is
	default:
		return nil
	}
}

// keyFixtures inserts the keys the samples name: AES, HMAC, a PGP pair.
func (s *suite) keyFixtures(t *testing.T) {
	t.Helper()

	key, err := pgp.PGP().KeyGeneration().AddUserId("Partenaire", "partenaire@example.org").New().GenerateKey()
	require.NoError(t, err)

	private, err := key.Armor()
	require.NoError(t, err)

	public, err := key.GetArmoredPublicKey()
	require.NoError(t, err)

	secret := "0123456789abcdef0123456789abcdef" // the model takes the key as it is: 32 bytes, AES-256

	for _, k := range []*model.CryptoKey{
		{Name: "aes", Type: model.CryptoKeyTypeAES, Key: secret},
		{Name: "hmac", Type: model.CryptoKeyTypeHMAC, Key: secret},
		{Name: "pgp-private", Type: model.CryptoKeyTypePGPPrivate, Key: private},
		{Name: "pgp-public", Type: model.CryptoKeyTypePGPPublic, Key: public},
	} {
		require.NoError(t, s.db.Insert(k).Run())
	}
}

// The form requires what the product requires: for every type, the required
// arguments of the description, with values the product accepts, make a task
// it saves (so it requires nothing the form leaves optional), and the
// product refuses the task without any one of them, but for the arguments
// the form requires on purpose (formStricter).
func TestTaskSpecsRequired(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)
	s.keyFixtures(t)

	for i := range taskSpecs {
		spec := &taskSpecs[i]
		full := sampleArgs(spec)

		withExtras := maps.Clone(full)
		maps.Copy(withExtras, sampleExtras[spec.Type])

		if !assert.NoError(t, validate(s.db, spec.Type, withExtras), "%s accepts its required arguments", spec.Type) {
			continue
		}

		for name := range full {
			partial := maps.Clone(withExtras)
			delete(partial, name)

			if _, stricter := formStricter[spec.Type+"."+name]; stricter {
				assert.NoError(t, validate(s.db, spec.Type, partial), "%s does not require %s", spec.Type, name)
			} else {
				assert.Error(t, validate(s.db, spec.Type, partial), "%s requires %s", spec.Type, name)
			}
		}

		// An argument required only when another one has a value: the product
		// requires it then, and only then.
		for j := range spec.Args {
			arg := &spec.Args[j]
			if arg.When == nil || !arg.Required {
				continue
			}

			with := maps.Clone(withExtras)
			with[arg.When.Arg], with[arg.Name] = arg.When.Value, sampleKinds[arg.Kind]
			assert.NoError(t, validate(s.db, spec.Type, with), "%s accepts %s when %s is %s", spec.Type, arg.Name,
				arg.When.Arg, arg.When.Value)

			delete(with, arg.Name)
			assert.Error(t, validate(s.db, spec.Type, with), "%s requires %s when %s is %s", spec.Type, arg.Name,
				arg.When.Arg, arg.When.Value)
		}
	}
}
