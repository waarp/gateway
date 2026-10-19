package protoform

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
)

func translator(t *testing.T) func(string, ...any) string {
	t.Helper()

	catalog, err := locale.LoadCatalog()
	require.NoError(t, err)

	return catalog.Translator("fr")
}

// Every protocol registered in the gateway belongs to a family the interface
// knows, and the families follow the registry.
func TestFamiliesFollowTheRegistry(t *testing.T) {
	t.Parallel()

	families := Families()
	require.NotEmpty(t, families)

	for key := range protocols.List() {
		p := ProtocolOf(key)
		assert.Containsf(t, variants, key, "protocol %q is registered but unknown to the interface", key)
		assert.True(t, IsFamily(p.Family.Key), key)
		assert.Equal(t, p.Family.Key, FamilyKeyOf(key), "a product key reads as its family")

		resolved, ok := Resolve(p.Family.Key, p.TLS)
		require.True(t, ok)
		assert.Equal(t, key, resolved, "family + TLS gives the key back")
	}

	r66Family := ProtocolOf(r66.R66TLS).Family
	assert.Equal(t, "R66", r66Family.Label)
	assert.Equal(t, r66.R66, r66Family.Plain)
	assert.Equal(t, r66.R66TLS, r66Family.TLS)

	sftp := ProtocolOf("sftp").Family
	assert.False(t, sftp.HasTLS(), "SFTP has no TLS variant")

	key, ok := Resolve("sftp", true)
	assert.True(t, ok)
	assert.Equal(t, "sftp", key, "TLS asked on a family without it gives the plain variant")

	_, ok = Resolve("bogus", false)
	assert.False(t, ok)
	assert.Empty(t, FamilyKeyOf("bogus"))

	unknown := ProtocolOf("bogus")
	assert.Equal(t, "bogus", unknown.Family.Label)
	assert.False(t, unknown.TLS)
}

func TestDescribe(t *testing.T) {
	t.Parallel()

	translate := translator(t)
	assert.Equal(t, "R66", Describe(translate, ProtocolOf("r66")))
	assert.Equal(t, "R66, chiffré par TLS (R66-TLS)", Describe(translate, ProtocolOf("r66-tls")))
	assert.Equal(t, "HTTP, chiffré par TLS (HTTPS)", Describe(translate, ProtocolOf("https")))
}

// The options of a role are the ones the product reads for that role: a
// server has the passive mode of FTP, a client its active mode; both have
// the R66 block size; the hash algorithm belongs to the client.
func TestOptionsByRole(t *testing.T) {
	t.Parallel()

	names := func(role Role, family string) []string {
		var out []string
		for _, opt := range optionsOf(family, role) {
			out = append(out, opt.Name)
		}

		return out
	}

	assert.Equal(t, []string{"ftp_disable_passive", "ftp_disable_active", "ftp_passive_min", "ftp_passive_max",
		"ftp_tls_requirement"}, names(RoleServer, FamilyFTP))
	assert.Equal(t, []string{"ftp_enable_active", "ftp_active_address", "ftp_active_min", "ftp_active_max"},
		names(RoleClient, FamilyFTP))
	assert.Equal(t, []string{"r66_login", "r66_block", "r66_no_final_hash", "r66_check_block_hash"},
		names(RoleServer, FamilyR66))
	assert.Equal(t, []string{"r66_block", "r66_final_hash_algo", "r66_no_final_hash", "r66_check_block_hash"},
		names(RoleClient, FamilyR66))
	assert.Empty(t, names(RoleClient, FamilyHTTP))

	// Every option has a label the catalogue knows, and a hint when it names one.
	translate := translator(t)

	for family, opts := range familyOptions {
		for _, opt := range opts {
			assert.NotEqual(t, opt.Label, translate(opt.Label), "%s.%s: label %q unknown", family, opt.Name, opt.Label)

			if opt.Hint != "" {
				assert.NotEqual(t, opt.Hint, translate(opt.Hint), "%s.%s: hint unknown", family, opt.Name)
			}

			if opt.Kind == KindSelect {
				require.NotNil(t, opt.Choices, opt.Name)
			}

			if opt.Kind == KindList {
				require.NotNil(t, opt.Values, opt.Name)
				assert.NotEmpty(t, opt.Values(RoleServer), opt.Name)
			}
		}
	}
}

// A form filled from an object and applied again gives the object back; a
// posted form resolves the family and TLS into the key of the product.
func TestFormRoundTrip(t *testing.T) {
	t.Parallel()

	translate := translator(t)

	t.Run("Given an FTPS server with options", func(t *testing.T) {
		t.Parallel()

		cfg := map[string]any{
			"disablePassiveMode": true, "passiveModeMinPort": 40000, "passiveModeMaxPort": 40100,
			"tlsRequirement": "mandatory", "minTLSVersion": "v1.3",
		}

		f := NewForm(RoleServer)
		f.Fill("ftps", cfg)
		assert.Equal(t, FamilyFTP, f.Family)
		assert.True(t, f.TLS)
		assert.True(t, f.Validate(translate))
		assert.Equal(t, "ftps", f.Protocol())

		applied := f.Apply()
		assert.Equal(t, true, applied["disablePassiveMode"])
		assert.Equal(t, false, applied["disableActiveMode"])
		assert.Equal(t, uint16(40000), applied["passiveModeMinPort"])
		assert.Equal(t, uint16(40100), applied["passiveModeMaxPort"])
		assert.Equal(t, "mandatory", applied["tlsRequirement"])
		assert.Equal(t, "v1.3", applied["minTLSVersion"])
		assert.NotContains(t, applied, "cipherSuites", "an empty list leaves the product default")

		f.Prepare(translate)
		assert.Equal(t, "ftp", f.Choice.Families[slicesIndex(f.Choice.Families, "ftp")].Value)
		assert.True(t, f.Choice.TLS)
		assert.Contains(t, f.Choice.TLSCases, "ftp")
		assert.NotContains(t, f.Choice.TLSCases, "sftp")
	})

	t.Run("Given a posted R66 client with TLS", func(t *testing.T) {
		t.Parallel()

		f := NewForm(RoleClient)
		f.Read(url.Values{
			FieldFamily: {"r66"}, FieldTLS: {"1"}, "r66_block": {"32768"}, "r66_final_hash_algo": {"SHA-512"},
			"r66_check_block_hash": {"1"}, FieldMinTLS: {"v1.2"},
			FieldTLSCiphers: {"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384", ""},
		})
		require.True(t, f.Validate(translate))
		assert.Equal(t, "r66-tls", f.Protocol())

		cfg := f.Apply()
		assert.Equal(t, uint32(32768), cfg["blockSize"])
		assert.Equal(t, "SHA-512", cfg["finalHashAlgo"])
		assert.Equal(t, true, cfg["checkBlockHash"])
		assert.Equal(t, false, cfg["noFinalHash"])
		assert.Equal(t, []string{"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"}, cfg["cipherSuites"])
		assert.NotContains(t, cfg, "serverLogin", "a server option is not stored on a client")
	})

	t.Run("Given TLS asked on SFTP", func(t *testing.T) {
		t.Parallel()

		f := NewForm(RoleServer)
		f.Read(url.Values{FieldFamily: {"sftp"}, FieldTLS: {"1"}, FieldMinTLS: {"v1.0"}})
		require.True(t, f.Validate(translate))
		assert.Equal(t, "sftp", f.Protocol())
		assert.NotContains(t, f.Apply(), "minTLSVersion", "the TLS block does not apply")
	})

	t.Run("Given a bad number and an unknown family", func(t *testing.T) {
		t.Parallel()

		f := NewForm(RoleServer)
		f.Read(url.Values{FieldFamily: {"pesit"}, "pesit_checkpoint_window": {"300"}})
		assert.False(t, f.Validate(translate), "300 does not fit the window (uint8)")
		assert.True(t, f.HasErrors())
		f.Prepare(translate)

		var found bool
		for _, family := range f.Families {
			if family.Key != FamilyPeSIT {
				continue
			}

			for _, opt := range family.Plain.Fields {
				if opt.Name == "pesit_checkpoint_window" {
					found = true
					assert.Equal(t, "300", opt.Value)
					assert.NotEmpty(t, opt.Error)
				}
			}
		}

		assert.True(t, found)

		f = NewForm(RoleServer)
		f.Read(url.Values{FieldFamily: {"bogus"}})
		assert.False(t, f.Validate(translate))
		assert.Empty(t, f.Protocol())
		assert.NotEmpty(t, f.Error)
	})
}

func slicesIndex(choices []Choice, value string) int {
	for i, c := range choices {
		if c.Value == value {
			return i
		}
	}

	return -1
}

// The rows of a details page say the effect for the operator, with the
// product's default when nothing is set.
func TestRows(t *testing.T) {
	t.Parallel()

	translate := translator(t)

	rows := Rows(translate, RoleServer, ProtocolOf("r66-tls"), map[string]any{"noFinalHash": true, "blockSize": 1024})
	labels := map[string]string{}

	for _, row := range rows {
		labels[row.Label] = row.Value
	}

	assert.Equal(t, "Le nom du serveur", labels["Identifiant attendu du partenaire"])
	assert.Equal(t, "1024", labels["Taille d'un bloc de données"])
	assert.Equal(t, "Non vérifié", labels["Hachage en fin de transfert"])
	assert.Equal(t, "Non vérifié", labels["Hachage de chaque bloc"])
	assert.Equal(t, "TLS 1.2", labels["Version TLS minimale"])
	assert.Equal(t, "Toutes les suites sûres de la passerelle", labels["Suites de chiffrement TLS"])

	rows = Rows(translate, RoleClient, ProtocolOf("ftp"), map[string]any{"enableActiveMode": true, "activeModeMinPort": 5000})
	labels = map[string]string{}

	for _, row := range rows {
		labels[row.Label] = row.Value
	}

	assert.Equal(t, "Activé", labels["Mode actif"])
	assert.Equal(t, "0.0.0.0", labels["Adresse du mode actif"])
	assert.Equal(t, "5000", labels["Premier port du mode actif"])
	assert.Equal(t, "N'importe quel port libre", labels["Dernier port du mode actif"])
	assert.NotContains(t, labels, "Mode passif")

	rows = Rows(translate, RoleClient, ProtocolOf("http"), nil)
	require.Len(t, rows, 1)
	assert.Equal(t, "Aucune option pour ce protocole", rows[0].Value)
}

// The hash algorithms offered for R66 are the ones the module accepts: the
// list starts with the product's default and every entry is a known hash.
func TestR66HashAlgos(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "SHA-256", r66HashAlgos[0])

	for _, algo := range r66HashAlgos {
		cfg := &r66.ClientConfig{SharedClientConfig: r66.SharedClientConfig{FinalHashAlgo: algo}}
		require.NoError(t, cfg.ValidConf(), algo)
	}
}
