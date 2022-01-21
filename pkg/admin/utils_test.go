package admin

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
)

//nolint:gochecknoinits // init is used by design
func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")
}

const cert = `-----BEGIN CERTIFICATE-----
MIIFkzCCA3ugAwIBAgIUHrLpS5newrfNk+dgVZ1uxogSCuMwDQYJKoZIhvcNAQEL
BQAwWTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDESMBAGA1UEAwwJbG9jYWxob3N0MB4X
DTE5MDUxNDEwNTY1NVoXDTI5MDUxMTEwNTY1NVowWTELMAkGA1UEBhMCQVUxEzAR
BgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5
IEx0ZDESMBAGA1UEAwwJbG9jYWxob3N0MIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEA2d+eU+/48mMRD6wavDcGJ8K1F4ewmnNYv0izem+ytBZar5A9QNjq
B+vzeLnHU89FYrBze2va6pk0WRoP8GKYnGiNMCzjXlL0mYZgrpxItT2Dgc9J467M
RHs74CQOUiGU9lgP2uPQLWfI4/IFax6DEe+OaQYAq0Hn33+RuBjfrLJrZno0yQaJ
94hYa7OeAxlAXn1R1JDieiZgBalPtk15a5TQtDdoGOQPQakYb1JWQb+pdTDlwmSP
7iVl9yDcz68zg4gr0RP/RoFod+1TvzU+z9zr3/gYsAGvPaY94nK0lTHM+25e6FCT
VIQwwKfx+6jIwxmW73on29a9H1FqJbGR3Q/PWlCMC3nZCNfeR6W+uMFrsIIpGeQQ
aaqnbxQX/jqfFtQt4kJzytCSRF9vt26zLIg8HIWlgZ+kfvksAXd2pNMi/5IzuCD/
stsw7NrAxvXqgupFqNkkeFamd8PBoDCFvE1aiKEPj9eHGjSrV25reYJqqLvpF4u7
ynHv4zTZK0a/WIcXfDkGoEraVzR6efnCa+eaH7/sFAJWVq1pvRAB195t92Md3NjO
T1/zR8lBGm6w/FaMjtUpr2oZ179OOSN1Lo6Oajv4xGGZ75Ktg1szGKiDX9S40qLX
ZdBU+g0oQJ2Qbo+TrNwinyP2V8R2CqMAV1ySDUYFdoPu4fkTKGvHNS0CAwEAAaNT
MFEwHQYDVR0OBBYEFDh83HFW8ydBBmAFW8vZtyq+dh9uMB8GA1UdIwQYMBaAFDh8
3HFW8ydBBmAFW8vZtyq+dh9uMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEL
BQADggIBAG2rlWryw0FuKUdeDin2hklqSY6fQzChbuQXPwIxEcOUsqcKccM0Nx2r
ucOECoeXNtIdZQIoGG7ujNPKlag0L+YV4/sAZflnhq79L6kNkPrzp+U+Va+0INL1
tavkWsQVwcjH2f+6OQgkzRRsR885mv60sFoTljYsmfl59MdV5els8buAV7vVFCmN
MpRVTCeTnq/NYWphcTDpfsWMfBnaWKAslu78tAMh97I9hEQNwmIHpu0L23fYiSMv
kL/6murfQ2TIgq1BytkpOhTa0wsLx9gPdACPyjyy8HJr+Jjaj3B4C3I3PCXj0NVX
R9UqfnrZcpVo6tGR4bqqRcpxlrgn8akVfpw7KwZxjFBxnRQw+tnWzW+V9yl6vRrB
OdHHQi8HmasBFQaaah2qVzYD5L8j/Mu2s6AHbk66u9LlkS4mNfEqeUsKo7go5Yd5
uUtdrXQvZpann/uVe739hBqVfVCJ5Cul8JVVroESU5/6rdiPKPjhxbZuMOrefLMj
GsWCImLbQiKt3RglU/6dL7in2QZ5YcD0SUIcvchHovRc6AO8nFeqg9PCmKR+ALN2
shaFiLSLUza3yD221LiPrtNuIIvn+snM3ljmErGWobgYD3DX8CDIeaMV/UuYXLsQ
CB6bziw87nzY/+giAaHj6kMM7ifHKhtQg0RAMVxhv2Jkoi0+9hTC
-----END CERTIFICATE-----`

const key = `-----BEGIN PRIVATE KEY-----
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
wom8hSOMFJQWLjNaqzHEKcqlp//TEcI=
-----END PRIVATE KEY-----`
