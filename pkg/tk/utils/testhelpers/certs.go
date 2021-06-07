package testhelpers

// LocalhostCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1:6666" , "[::1]:6666" and "localhost:6666", expiring at
// Jan 29 16:00:00 2084 GMT.
const LocalhostCert = `-----BEGIN CERTIFICATE-----
MIICMzCCAZygAwIBAgIRAJFIx3lh/L57UPaTaMcBJ8wwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCBnzANBgkqhkiG9w0BAQEFAAOBjQAw
gYkCgYEAyLZU8wra4/hvLmEJWD/mdCq3BVW2zqmEa7gYZKyrNSN+iOzu9sLUR3fx
oo5UYT87x6xi+762QI+yiwZOxkdkbKv2yQXqpF6CO1J2IuCjbdwV9ZLapGsLT2jt
RUyR2w8qSQP7pl1Lk1K8mos+sdsRINX4VmsLG/pOukMyvUu7NTECAwEAAaOBhjCB
gzAOBgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/
BAUwAwEB/zAdBgNVHQ4EFgQUMlkJ+EgiVFx6OlaVub4NQ9HgRwEwLAYDVR0RBCUw
I4IJbG9jYWxob3N0hwR/AAABhxAAAAAAAAAAAAAAAAAAAAABMA0GCSqGSIb3DQEB
CwUAA4GBAGnpw8im001qnW+e+V339MBTabqvXvsaMKIf75+sYkGsFhLOYw+kT4fg
31bd3B7u5azc/FKfQdDOjjhvnGqoHtyjjVMhxLIN0fjugMTGxw4Er5xIC5RGuynB
lqNcbCum94NGVmx0wDs3WOgcN0GCpiasPZcFs7VoVanerLOBIMXj
-----END CERTIFICATE-----`

// LocalhostKey is the private key for LocalhostCert.
const LocalhostKey = `-----BEGIN PRIVATE KEY-----
MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBAMi2VPMK2uP4by5h
CVg/5nQqtwVVts6phGu4GGSsqzUjfojs7vbC1Ed38aKOVGE/O8esYvu+tkCPsosG
TsZHZGyr9skF6qRegjtSdiLgo23cFfWS2qRrC09o7UVMkdsPKkkD+6ZdS5NSvJqL
PrHbESDV+FZrCxv6TrpDMr1LuzUxAgMBAAECgYEAmKww5Arew8gG8lV3oTxCFR0k
yJcRjhPeGX4YeAPr22jbaFYp02QRyydOk2MGhk5uL41OYcYIpgVoP14V77cAiFvJ
V4GMSvp4+YACzK2/kpCm6vdeZJlrbix32kHJo3+HgCa7tlW5NyLjcRuGBdECsqqv
pDHb/Gb7oppXJqXjHQ0CQQDWeUJYVgwS+aTAFsdeDmQWwSDRksiBo6Y55LJAPR+U
zaHuQ4MeWm1olPwPwHsAlIJIvdlF31i9j7kaNC14/smvAkEA75L259DZj5JIpMP+
3x0pxduam0Mf258ibxNiuL/T3qt4kBwTUa2JlIFVNPlwc2sqhc0eJliTvGOWhpOG
2LMHHwJBAIH2jtZ6pexlrIjeBMehDtOfCiUUvj2YjiTsyXsVzupbxTFdZbnh8AR8
q1VcPOz4EQ7FREEL+3k6+16+mYOFWW8CQBtdkDJ+mrtZnE6lzLEzpZfiM9DUZAk0
Ljy93CL6Vnsy3vynGFXWGscJ1u/MJloovZy3B2Cd8ZItVf5dT6PlH0UCQQCxlbQB
uxd0sOxCNiQOnnD12ma8nn21Vex34lvKIAI6Zi55ikCtrezcrB67DI9ANEq8w6vl
sOYLXpHzyo7oGMVz
-----END PRIVATE KEY-----`

// ClientCert is a PEM-encoded TLS cert to be used for testing client TLS
// authentication
const ClientCert = `-----BEGIN CERTIFICATE-----
MIICJTCCAY6gAwIBAgIQIKHvcsM3cly5gnpNEdxSXTANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTgwMDEwMTAwMDAwMFoYDzIwOTQwMTI4MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
iQKBgQDPaGdQDzA/C4GaCgouUUXn0ngCIyNTTn6bjUSYI2Vd8EhILreb2Bl6848A
ScAR6E4+vlvNo7rWAGP9pHS2JqCfio/LHcudFoEiFvgEfk/+2WL9JNXjlRSBsuZm
tXlKLgb4Zg6NCQrFLH3HmJbo/EWRp716aXxfz6gbJYXg62a5GQIDAQABo3oweDAO
BgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwIwDwYDVR0TAQH/BAUw
AwEB/zAdBgNVHQ4EFgQUnjMDLHQqjzwodoDHRu82q9zkLLwwIQYDVR0RBBowGIcE
fwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG9w0BAQsFAAOBgQAD0G1ENOLg
Wf06w9SikUMzDHXWUsVA8PrODpWU0cmDY06sdpa4IIWKmhf95BVXnrOjJy7d3y1N
b1Wte/HVOk8zgAta5W5WnQAMPvXuXFaC3Jy0YmQfY1rSjl/PLbXzA0gO0IcP93UF
hZ0if1CWX+PzVETBXFKURT905E5qS+Ebng==
-----END CERTIFICATE-----`

// ClientKey is the private key for ClientCert.
const ClientKey = `-----BEGIN PRIVATE KEY-----
MIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAM9oZ1APMD8LgZoK
Ci5RRefSeAIjI1NOfpuNRJgjZV3wSEgut5vYGXrzjwBJwBHoTj6+W82jutYAY/2k
dLYmoJ+Kj8sdy50WgSIW+AR+T/7ZYv0k1eOVFIGy5ma1eUouBvhmDo0JCsUsfceY
luj8RZGnvXppfF/PqBslheDrZrkZAgMBAAECgYEAjHHsE4BVcTt/ZSmLP1X1ekdA
0GGu2Ah9HyQH4OWHDJdautY3qqYoiuNGYDGQiA/AfCg2zgciyyq0itrD1VxOwsG0
dO7yu5i9ooWnETV/tTZq1aM4HyeXaK/dl1LzJ+tBIVOeGa3AMQvSF84IjJEN9dYg
2a4BUh/nt+fmRNb52SECQQDupRSvff1rTmBjrZOOs9s56GSMryyjvggJHYcBhSyk
liagybxWxCinkUP0VdfESzd9j6xDhygO2Islq0BFr9FtAkEA3n3GNKpAzQ2QlyRr
w5cMECypYXdPyjNAG6rP/HB4adWJRxnMAGglRSmYNjitHLxG0+wo0IfDXq/5f4wZ
yvPm3QJAZhBqWWf8A3HA3cC11BluEEUpA9ZDtEAo9aUQQYEwh6/EE45UI5O/g3Mo
ag5wun4k3GmfFj5uznKkiFbGpUc9vQJAKvBLGE7jQq+jgAffZFf6VATKi6zjETri
3HQSv71U/9feLoKkBFAVIUvtvEkj36/WW3/wQI5y/gsoM51uPOTlYQJAVhFbI4s2
Zht/QWMq1v8BtVVZIFRksEIn3LIHga7Q5HpkqXmpl9lNh7s0DAvReDb3wyW0UxJS
vkxL195flB04sw==
-----END PRIVATE KEY-----`
