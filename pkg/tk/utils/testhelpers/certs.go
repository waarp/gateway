package testhelpers

// LocalhostCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1:6666" , "[::1]:6666" and "localhost:6666", expiring at
// Jan 29 16:00:00 2084 GMT.
const LocalhostCert = `-----BEGIN CERTIFICATE-----
MIICPDCCAaWgAwIBAgIRAMozibNPf0LHnyUC25vjrzQwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCBnzANBgkqhkiG9w0BAQEFAAOBjQAw
gYkCgYEAzAWD0DQX+nwfZcM3ZRnAAjAxCBM5SOsmMsr9rrgdXkZVrJ+e2obw3wYU
kWNtmzCE4oKLgkXz7amrc4Z5MfJ/UROGURDge/PwWRa6PgCyHQK2TA2vup1GH16n
+2uE7gOtCPHzENGIsN2bqHx9suO+NsO2+56A/AulQfNLYYEszbcCAwEAAaOBjzCB
jDAOBgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/
BAUwAwEB/zAdBgNVHQ4EFgQU3Dn86/SOlQoDldWdm3831wOsGKwwNQYDVR0RBC4w
LIIOMTI3LjAuMC4xOjY2NjaCCls6OjFdOjY2NjaCDmxvY2FsaG9zdDo2NjY2MA0G
CSqGSIb3DQEBCwUAA4GBAFFL4e0IBbdxK8ohjnZz5c5PuCXzQy14fqVCozcHGVaf
SKpWXKwjJnCpAmgzgwz60wFQuXAZNMxhCSTOxsuHrgJb+8EBNwiB8L1QNvI0TwQj
7a9xLI4RZOju8VUANmTztJajWV+29Hs4fJkHKZtPvMhOAt0SWp1D9lxB6ChxY5c3
-----END CERTIFICATE-----`

// LocalhostKey is the private key for LocalhostCert.
const LocalhostKey = `-----BEGIN PRIVATE KEY-----
MIICdwIBADANBgkqhkiG9w0BAQEFAASCAmEwggJdAgEAAoGBAMwFg9A0F/p8H2XD
N2UZwAIwMQgTOUjrJjLK/a64HV5GVayfntqG8N8GFJFjbZswhOKCi4JF8+2pq3OG
eTHyf1EThlEQ4Hvz8FkWuj4Ash0CtkwNr7qdRh9ep/trhO4DrQjx8xDRiLDdm6h8
fbLjvjbDtvuegPwLpUHzS2GBLM23AgMBAAECgYEAjfGYT5auyBrrTUWQmMpdiCg3
NMMLK+xOWzBXZuO5qwmMOdmkD62qj8APN0fRzhLnoR/qJ+y7VTKiknGQiGuKn4Ln
eF49I5cIAIiYw/zHWWb9Gp6p2zDC6cC+c+Sq0iJIa49b5nh3eIkHqJZh9BqCQx5q
DVCvhTz8h5CniZ/QqaECQQD2f8qMp+hX7MM9tu2gtoV8rm0PsNUXY7Drx+Vu27/F
+22xIT6t9TXqfEZ9bTkQ9E69ym75NLhTINZ7w4s2BIjNAkEA0+KYuG16RpXswdnx
XlMZUXy2KiVKHuE6H435t5ZBUv3/pGbjNH+k5g0Gg1QPkkxVeuyYiHq99RltH0Z4
n9tAkwJAOBN4Q6lK/P2aqOZ9hosfMO8JVoF26JxAOlM+SYrqRKLfIGWcubxH6LEe
5Be93LKHWzu7JSwuJpMY2AzzFXXQnQJBAKetH3x7rpMjXBxAM8GYc2XIEoShw9lS
FWQZP6/oKUPbG65neY/3H3CqiCfvou78l3zStRb0Q1UuTOu+IgEnSh8CQEphmrqd
rxNpXCpw8JZOywYZjs9v7FDM8q7WA4436KMwXwQmoQjniyrn6S+O/86cMzYRZG32
6tl3qHj2eUyRuvw=
-----END PRIVATE KEY-----`

// ClientCert is a PEM-encoded TLS cert to be used for testing client TLS
// authentication. The subject's CommonName is 'foo'.
const ClientCert = `-----BEGIN CERTIFICATE-----
MIICTzCCAbigAwIBAgIRAMumaDCfbtdI43RqdlPVbcEwDQYJKoZIhvcNAQELBQAw
IDEQMA4GA1UEChMHQWNtZSBDbzEMMAoGA1UEAxMDZm9vMCAXDTcwMDEwMTAwMDAw
MFoYDzIwODQwMTI5MTYwMDAwWjAgMRAwDgYDVQQKEwdBY21lIENvMQwwCgYDVQQD
EwNmb28wgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAJ9KsdR0BZMhvf+/UoT2
QbCUk/f6WqdIsQJRm5AUckmuRbqi1xb5vIHilOIx3v4dgASyktR3YnXEyXaiHIRM
RKYXDbNSAiEbGxk/I+OzeFj2R9zHy1N4/hRJV8SPkZKPWUeCzmUYQCWeSOFNR8Tq
hPm68FgQ6KDvU5Qnfrh0vPQpAgMBAAGjgYYwgYMwDgYDVR0PAQH/BAQDAgKkMBMG
A1UdJQQMMAoGCCsGAQUFBwMCMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGv8
IEMlc8HTSMHQ+r0CNEZnkpVXMCwGA1UdEQQlMCOCCWxvY2FsaG9zdIcEfwAAAYcQ
AAAAAAAAAAAAAAAAAAAAATANBgkqhkiG9w0BAQsFAAOBgQBwKRCJOvTrxKsorX5p
M5pJPyo7WTb9cFSqHfOM9KRgMMJHFsPftMDVGMgpK0kOh5IxZ2uNUZn+v2Qghtqc
xX48yPZ4kIFJ4K5mjy5vj6cjKy7nuSd725twmB2RjJLQSI3CMdABD8ZwjNEfX+UY
+zP6VsNAgKNiOXHRo4jp6ZDA+Q==
-----END CERTIFICATE-----`

// ClientKey is the private key for ClientCert.
const ClientKey = `-----BEGIN PRIVATE KEY-----
MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBAJ9KsdR0BZMhvf+/
UoT2QbCUk/f6WqdIsQJRm5AUckmuRbqi1xb5vIHilOIx3v4dgASyktR3YnXEyXai
HIRMRKYXDbNSAiEbGxk/I+OzeFj2R9zHy1N4/hRJV8SPkZKPWUeCzmUYQCWeSOFN
R8TqhPm68FgQ6KDvU5Qnfrh0vPQpAgMBAAECgYBDkJU5PjUXIVrL3cUMrL9UPNE+
f6xwBD9Acoj/ZgzL/+WHsoZ1Mlyo4wivoOOq+axRUcVB5ZmXxm6FqWFbJFaBSROh
GM1Li2jsNiFkA7oshYfSRyMhFMd47C/TGu/YHXrHd7I4DK1N4eMOQ2xdPYJZXMKU
U19cpjEA8ES7xKWU4QJBAMGSUztDBOHj5jp2un/pJC4UtI3gyWoNZf06tBZBVP7r
t00D1FBbpJmziCHdhtDRoMMMTYXd/96SrBkYYoCTDTUCQQDSqigvOKIHL7mObsad
ZBFKyNQQF89nuyi/S+jLEJg1UkbTecixARWgZWKKTUOXHdX3D/8RnFHsxE2yXXUp
Gs2lAkBM6M2HE0bCVaFpAzlwjvpgELv4TyLXr0Ehjwx0dzrFGnS29dmKoA7TPuDM
y86/8zpTpPS8RoteLJqSUfz5JvQtAkBgIlxSKELwipvf3rduTaMCgKEdcvAoAyW2
Hlruh/UdqB1AFjw6YidPWdTdDiNBC9F/fGJG1BIivPZD5hg4GM4tAkBF8muIUIjL
YS3DRwQOzRHIzW4ZZltc7Mr36GpeiAfOkPy3ZfP1XMLudoqTBjtbqBOkq0bmAB2m
cUqCCsYA43xI
-----END PRIVATE KEY-----`
