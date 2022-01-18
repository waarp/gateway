package compatibility

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

//nolint:gochecknoinits //init is needed here to parse the legacy R66 cert
func init() {
	var err error
	if LegacyR66Cert, err = tls.X509KeyPair([]byte(LegacyR66CertPEM),
		[]byte(LegacyR66KeyPEM)); err != nil {
		panic(fmt.Sprintf("failed to load legacy R66 cert: %v", err))
	}

	if LegacyR66Cert.Leaf, err = x509.ParseCertificate(LegacyR66Cert.Certificate[0]); err != nil {
		panic(fmt.Sprintf("failed to parse legacy R66 cert: %v", err))
	}
}

// IsTLS checks whether the given R66 proto config map contains an "isTLS"
// property, and whether that property is true. If the property does not exist,
// this returns false.
//
// The "isTLS" property has been replaced by the "r66-tls" protocol, but to
// maintain backwards compatibility, the property has been kept. This function
// can be used to check its presence.
func IsTLS(mapConf map[string]any) bool {
	if isTLSany, hasTLS := mapConf["isTLS"]; hasTLS {
		if isTLS, isBool := isTLSany.(bool); isBool && isTLS {
			return true
		}
	}

	return false
}

const AllowLegacyR66CertificateVar = "WAARP_GATEWAY_ALLOW_LEGACY_CERT"

//nolint:gochecknoglobals //global vars are required here
var (
	IsLegacyR66CertificateAllowed = os.Getenv(AllowLegacyR66CertificateVar) == "1"
	LegacyR66Cert                 tls.Certificate
)

func IsLegacyR66Cert(cert *x509.Certificate) bool {
	if IsLegacyR66CertificateAllowed &&
		cert.SignatureAlgorithm == x509.SHA256WithRSA &&
		bytes.Equal(cert.Signature, LegacyR66Cert.Leaf.Signature) {
		return true
	}

	return false
}

const LegacyR66CertPEM = `-----BEGIN CERTIFICATE-----
MIIDdzCCAl+gAwIBAgIENnEdtTANBgkqhkiG9w0BAQsFADBsMRAwDgYDVQQGEwdV
bmtub3duMRAwDgYDVQQIEwdVbmtub3duMRAwDgYDVQQHEwdVbmtub3duMRAwDgYD
VQQKEwdVbmtub3duMRAwDgYDVQQLEwdVbmtub3duMRAwDgYDVQQDEwdVbmtub3du
MB4XDTEzMDQyOTA5NTQ1N1oXDTEzMDcyODA5NTQ1N1owbDEQMA4GA1UEBhMHVW5r
bm93bjEQMA4GA1UECBMHVW5rbm93bjEQMA4GA1UEBxMHVW5rbm93bjEQMA4GA1UE
ChMHVW5rbm93bjEQMA4GA1UECxMHVW5rbm93bjEQMA4GA1UEAxMHVW5rbm93bjCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAITDuqCocEqDFTVuHusQmb24
L5haEN4tSRbULD9NHe0SehLU+3kXrSm97m6ffIbBj95ChocvMQhCpwQsfiTNa+pT
dMTlwWN/jJEwAgfphqsDndoI+laGYJeeEhByxUaFQ608QXBUVigCdirz/T5cbkXl
jmYWA9Rar259vefE6Eubfb/wS2kBKTbP96IqOH84R2Edsl45KM6tHVXh8/VynQdZ
MVqJrMg5julPF1d0/Y/4UYoemOV+qaVrnawriZvg7+o8MLb1v7I7yok3lxpt/9TB
Rs23OElpCzNlY7Zz3f0BD+lt8ZpoeXR7rN+1RMm0VwlIrr6Sske6211AtG3+qwEC
AwEAAaMhMB8wHQYDVR0OBBYEFEVpbVKGEkgeXHQ0tN5lzLr36mVsMA0GCSqGSIb3
DQEBCwUAA4IBAQBI17aBzzRZoQP8BSLTCPIJgApjoD0DWJ8TyKb76uLADcTVqtc4
B1m7di0B6PT2dIT7+E4Ek0twOvfpUyPcgNUz0auAfBF27PJMSu2hug9HdSndvFBx
aDANCVj1H7S+QgQpxQXNs6d9mwfpyOS/SCKE7Xy26/kKbQ1oWop2gV7w/+0LHK9A
AdsPDkoTxPZFcPS7kZwg+eAov+DEOksOkWeHGypdFEsn4RqT67RheCeOZsiED4zh
ADwdXdxu2+QYlSw3p/vuriC6FEWJ4E7MPMbFTUbax3Zx7ejvs8e/DgfjZZHkwkFd
SRJz9CF60Oo+fqyp/TvfM/p3So5W5kAs9MLs
-----END CERTIFICATE-----`

const LegacyR66KeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCEw7qgqHBKgxU1
bh7rEJm9uC+YWhDeLUkW1Cw/TR3tEnoS1Pt5F60pve5un3yGwY/eQoaHLzEIQqcE
LH4kzWvqU3TE5cFjf4yRMAIH6YarA53aCPpWhmCXnhIQcsVGhUOtPEFwVFYoAnYq
8/0+XG5F5Y5mFgPUWq9ufb3nxOhLm32/8EtpASk2z/eiKjh/OEdhHbJeOSjOrR1V
4fP1cp0HWTFaiazIOY7pTxdXdP2P+FGKHpjlfqmla52sK4mb4O/qPDC29b+yO8qJ
N5cabf/UwUbNtzhJaQszZWO2c939AQ/pbfGaaHl0e6zftUTJtFcJSK6+krJHuttd
QLRt/qsBAgMBAAECggEAE9avj4w741Z9F9PRuOxtHMVmD0z+EkUQE+I2jmr2mtNU
/HVo8mpQTNl9xHf+gqBv4BVuxsqNeB+Fl4EShGtRwd0gqL9wS27m0VcsJoSFxA4x
S0BmMAG6c02Cg4Sy59vIBh3n5WIk0au0fqyg3e2v6K/pvGVzwwqeBlOxye1JjOqD
G3aL2UefVjxPgLLE1mDoqV5ZIN2+XRXGFHJlvhA50RVDq1KQldFcbWrVTZf+Igi7
XFLR+hIOFoZmLku2BHxXBjZRJO7REV8HbT/zIHi0iFv7IK/x+66r/wL8rLiwFGeK
yA61EF0jPECgOxXURTZgTxhDwC9QPDmNSdgM1F1IBQKBgQC+Gtrc0P0fOQjehgyP
4sHhvO/2BUKGUmi5c7QawE/ja2ueefosmGRU87l3bV4x2+GrR9yX5ymv08bVtJwC
u/yncnyx6mjkMaiNXBtfrdNhKWN4GQJDF2GNur+hpXNvtBmlvulSBngbCwPrxjKa
daflVYbADyreaO7iXMUgWjJZrwKBgQCyyLkem0Vm39r44Knxq/iGx/CAD3vsGnGI
FUx0a+bxhFIKYQm9MLJtGN5Ag6kP+76snBLxJ6JSwxIBpG9JYrFLaEN49oiswcty
mfO2zIUoZ8CHnFdoR0POXDTWLTLPWCd0ogxzDsVTKT4gavA9WErvFr0twIAMqS/Y
LzbV9+BiTwKBgH2tR0+AIjbH/+MMf7WH1WElBQaCB67BQFaJ9WFSDf5s/6KvRQLC
ZGH9FnmrpgAUOyZ+xYju25JP0T1qv1DXcnpIp8L/EwT5B1Mct0QTqJCtSgMVlXdB
N874zMNSm/QW/nWitqDxgelu6NKwHrgaXDqyxfimjlKm0HZ5miB/QJYlAoGAEyid
ZeE/w7Fzdr4kmAhUvqTIagC+x+NhjTKzGbrCadlDLWeOsp54UGac0o8JW/QfT8H9
6afUpkfPMyva3SNdWnZW3KyWouS1l5dV3Z33GwhbQm0HlN4mLwQEiXsYec25lK8U
5HONw8akqLas/fXrOcnXBgMd9b1fqiwNFUrV2dMCgYAnRZ7Ig3w+pkc5dAV22SNO
4M3JJYqCiGBoGJR/w5IP1FgT+IshA/5fIBJl7s8Cg8aaWWoRYuLLjA1xTFqw+Ma9
wvThKXCE78uQIzRIyp9X6W+enbMKesrtprpsZlBHU/lZ5m/bh3EXBuCFV1Q2rrVc
5VAeza4keDveGJVWVTdTlw==
-----END PRIVATE KEY-----`
