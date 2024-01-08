package testhelpers

// LocalhostCert is a PEM-encoded TLS cert with SAN IPs "127.0.0.1" , "[::1]"
// and "localhost", expiring on Jan 29 16:00:00 2084 GMT.
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

// OtherLocalhostCert is a PEM-encoded TLS cert similar with the same
// characteristics as LocalhostCert.
const OtherLocalhostCert = `-----BEGIN CERTIFICATE-----
MIICMjCCAZugAwIBAgIQZc/RAR/MfaG39KmM87xdejANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
iQKBgQCpsUuT74yxJYZSojX8vFJK8Jha64EeLOz6ExW86VkdYa4wNocmbc6qxDms
LAmyC1Li7Z02IKi+ssHHFoKDJ1MXkAHJFBjtlHECktImR7mE5ktFNKOqqpItBrUg
8F9fU8iytljjrIc9UDLN2kF7t2cEOMuYfnBnWttDnmHd85vjkwIDAQABo4GGMIGD
MA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8E
BTADAQH/MB0GA1UdDgQWBBTftkE6/G+N6LAmXiz25lNPNZDUbDAsBgNVHREEJTAj
gglsb2NhbGhvc3SHBH8AAAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQEL
BQADgYEAJjqFG/wPjPWBGWJk1jxxE+gp06kttQ1r/PrJDiIEzLdvl3egd9ewNgSj
rg6YSHYJdUCsiWJn6p59FirqXq1TmkPx320YZJux5zwL/PdThcRsWGZe5WvQH97i
1DenaJitih8P2oUx0z6oi3Z9J8CYO/kHb+f6nB4TJrF7pwjZ/TU=
-----END CERTIFICATE-----`

// OtherLocalhostKey is the private key for OtherLocalhostCert.
const OtherLocalhostKey = `-----BEGIN PRIVATE KEY-----
MIICdwIBADANBgkqhkiG9w0BAQEFAASCAmEwggJdAgEAAoGBAKmxS5PvjLElhlKi
Nfy8UkrwmFrrgR4s7PoTFbzpWR1hrjA2hyZtzqrEOawsCbILUuLtnTYgqL6ywccW
goMnUxeQAckUGO2UcQKS0iZHuYTmS0U0o6qqki0GtSDwX19TyLK2WOOshz1QMs3a
QXu3ZwQ4y5h+cGda20OeYd3zm+OTAgMBAAECgYBM03cbL/4riYipysTUetJrGUhx
CWR4q/BSO+fAkokFE19Qdq9zh41kpNhTidCN6jCJivh9NAYR6E5w+1D1Xg1wxbMj
cJfzOHNuSWs79Q5vOma594G0DEVRJ15gUOZYfClhLFp9xpDW8qWp/Jr86VuOfvg+
1IGgAY4/v+PPWVkosQJBAMpDJ2IMOZxNPe6batUMTqpWlJQLig6F+9UwgdJHYmIb
zzsnXJ0l0kEP4bGX2cEeDFDzZYeKDxofgoHEOtGVtwkCQQDWxunLAcXi1x6h3wIx
1YuKShYU120HDpZKNOkd/AWEDvFA9SpRVf+9jchpMPmTcGzeyZOQrjT0UD0g99K7
YbC7AkEAoUVgPoZe3SidjAYx7YOtqZX1TAHTZ3cfcEIIEUFHydBAsPrWlpqUhboD
C9Z3EstBqL4ZozPKjNq49X0gJQKJ+QJBALFyUkK5SkcqOFLAr02BAvLUVv4NJjUI
Zj8VC+0MBXqf5c8nPzlH9i0j8QqHvguEfU5r+JE2fOXFTVvppJ/QDHUCQDTw216/
RQrC3vHpZAIZvmWhW+ds9uAr8nCYItE5yDCUj3AZvld3THPhvGXy0luB1MRDpmx1
a0UAaV8gdLa5KuE=
-----END PRIVATE KEY-----`

// ClientFooCert is a PEM-encoded TLS cert to be used for testing client TLS
// authentication. The subject's CommonName is 'foo'.
const ClientFooCert = `-----BEGIN CERTIFICATE-----
MIICEzCCAXygAwIBAgIRAIY589MsqFvOXfGixwA9NagwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCBnzANBgkqhkiG9w0BAQEFAAOBjQAw
gYkCgYEA2aPHsgqq/3/jqRk/13AAwQvZlnlUaK5+9U2gQCGHpMuR4mOZ0BcrJQAK
fAIeQNSCao50EF9b6akU1SSWTsTSL7x0ect+4LTLHN4PldneItSF1z4+E+L/Wou+
5IF8ukk9uF6nGB+/Nr57huTMgb1YbQXVajSiFQjCgPrRzOc3apkCAwEAAaNnMGUw
DgYDVR0PAQH/BAQDAgKkMBMGA1UdJQQMMAoGCCsGAQUFBwMCMA8GA1UdEwEB/wQF
MAMBAf8wHQYDVR0OBBYEFIJzqwXlLAwf3Ck5tu1xYkGHpbnxMA4GA1UdEQQHMAWC
A2ZvbzANBgkqhkiG9w0BAQsFAAOBgQCt/W06Xb1OxFxdzLy3+9rtslkWo6rPa5t0
JyaWMZEcuQxWsOQ1zY883m6rCzntfxGQ8vqw3uJUXgGiFIvGJznbspqjV39gSkaA
xorASmIByRxkct4H26r9Smjta1qgfqv7OUOz1oAlcd6HZSWsz8gqwgizzxwS2ALi
xGEB2kD1+A==
-----END CERTIFICATE-----`

// ClientFooKey is the private key for ClientFooCert.
const ClientFooKey = `-----BEGIN PRIVATE KEY-----
MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBANmjx7IKqv9/46kZ
P9dwAMEL2ZZ5VGiufvVNoEAhh6TLkeJjmdAXKyUACnwCHkDUgmqOdBBfW+mpFNUk
lk7E0i+8dHnLfuC0yxzeD5XZ3iLUhdc+PhPi/1qLvuSBfLpJPbhepxgfvza+e4bk
zIG9WG0F1Wo0ohUIwoD60cznN2qZAgMBAAECgYBsTbFhayefD6BWFOeNKQJnDqOP
2v7jPPqWzbNSVp0up9MICrKPOAhTWErfXRp6/oWLyyn8v8d4ZpikXJmjxxQknDJP
zCXHqSZb/0v2hJlEHAimyyZRdB1AIaY4vQXfUEULXEwtqyoU1VgRtfV/FCU1lbBu
ZrKBxoHNtxNibOPbgQJBAObGPHTqmwO/CpP2xthg/QBZ0LCAKqheHof9w/swWUI5
P0AStsMgM6sMOngqEQ5TfamTe6NVtDNbcdRDlHMRvvECQQDxbf+O2UGcPtd5kF4n
td0XfgYxtDxOXbnGPe7hsdU1a/Mprr9bwngEMQjq1NekaYOyFdT8D9iWY5iDIiCO
qTYpAkAZ65AkzakFnbKRdflVmmcwX+YpvOuNp6ykN6OIliCgaI+rIa73cal7/86d
apQp2MTXhCIx8VFhJ1c8sS5+UjLxAkBJa2QeMt+K/mlUpJydgubbcA2+K8tzIXmP
WeI9bHEkL9HgyS2UYA1TaP4HO/bgHt5X19/PT5pUEbGdn1E7USYRAkAWjX1/l2Zn
/JWVDmWuasZv+R0qvdc8PxOy8xzQ7cu5hWpWAFQ9jyIParjaaaz8aHpvpzVej57D
mB4z1PJWsOsa
-----END PRIVATE KEY-----`

// ClientFooCert2 is a PEM-encoded TLS cert to be used for testing
// client TLS authentication. The subject's CommonName is 'foo'.
const ClientFooCert2 = `-----BEGIN CERTIFICATE-----
MIICEzCCAXygAwIBAgIRAIMmI5oWMM3rqvZCI0ljwNMwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCBnzANBgkqhkiG9w0BAQEFAAOBjQAw
gYkCgYEAyOs2/fhLMfvwfpwUgjSX6RrcehHENaxamkW7qZ1LbYuajfGJZ17B0TNq
pRT16xVjTGgEuYSq703IvzHXnLKSP7eRLjRSC7AXC8/p+URJ+q0Q0oepeXe5Za/7
mFf3mkTilTO+1FDEBdgRexCU0pGCrI/Y/g5gIOEY/NrSol/cc+0CAwEAAaNnMGUw
DgYDVR0PAQH/BAQDAgKkMBMGA1UdJQQMMAoGCCsGAQUFBwMCMA8GA1UdEwEB/wQF
MAMBAf8wHQYDVR0OBBYEFHLCjFis12MScQ0mOBtZr3ROBnZeMA4GA1UdEQQHMAWC
A2ZvbzANBgkqhkiG9w0BAQsFAAOBgQA39v2t6w7Bz1l/bwtpZ1lRdG4UWz4rAHPr
BKxr6IAb6fsIzxmljSsCY94mstXawoXHAyOYt+ZpP3/ivJHRHimiQ/La5DMxVog+
vDAkfuy+jcH7YrjU7wZEcbXaPj84C0F1ZCHDLOsQFv5CtgrdhLpJkGPeF3U3WGMO
thTTJIHQPw==
-----END CERTIFICATE-----`

// ClientFooKey2 is the private key for ClientFooCert.
const ClientFooKey2 = `-----BEGIN PRIVATE KEY-----
MIICdwIBADANBgkqhkiG9w0BAQEFAASCAmEwggJdAgEAAoGBAMjrNv34SzH78H6c
FII0l+ka3HoRxDWsWppFu6mdS22Lmo3xiWdewdEzaqUU9esVY0xoBLmEqu9NyL8x
15yykj+3kS40UguwFwvP6flESfqtENKHqXl3uWWv+5hX95pE4pUzvtRQxAXYEXsQ
lNKRgqyP2P4OYCDhGPza0qJf3HPtAgMBAAECgYBNS9ibOnPLZg6u1uM4+Hzc7D2Y
JM+kiotMwLki1uXW3hd2tk7TWuwbzLkhi3/UkiTilz4CFV0htX4euFEn8rc6Z6Ki
ad7fnU5tdqIFUaMMVsG5uxTXX/5NqatA0/cWHS2EZAnQEjtSyGEDlmoigTIa8WU5
bg7eLy2p6SLZ6ixHzQJBAOXspTo2UZO/rWZV6A6LzgyRlzAz+bKbE2WEZ6Y/hSli
e7fGMIvBNiUYwDNJfv2W5vpW96v0Kh0M9yrjx664l/sCQQDftHSLBBrkWCcoECno
HZ4Nt6uMwk4+MSOtLb61egs3GfCCYEPhKaDDNYeGJk7C/xzg777PQ58kBPG/cJhW
sNc3AkEA4pEi8A3+rR1AfYtBtLPHQ1NkLDfLYli18F9c09HcIj/NsfY2eEDYXg3Z
t7BA1xsQWLfCL0vXA/F2zmjOqDl2aQJAPgkO4JYs1vHTOfrxhBrif69VdV1U0U5T
NG0hG7ZScd9RoPYNHN2sZTXs9TieUtjoK0CQy21XLmfomkwhErlLlQJBAJJQkzdv
cM9mkQYjyzIfsqILnUScYQKc2z8GDSkcHQ4RQ/A68SD+W/AxDV3VBqykiI3JPlhA
B9TEmS/gyTI2908=
-----END PRIVATE KEY-----`

// ClientBarCert is a PEM-encoded TLS cert similar with the same
// characteristics as ClientFooCert. The subject's CommonName is 'bar'.
const ClientBarCert = `-----BEGIN CERTIFICATE-----
MIICEjCCAXugAwIBAgIQMWMfMOncft6b2Ggo6MideDANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
iQKBgQDbGSyUAOhwpsNEhXkZDHdoFr51zXgffRQGl01tkTEH2p4QrYt7fq6g8Xdv
DYimVKtZfuwgfj/bk+c7EWfeRlPGGLnEn5pXlM6xsn1N1dXdrWjuOojHj99tTxcB
/4PIAp9WfVBL541ILFIpsPimfnC49P29PfN+/3YoG60ldVUZAwIDAQABo2cwZTAO
BgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwIwDwYDVR0TAQH/BAUw
AwEB/zAdBgNVHQ4EFgQU8qmqFaR9TuSmWdoYJBBvQUy++M4wDgYDVR0RBAcwBYID
YmFyMA0GCSqGSIb3DQEBCwUAA4GBABIGrsX0JlQDsgJQw7emWcvx8TRXSoSVaOFA
yLxNeYqNOyjdcIWb/Yhpj+LDbwfHaXKuK1qXwk970Y3mTA58xsRNynV019/JGX1U
OmnqVei1wR8W4kCD5VN1xwHfG0gHqNCehM/Jg9Zd0e0GGypaMwk7ogpVJMy6SLbH
n4C1VNyQ
-----END CERTIFICATE-----`

// ClientBarKey is the private key for ClientBarCert.
const ClientBarKey = `-----BEGIN PRIVATE KEY-----
MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBANsZLJQA6HCmw0SF
eRkMd2gWvnXNeB99FAaXTW2RMQfanhCti3t+rqDxd28NiKZUq1l+7CB+P9uT5zsR
Z95GU8YYucSfmleUzrGyfU3V1d2taO46iMeP321PFwH/g8gCn1Z9UEvnjUgsUimw
+KZ+cLj0/b09837/digbrSV1VRkDAgMBAAECgYEA2ha14+D/flrQ1h0SDJf3J7oz
/aj34Egtrd3fqaezqYC4hBtrUxMnmnahDv7mvcJcCaqoOjPRNq2Dpq1NudhBO3+L
9LTb66f+YkjHeZBOL1LEAP9DMoAKduqmudbq1707AN0bPA7xMX2lCKf5vVutr8fr
su9cjB3r9I07iiggGtECQQD4ct8Kv5zRecE43no1bJfqDDxoBTs8pTGDn9jKXvIu
PIwkvIQRx8UjvJTl1WMOs2fAeEupQfSuK0ZOtlicsYxZAkEA4cHu/hrapOcfEz+z
b8hY6/ATWOpheJo2sFih0lD0C9dALwTRlAz8o7p43ktkSPGoSU1l09BF+XeL17VX
ff20uwJBAN+p0h+EBnISUR+YMZ6cx1odb9gZNY3QDXY4VdtBhHaZbXS4/ZBgLpqQ
b99EreuTGQkNgte6F8MgFChSQg22TOkCQA/iODFyrD178WjGS5aqzu7StlnEK9Vz
bDOeGMyWW4VVwLNOMHytKT1Pyl9BiK3FKuT1aBuuBK5XpeQoYx5/Mi8CQQCchVl0
ldKzqnyprbpaDNOg/MdZ04BpcNgxs2m/sGBfYwoRNYOYUxcquTsiaWgpQOxOz1FV
j5vNLq6nhm51BDtW
-----END PRIVATE KEY-----`

const LegacyR66Cert = `-----BEGIN CERTIFICATE-----
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

const LegacyR66Key = `-----BEGIN PRIVATE KEY-----
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

//nolint:gosec //this is only for testing
const RSAPk = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`

const SSHPbk = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDhbxVecyg3NbOuGgIbzUuB3" +
	"GyVIKBRWhYUOaEJtqMR8ckb3WM6cy0yplbZ6is4y8gqGhE9pQ8g3JrbYYlrb8/HnjuCnSzA9" +
	"BVhMNUxp/9Ar7GSvBO2bPIcPYBePe19AJ6MsjoT2jcZhUwlsiacHAnRWaOfYeJQP0Fw9zqhh" +
	"PcjOnWIewNQaghwBXyyzQB/BbiYAMvPo0uveYY+Yr18ExIv3ybtqgAgSVHji4Jg4JFwVd9VP" +
	"fAz3y4ucEYiOr/4bkOBTuAMxbvE+S8mvbOTQ+itsFQxuJgWTrx/53Yth3QYDwgjTaT7TLSSR" +
	"pi1+s9QQg6XTanJyjtEmmYbnaB+EhAQfI0mfOripP/1cTq9StZfYTKl58ObrYWmc5CDH338u" +
	"CdK5GxIP9eNz4RcLqPLvcVBrm62qsYReoD62InykggeOSgkOo4UGbC7JSEdW3afMBGdh797e" +
	"ht6qX3ywKbs7GNVwOt2M7xrpmCehU1uegN7GtIRvCZR0JH4+KSGitWFY3E="
