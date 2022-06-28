package compatibility

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/lib/log"
)

func initLogger() (*log.Logger, io.Reader, func()) {
	r, w, err := os.Pipe()
	So(err, ShouldBeNil)

	stdout := os.Stdout
	os.Stdout = w

	Reset(func() { os.Stdout = stdout })

	back, err := log.NewBackend(log.LevelWarning, "stdout", "", "")
	So(err, ShouldBeNil)

	once := sync.Once{}
	done := func() { once.Do(func() { So(w.Close(), ShouldBeNil) }) }

	return back.NewLogger("test_log_sha1"), r, done
}

const sha1Key = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`

const sha1Cert = `-----BEGIN CERTIFICATE-----
MIIFnjCCA4agAwIBAgIUWhM4ERmQN2p//QpVIhWirUg997kwDQYJKoZIhvcNAQEF
BQAweDELMAkGA1UEBhMCWFgxDDAKBgNVBAgMA04vQTEMMAoGA1UEBwwDTi9BMSAw
HgYDVQQKDBdTZWxmLXNpZ25lZCBjZXJ0aWZpY2F0ZTErMCkGA1UEAwwiMTIwLjAu
MC4xOiBTZWxmLXNpZ25lZCBjZXJ0aWZpY2F0ZTAgFw0yMjA0MDUxMTIyMzdaGA80
NzYwMDMwMjExMjIzN1oweDELMAkGA1UEBhMCWFgxDDAKBgNVBAgMA04vQTEMMAoG
A1UEBwwDTi9BMSAwHgYDVQQKDBdTZWxmLXNpZ25lZCBjZXJ0aWZpY2F0ZTErMCkG
A1UEAwwiMTIwLjAuMC4xOiBTZWxmLXNpZ25lZCBjZXJ0aWZpY2F0ZTCCAiIwDQYJ
KoZIhvcNAQEBBQADggIPADCCAgoCggIBALINIQiGjaXS7L2Rke33sWCBT4uVA05R
CmM8bGyBBTBQMIVGjZDQetuoJYwoVXJSPuJI3KEhzaB4dWA646G2ibYD4lzhBlpk
vVg1JZ79t5NTgW/qHjn/rlUbYDybdh4LRcR3Vfgcvf12zUy5w6vTDKlnWmCtkErv
CpgCSCzwX2yCWr6Wb7dwmQtIHKnLxSUCX7+OUYm1HtV3Tc/fDcRtn14dPcKMOz+K
0VNnxipOYD24OfMUMk/0UvCgdB1SEaSXjfdLVVBl3lGlk2pKaKcCqps6+7NyPFCW
Pp+wxfB1vnWP7cOHqEZeF1nPrIBjzN5lA/FD6JTOQkCevooobTQLoVMRBqXZ3p4g
eWtiSO7mkwi5/1rZ37RHJQSsCFO/W9YKUEcytnx32DBs5UXC7g36ZkMaUQUxRTXq
c9GBW1oEZRreQQwLDuaSQj3byi3bwdUL6mLclWF+pRO4VGptocwTExe/2EAUCjCc
cJ2SE3Cm1I0cSPjp60IsO5LreZUG8oWJsPkfG/m4U1tZnFv4FO8YUAAtEWglf7bY
dpTvDFWTy+ZWpsuSXsEcNovJBpEf2AWFJ/cTW1tCWRel8yennUmWehj6ocbWx3eO
BDvUt2lc32SVwgGWsSP5GKTn2lKVdD3rfWWbmRXdEqGcgxbYYexNI2jsyzqlqJlt
o7sFHVMoF75HAgMBAAGjHjAcMBoGA1UdEQQTMBGHBH8AAAGCCWxvY2FsaG9zdDAN
BgkqhkiG9w0BAQUFAAOCAgEANxVNlhjXkPwLx6cWMardNUiibZKRegyo/GDxxULH
FqR1wxzWigPOeAFBSopNrkseNnaB4THQZWh7pF1WrtOR4cVSFI/cXsf9rNzEziWE
/dISvDd8WRgRgz1p6N5ZWLET/EHImaOQAbwa80iJj9bnGOHaw+zxYV+/4P9X+zYP
k65m3eYPRizCDC0l/K14eZ/np5FOI1EEumYesOB6Ozltc8fXzqNfIplSbOGTnJ2j
RSOKsUA6xCj/WAS//mOsYKh6DdgPN1xovfya5f/uxeHadrQcWgS7W8+ehVHe97Hm
T2HcaQkQ5KgChbwixDKjLuKd3tPWzu7yAz2T0jxYDbRlcZFUlZ63ekCWbee4TfLp
MwA4dXth5zv9XVaFBiXg1H3nJabcqhB+YZOLYYvTYCu/zdzJvR130T3SiQTl+04O
OZgE/4FN0Q5XXhRRUj6EDCukn8T4bWrT8OiFhrtHMsZwPHn+Ghl/J35VOOt2OzDe
13KmrvNoda/pOCZwwolgjryH/egIESyKMdhVwfis6SuWR7KMQqqMOrkq4ey3YF+S
KP5eIe3oX4nU602qnb31VIfURhetXkL3rc8KZO65KP22nF4U4NsW7bHOCFmxJ2Em
BBjhUyav8tNB2dneTL6mA/u+F1SeA/FRSowbuFWyA1/PixXTN4Kqdp5d2iMsj9rO
BeA=
-----END CERTIFICATE-----`

func TestLogSha1(t *testing.T) {
	Convey("Given a TLS certificate signed with SHA1", t, func() {
		cert, err := tls.X509KeyPair([]byte(sha1Cert), []byte(sha1Key))
		So(err, ShouldBeNil)

		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		So(err, ShouldBeNil)

		So(cert.Leaf.SignatureAlgorithm, ShouldEqual, x509.SHA1WithRSA)

		Convey("Given a TLS server with the SHA1 certificate", func() {
			list, err := tls.Listen("tcp", "localhost:0", &tls.Config{
				Certificates: []tls.Certificate{cert},
			})
			So(err, ShouldBeNil)

			go http.Serve(list, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			Convey("When connecting to the server", func() {
				logger, reader, done := initLogger()
				defer done()

				//nolint:forcetypeassert //assertion will always succeed
				client := http.DefaultTransport.(*http.Transport)
				client.TLSClientConfig = &tls.Config{
					VerifyConnection: LogSha1(logger),
					RootCAs:          x509.NewCertPool(),
				}
				client.TLSClientConfig.RootCAs.AddCert(cert.Leaf)

				req, err := http.NewRequestWithContext(context.Background(),
					http.MethodGet, "https://"+list.Addr().String(), nil)
				So(err, ShouldBeNil)

				resp, err := client.RoundTrip(req)
				So(err, ShouldBeNil)
				defer resp.Body.Close()

				Convey("Then it should have logged the deprecation of SHA1", func() {
					done()
					bytes, err := io.ReadAll(reader)
					So(err, ShouldBeNil)

					So(string(bytes), ShouldContainSubstring, "[WARNING ] test_log_sha1: "+
						"The certificate of partner 'localhost' is signed using "+
						"SHA-1 which is deprecated. All SHA-1 based signature "+
						"algorithms will be disallowed out shortly.")
				})
			})
		})
	})
}

func TestCheckSha1(t *testing.T) {
	Convey("When calling 'CheckSHA1' with a TLS certificate signed with SHA1", t, func() {
		warn := CheckSHA1(sha1Cert)

		Convey("Then it should have returned a warning message", func() {
			So(warn, ShouldEqual, "This certificate is signed using SHA-1 "+
				"which is deprecated. All SHA-1 based signature algorithms "+
				"will be disallowed out shortly.")
		})
	})

	Convey("When calling 'CheckSHA1' with any other string", t, func() {
		warn := CheckSHA1("foobar")

		Convey("Then it should return nothing", func() {
			So(warn, ShouldBeEmpty)
		})
	})
}
