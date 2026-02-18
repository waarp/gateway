package tasks

import (
	"archive/zip"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/authtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
)

func TestUpdateConf(t *testing.T) {
	// Setup test
	db := dbtest.TestDatabase(t)
	logger := logtest.GetTestLogger(t)
	root := fstest.TestRoot(t)
	t.Setenv(conf.ConfigDirEnvVar, root.Name())

	// Setup data
	const (
		zipName   = "updateconf.zip"
		fwContent = `{"foo":"bar"}`
		grContent = "hello world"
	)
	setupUpdateConfZip(t, root, zipName, fwContent, grContent)

	// Setup task
	task := &updateconfTask{}
	params := map[string]string{
		"zipFile": fs.JoinPath(root.Name(), zipName),
	}
	transCtx := &model.TransferContext{Transfer: &model.Transfer{}}

	// Run task
	require.NoError(t, task.Run(t.Context(), params, db, logger, transCtx, nil))

	// Check conf was imported
	var servers model.LocalAgents
	require.NoError(t, db.Select(&servers).Owner().Run())
	assert.NotEmpty(t, servers)

	// Check FW file was moved
	cont, err := root.ReadFile(updateconfFwFilename)
	require.NoError(t, err)
	assert.Equal(t, fwContent, string(cont))

	// Check get-remote file was moved
	cont, err = root.ReadFile(updateconfGetRemoteFilename)
	require.NoError(t, err)
	assert.Equal(t, grContent, string(cont))
}

func setupUpdateConfZip(tb testing.TB, root *os.Root, archName, fwContent, grContent string) {
	tb.Helper()

	confJson := getConfJson(tb)

	// Create archive
	archFile, err := root.Create(archName)
	require.NoError(tb, err)
	defer func() { require.NoError(tb, archFile.Close()) }()

	arch := zip.NewWriter(archFile)
	defer func() { require.NoError(tb, arch.Close()) }()

	// Add config file
	confFilename := conf.GlobalConfig.GatewayName + ".json"
	writeToZip(tb, arch, confFilename, []byte(confJson))

	// Add filewatcher file
	writeToZip(tb, arch, updateconfFwFilename, []byte(fwContent))

	// Add get-remote file
	writeToZip(tb, arch, updateconfGetRemoteFilename, []byte(grContent))
}

func writeToZip(tb testing.TB, arch *zip.Writer, fileName string, data []byte) {
	tb.Helper()

	file, err := arch.Create(fileName)
	require.NoError(tb, err)
	_, err = file.Write(data)
	require.NoError(tb, err)
}

func getConfJson(tb testing.TB) string {
	tb.Helper()
	const confJson = `
{
    "locals": [
        {
            "name": "wg-r66-tls-server",
            "protocol": "r66-tls",
            "address": ":10066",
            "accounts": [
                {
                    "login": "waarp",
                    "password": "sesame"
                }
            ],
            "credentials": [
                {
                    "name": "server-password",
                    "type": "password",
                    "value": "sesame"
                },
                {
                    "name": "r66-cert",
                    "type": "tls_certificate",
                    "value": "-----BEGIN CERTIFICATE-----\nMIIDNzCCAh+gAwIBAgIQVU94EFw5Hw4XQqMFZZy61jANBgkqhkiG9w0BAQsFADAS\nMRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw\nMDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A\nMIIBCgKCAQEAt8c/7Sd6PG7Dk3Xln9IIXJS/5BopJLWRGpiefjlnxQisTm+BELjM\nxg4H8mIL5wzaSsYTreAdBiry8sQlmk2e0m6QanifusKDVOcd3ChRkSXHoVatpkg7\nHAdfjYL+20fWJ9bjCoHmAgN96rh0SeHRiqj+eHUsIT+0exzH4x1rLoqm5k/5ihur\nW+PbdGQDLDcu4JqwK+ru7m+mqoyvfnt7b23b6/d+Vs5rpw6BHQlydf68ilzPBlDn\nwvMzcgyFOQIqellTlEtbfwWe8okis5PCWdvOBVFEYfO2YW1W2/UV7MpudQYvSezx\nL5QRwvfOBsxHfnO/0B+lHOZ7U3HL+3RQOwIDAQABo4GGMIGDMA4GA1UdDwEB/wQE\nAwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud\nDgQWBBS3C8GPVx+7k+/UJ8JrJviJh8s2GTAsBgNVHREEJTAjgglsb2NhbGhvc3SH\nBH8AAAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQELBQADggEBAKe4BlaY\ndfb6XTOmRc764SaHozqR3Jsa6qbvYZEIpEPowkKfg5Wa+A3qVRvbwo5vPuJ4BFWD\nndciuwoi0KWx1SfrvgairIki1qSTChOZGy4+VpoBTvSbdyQUI1rtafC47M984QWp\nwbec6zteONJJmB6THRCAkHuiFmO6gSkEAnpBuOH8CdoJbaD/7Y0DoukpLM5lfb/U\nDOd8t+oxMTHLEkgpD4J7mT/IEGAUg3J/7o0/kjRMBDP9EdBStgrWCdrX6wIe0Dd4\nw8jMX2C4WYi1g07fT3TujXTIlZi153Pj9YoLKYq7kjn3BpJ19pgEqLXpEZ4iOY9u\nKT1x3EqvZ6VFx1I=\n-----END CERTIFICATE-----\n",
                    "value2": "-----BEGIN PRIVATE KEY-----\nMIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC3xz/tJ3o8bsOT\ndeWf0ghclL/kGikktZEamJ5+OWfFCKxOb4EQuMzGDgfyYgvnDNpKxhOt4B0GKvLy\nxCWaTZ7SbpBqeJ+6woNU5x3cKFGRJcehVq2mSDscB1+Ngv7bR9Yn1uMKgeYCA33q\nuHRJ4dGKqP54dSwhP7R7HMfjHWsuiqbmT/mKG6tb49t0ZAMsNy7gmrAr6u7ub6aq\njK9+e3tvbdvr935WzmunDoEdCXJ1/ryKXM8GUOfC8zNyDIU5Aip6WVOUS1t/BZ7y\niSKzk8JZ284FUURh87ZhbVbb9RXsym51Bi9J7PEvlBHC984GzEd+c7/QH6Uc5ntT\nccv7dFA7AgMBAAECggEBAJKO64QM/4ZCLuXiF4Uk0lZCqeUWl8kWouk63Op8jSys\nhfznH15ega9QcTXyytsvfMY0wGzhVUQd7DF+Cx7K1+WpFrJSD8+4X1POHBn+bU37\newBHR6Rb5gesOZ944BWvbDHJRLaUcQEaF8if4N0qoRibPJSDnPXG//9OLHoKc/dg\ncUhEJ9s78TFZiBxzOErp/jMcAZPnFhe1Oqrfy0CzLeB9iwDbJpBYsah/9gniGnDS\nhaXOTE8CjA7QnRVPox2NNUpSbln0d/yYFqS75398EpuueKUNLgfQBlKWbDsUExhY\n3WkqgVL8ezXL65/3zUESu+5bnMt2ZrzG0W/DYXOjZMECgYEA1U7Y8/uE93Tnq1UW\n/JKTSSwnOlgwC2hNqRmTfh3DsSEwr0RL/n0UZnvYz60yWZ8h2YU0zKtNJSMx+OUN\nl1x/m3ZrmxChE7Bhog4KveqFcuVhWrsBxmcQ/CSHSLoFrlZ7Vu0Dv/Q2XsZMOr0C\n22FKMrCZ+EQQtRkIR1wbXq6yA2UCgYEA3I9nvRsM+4wFlduwdK6N3pZYekFoUJVm\nH2wWbz/jVR9bLG18/SYueF5q4A8NCNZUQPxb/4wetHip7u69rtNMvihzdUoWbGbC\nTT0ehlnUWDHRdeYBf13CuNB84vtV9mWzBr/3kXpSxlw8bhCjQmKUPSj1qgt5eP4e\noguwyH1dWx8CgYEAlS625zRyk0rMt/QjxnOQ1O0vZkvFFkVVgz2i/OI+OgSXcwzW\nBV9fRCm7wctE2o9D8kiKW9Y2dxG9YnB35/NGP+k7atDfhtCmB9vAQYDi4i9wvi7q\nF+N/aoj1oLSRQpOzYWEUbUyUNgaDy9TjSaEqbnc6x/p6oN2n/5h4f+i4EU0CgYBR\n9Hd3rTWFwuHQbXGD6diNfRAjXWqFhv4Lbv5nGDZAywX13Dk1V5qs32iXGQCe6AUm\noJ8OteIy7SM3xT52V82MWzuLuZvba9OHH87X3Ukp/Fj4lh64VP6l7dJ6BSpMBD4h\no/M5+1oGmv9ZZpVDdZ3fm/is1tasPsDjNbTayrqFwQKBgQCvCFJqVARZ47yXJDsO\ng++YLf0IeP+U103C2cgFzNE7p36oYlaiSsBX4Tj6/6bJI+uQsG2g4vm8iUMApkKG\nc8Q1TMzoFx8JafdVx6qwMPuDd9iXic0x12UzJJcjFr7XT/cUA1xTmg+FKRd17ov5\nnGic5g65y2duPAfKX3cngJyfUA==\n-----END PRIVATE KEY-----\n"
                }
            ]
        },
        {
            "name": "wg-sftp-server",
            "protocol": "sftp",
            "address": ":10022",
            "accounts": [
                {
                    "login": "waarp",
                    "password": "sesame"
                }
            ],
            "credentials": [
                {
                    "name": "sftp-hostkey",
                    "type": "ssh_private_key",
                    "value": "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAlwAAAAdzc2gtcn\nNhAAAAAwEAAQAAAIEAwg0KXLz4S0UF4QRgDp+WTWzCewS+sLUtZ0aLQSAN2HWdCssm/Lu0\nk2gLxH3XJjYJd400tc/CMcWW+IUVrhMhfLNR1pjVixqGVPSxXjGpNgfwzxyZ5QC0o6aNbB\nTZl1U8v/qi7+xFbEX7gNTMnAgQy79BzCoqH+CIXUHjHl5CEi8AAAIItGeBDLRngQwAAAAH\nc3NoLXJzYQAAAIEAwg0KXLz4S0UF4QRgDp+WTWzCewS+sLUtZ0aLQSAN2HWdCssm/Lu0k2\ngLxH3XJjYJd400tc/CMcWW+IUVrhMhfLNR1pjVixqGVPSxXjGpNgfwzxyZ5QC0o6aNbBTZ\nl1U8v/qi7+xFbEX7gNTMnAgQy79BzCoqH+CIXUHjHl5CEi8AAAADAQABAAAAgHafwCjnAC\nYLQMfIg+wMlLYp+U14nhWp++J5VmFudehQJbtWazPujznZTfBFOUPLnsftkz5djgodDuhH\nevRaD4p6FvjZ8Rpr0zH5Ieg8hSRpjJZezGuOsAEswpLFau5bKkLj5LkQrFNv1TndcrUadi\nHt6lJPopALaJ5918BRSeqZAAAAQHT/b5HbiwGx/7uTisLQ19YK9GwqH9kW1a6kvzyG2Yct\nEoGd6VmkZ5cxr0AIY2XyoxhfXFPyk0bqilC+K0e316AAAABBAN/OkGhlhlO1zvkiDIsdzh\n39YrjB7SJD0H6qViru04IndUnwLet2d9B7/tAnoqMEkSWGLZVMvE/cl2i0sG7genMAAABB\nAN32v0DkVrcZBl39vbbk2+syaNWxx5rjnVDSlVkpTf/5lAhtnNLwX6LoOMDTCsfY8nodJ2\nwZCoCb3M+TY690blUAAAAMcGFvbG9AcXVhc2FyAQIDBAUGBw==\n-----END OPENSSH PRIVATE KEY-----\n"
                }
            ]
        },
        {
            "name": "wg-https-server",
            "protocol": "https",
            "address": ":10080",
            "accounts": [
                {
                    "login": "waarp",
                    "password": "sesame"
                }
            ],
            "credentials": [
                {
                    "name": "https-cert",
                    "type": "tls_certificate",
                    "value": "-----BEGIN CERTIFICATE-----\nMIIDNzCCAh+gAwIBAgIQVU94EFw5Hw4XQqMFZZy61jANBgkqhkiG9w0BAQsFADAS\nMRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw\nMDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A\nMIIBCgKCAQEAt8c/7Sd6PG7Dk3Xln9IIXJS/5BopJLWRGpiefjlnxQisTm+BELjM\nxg4H8mIL5wzaSsYTreAdBiry8sQlmk2e0m6QanifusKDVOcd3ChRkSXHoVatpkg7\nHAdfjYL+20fWJ9bjCoHmAgN96rh0SeHRiqj+eHUsIT+0exzH4x1rLoqm5k/5ihur\nW+PbdGQDLDcu4JqwK+ru7m+mqoyvfnt7b23b6/d+Vs5rpw6BHQlydf68ilzPBlDn\nwvMzcgyFOQIqellTlEtbfwWe8okis5PCWdvOBVFEYfO2YW1W2/UV7MpudQYvSezx\nL5QRwvfOBsxHfnO/0B+lHOZ7U3HL+3RQOwIDAQABo4GGMIGDMA4GA1UdDwEB/wQE\nAwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud\nDgQWBBS3C8GPVx+7k+/UJ8JrJviJh8s2GTAsBgNVHREEJTAjgglsb2NhbGhvc3SH\nBH8AAAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQELBQADggEBAKe4BlaY\ndfb6XTOmRc764SaHozqR3Jsa6qbvYZEIpEPowkKfg5Wa+A3qVRvbwo5vPuJ4BFWD\nndciuwoi0KWx1SfrvgairIki1qSTChOZGy4+VpoBTvSbdyQUI1rtafC47M984QWp\nwbec6zteONJJmB6THRCAkHuiFmO6gSkEAnpBuOH8CdoJbaD/7Y0DoukpLM5lfb/U\nDOd8t+oxMTHLEkgpD4J7mT/IEGAUg3J/7o0/kjRMBDP9EdBStgrWCdrX6wIe0Dd4\nw8jMX2C4WYi1g07fT3TujXTIlZi153Pj9YoLKYq7kjn3BpJ19pgEqLXpEZ4iOY9u\nKT1x3EqvZ6VFx1I=\n-----END CERTIFICATE-----\n",
                    "value2": "-----BEGIN PRIVATE KEY-----\nMIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC3xz/tJ3o8bsOT\ndeWf0ghclL/kGikktZEamJ5+OWfFCKxOb4EQuMzGDgfyYgvnDNpKxhOt4B0GKvLy\nxCWaTZ7SbpBqeJ+6woNU5x3cKFGRJcehVq2mSDscB1+Ngv7bR9Yn1uMKgeYCA33q\nuHRJ4dGKqP54dSwhP7R7HMfjHWsuiqbmT/mKG6tb49t0ZAMsNy7gmrAr6u7ub6aq\njK9+e3tvbdvr935WzmunDoEdCXJ1/ryKXM8GUOfC8zNyDIU5Aip6WVOUS1t/BZ7y\niSKzk8JZ284FUURh87ZhbVbb9RXsym51Bi9J7PEvlBHC984GzEd+c7/QH6Uc5ntT\nccv7dFA7AgMBAAECggEBAJKO64QM/4ZCLuXiF4Uk0lZCqeUWl8kWouk63Op8jSys\nhfznH15ega9QcTXyytsvfMY0wGzhVUQd7DF+Cx7K1+WpFrJSD8+4X1POHBn+bU37\newBHR6Rb5gesOZ944BWvbDHJRLaUcQEaF8if4N0qoRibPJSDnPXG//9OLHoKc/dg\ncUhEJ9s78TFZiBxzOErp/jMcAZPnFhe1Oqrfy0CzLeB9iwDbJpBYsah/9gniGnDS\nhaXOTE8CjA7QnRVPox2NNUpSbln0d/yYFqS75398EpuueKUNLgfQBlKWbDsUExhY\n3WkqgVL8ezXL65/3zUESu+5bnMt2ZrzG0W/DYXOjZMECgYEA1U7Y8/uE93Tnq1UW\n/JKTSSwnOlgwC2hNqRmTfh3DsSEwr0RL/n0UZnvYz60yWZ8h2YU0zKtNJSMx+OUN\nl1x/m3ZrmxChE7Bhog4KveqFcuVhWrsBxmcQ/CSHSLoFrlZ7Vu0Dv/Q2XsZMOr0C\n22FKMrCZ+EQQtRkIR1wbXq6yA2UCgYEA3I9nvRsM+4wFlduwdK6N3pZYekFoUJVm\nH2wWbz/jVR9bLG18/SYueF5q4A8NCNZUQPxb/4wetHip7u69rtNMvihzdUoWbGbC\nTT0ehlnUWDHRdeYBf13CuNB84vtV9mWzBr/3kXpSxlw8bhCjQmKUPSj1qgt5eP4e\noguwyH1dWx8CgYEAlS625zRyk0rMt/QjxnOQ1O0vZkvFFkVVgz2i/OI+OgSXcwzW\nBV9fRCm7wctE2o9D8kiKW9Y2dxG9YnB35/NGP+k7atDfhtCmB9vAQYDi4i9wvi7q\nF+N/aoj1oLSRQpOzYWEUbUyUNgaDy9TjSaEqbnc6x/p6oN2n/5h4f+i4EU0CgYBR\n9Hd3rTWFwuHQbXGD6diNfRAjXWqFhv4Lbv5nGDZAywX13Dk1V5qs32iXGQCe6AUm\noJ8OteIy7SM3xT52V82MWzuLuZvba9OHH87X3Ukp/Fj4lh64VP6l7dJ6BSpMBD4h\no/M5+1oGmv9ZZpVDdZ3fm/is1tasPsDjNbTayrqFwQKBgQCvCFJqVARZ47yXJDsO\ng++YLf0IeP+U103C2cgFzNE7p36oYlaiSsBX4Tj6/6bJI+uQsG2g4vm8iUMApkKG\nc8Q1TMzoFx8JafdVx6qwMPuDd9iXic0x12UzJJcjFr7XT/cUA1xTmg+FKRd17ov5\nnGic5g65y2duPAfKX3cngJyfUA==\n-----END PRIVATE KEY-----\n"
                }
            ]
        },
        {
            "name": "wg-ftps-server",
            "protocol": "ftps",
            "address": ":10021",
            "configuration": {
                "tlsRequirement": "Optional",
                "passiveModeMinPort": 0,
                "passiveModeMaxPort": 0
            },
            "accounts": [
                {
                    "login": "waarp",
                    "password": "sesame"
                }
            ],
            "credentials": [
                {
                    "name": "ftps-cert",
                    "type": "tls_certificate",
                    "value": "-----BEGIN CERTIFICATE-----\nMIIDNzCCAh+gAwIBAgIQVU94EFw5Hw4XQqMFZZy61jANBgkqhkiG9w0BAQsFADAS\nMRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw\nMDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A\nMIIBCgKCAQEAt8c/7Sd6PG7Dk3Xln9IIXJS/5BopJLWRGpiefjlnxQisTm+BELjM\nxg4H8mIL5wzaSsYTreAdBiry8sQlmk2e0m6QanifusKDVOcd3ChRkSXHoVatpkg7\nHAdfjYL+20fWJ9bjCoHmAgN96rh0SeHRiqj+eHUsIT+0exzH4x1rLoqm5k/5ihur\nW+PbdGQDLDcu4JqwK+ru7m+mqoyvfnt7b23b6/d+Vs5rpw6BHQlydf68ilzPBlDn\nwvMzcgyFOQIqellTlEtbfwWe8okis5PCWdvOBVFEYfO2YW1W2/UV7MpudQYvSezx\nL5QRwvfOBsxHfnO/0B+lHOZ7U3HL+3RQOwIDAQABo4GGMIGDMA4GA1UdDwEB/wQE\nAwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud\nDgQWBBS3C8GPVx+7k+/UJ8JrJviJh8s2GTAsBgNVHREEJTAjgglsb2NhbGhvc3SH\nBH8AAAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQELBQADggEBAKe4BlaY\ndfb6XTOmRc764SaHozqR3Jsa6qbvYZEIpEPowkKfg5Wa+A3qVRvbwo5vPuJ4BFWD\nndciuwoi0KWx1SfrvgairIki1qSTChOZGy4+VpoBTvSbdyQUI1rtafC47M984QWp\nwbec6zteONJJmB6THRCAkHuiFmO6gSkEAnpBuOH8CdoJbaD/7Y0DoukpLM5lfb/U\nDOd8t+oxMTHLEkgpD4J7mT/IEGAUg3J/7o0/kjRMBDP9EdBStgrWCdrX6wIe0Dd4\nw8jMX2C4WYi1g07fT3TujXTIlZi153Pj9YoLKYq7kjn3BpJ19pgEqLXpEZ4iOY9u\nKT1x3EqvZ6VFx1I=\n-----END CERTIFICATE-----\n",
                    "value2": "-----BEGIN PRIVATE KEY-----\nMIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC3xz/tJ3o8bsOT\ndeWf0ghclL/kGikktZEamJ5+OWfFCKxOb4EQuMzGDgfyYgvnDNpKxhOt4B0GKvLy\nxCWaTZ7SbpBqeJ+6woNU5x3cKFGRJcehVq2mSDscB1+Ngv7bR9Yn1uMKgeYCA33q\nuHRJ4dGKqP54dSwhP7R7HMfjHWsuiqbmT/mKG6tb49t0ZAMsNy7gmrAr6u7ub6aq\njK9+e3tvbdvr935WzmunDoEdCXJ1/ryKXM8GUOfC8zNyDIU5Aip6WVOUS1t/BZ7y\niSKzk8JZ284FUURh87ZhbVbb9RXsym51Bi9J7PEvlBHC984GzEd+c7/QH6Uc5ntT\nccv7dFA7AgMBAAECggEBAJKO64QM/4ZCLuXiF4Uk0lZCqeUWl8kWouk63Op8jSys\nhfznH15ega9QcTXyytsvfMY0wGzhVUQd7DF+Cx7K1+WpFrJSD8+4X1POHBn+bU37\newBHR6Rb5gesOZ944BWvbDHJRLaUcQEaF8if4N0qoRibPJSDnPXG//9OLHoKc/dg\ncUhEJ9s78TFZiBxzOErp/jMcAZPnFhe1Oqrfy0CzLeB9iwDbJpBYsah/9gniGnDS\nhaXOTE8CjA7QnRVPox2NNUpSbln0d/yYFqS75398EpuueKUNLgfQBlKWbDsUExhY\n3WkqgVL8ezXL65/3zUESu+5bnMt2ZrzG0W/DYXOjZMECgYEA1U7Y8/uE93Tnq1UW\n/JKTSSwnOlgwC2hNqRmTfh3DsSEwr0RL/n0UZnvYz60yWZ8h2YU0zKtNJSMx+OUN\nl1x/m3ZrmxChE7Bhog4KveqFcuVhWrsBxmcQ/CSHSLoFrlZ7Vu0Dv/Q2XsZMOr0C\n22FKMrCZ+EQQtRkIR1wbXq6yA2UCgYEA3I9nvRsM+4wFlduwdK6N3pZYekFoUJVm\nH2wWbz/jVR9bLG18/SYueF5q4A8NCNZUQPxb/4wetHip7u69rtNMvihzdUoWbGbC\nTT0ehlnUWDHRdeYBf13CuNB84vtV9mWzBr/3kXpSxlw8bhCjQmKUPSj1qgt5eP4e\noguwyH1dWx8CgYEAlS625zRyk0rMt/QjxnOQ1O0vZkvFFkVVgz2i/OI+OgSXcwzW\nBV9fRCm7wctE2o9D8kiKW9Y2dxG9YnB35/NGP+k7atDfhtCmB9vAQYDi4i9wvi7q\nF+N/aoj1oLSRQpOzYWEUbUyUNgaDy9TjSaEqbnc6x/p6oN2n/5h4f+i4EU0CgYBR\n9Hd3rTWFwuHQbXGD6diNfRAjXWqFhv4Lbv5nGDZAywX13Dk1V5qs32iXGQCe6AUm\noJ8OteIy7SM3xT52V82MWzuLuZvba9OHH87X3Ukp/Fj4lh64VP6l7dJ6BSpMBD4h\no/M5+1oGmv9ZZpVDdZ3fm/is1tasPsDjNbTayrqFwQKBgQCvCFJqVARZ47yXJDsO\ng++YLf0IeP+U103C2cgFzNE7p36oYlaiSsBX4Tj6/6bJI+uQsG2g4vm8iUMApkKG\nc8Q1TMzoFx8JafdVx6qwMPuDd9iXic0x12UzJJcjFr7XT/cUA1xTmg+FKRd17ov5\nnGic5g65y2duPAfKX3cngJyfUA==\n-----END PRIVATE KEY-----\n"
                }
            ]
        },
        {
            "name": "wg-pesit-tls-server",
            "protocol": "pesit-tls",
            "address": ":10010",
            "accounts": [
                {
                    "login": "waarp",
                    "password": "sesame"
                }
            ],
            "credentials": [
                {
                    "name": "pesit-cert",
                    "type": "tls_certificate",
                    "value": "-----BEGIN CERTIFICATE-----\nMIIDNzCCAh+gAwIBAgIQVU94EFw5Hw4XQqMFZZy61jANBgkqhkiG9w0BAQsFADAS\nMRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw\nMDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A\nMIIBCgKCAQEAt8c/7Sd6PG7Dk3Xln9IIXJS/5BopJLWRGpiefjlnxQisTm+BELjM\nxg4H8mIL5wzaSsYTreAdBiry8sQlmk2e0m6QanifusKDVOcd3ChRkSXHoVatpkg7\nHAdfjYL+20fWJ9bjCoHmAgN96rh0SeHRiqj+eHUsIT+0exzH4x1rLoqm5k/5ihur\nW+PbdGQDLDcu4JqwK+ru7m+mqoyvfnt7b23b6/d+Vs5rpw6BHQlydf68ilzPBlDn\nwvMzcgyFOQIqellTlEtbfwWe8okis5PCWdvOBVFEYfO2YW1W2/UV7MpudQYvSezx\nL5QRwvfOBsxHfnO/0B+lHOZ7U3HL+3RQOwIDAQABo4GGMIGDMA4GA1UdDwEB/wQE\nAwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud\nDgQWBBS3C8GPVx+7k+/UJ8JrJviJh8s2GTAsBgNVHREEJTAjgglsb2NhbGhvc3SH\nBH8AAAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQELBQADggEBAKe4BlaY\ndfb6XTOmRc764SaHozqR3Jsa6qbvYZEIpEPowkKfg5Wa+A3qVRvbwo5vPuJ4BFWD\nndciuwoi0KWx1SfrvgairIki1qSTChOZGy4+VpoBTvSbdyQUI1rtafC47M984QWp\nwbec6zteONJJmB6THRCAkHuiFmO6gSkEAnpBuOH8CdoJbaD/7Y0DoukpLM5lfb/U\nDOd8t+oxMTHLEkgpD4J7mT/IEGAUg3J/7o0/kjRMBDP9EdBStgrWCdrX6wIe0Dd4\nw8jMX2C4WYi1g07fT3TujXTIlZi153Pj9YoLKYq7kjn3BpJ19pgEqLXpEZ4iOY9u\nKT1x3EqvZ6VFx1I=\n-----END CERTIFICATE-----\n",
                    "value2": "-----BEGIN PRIVATE KEY-----\nMIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC3xz/tJ3o8bsOT\ndeWf0ghclL/kGikktZEamJ5+OWfFCKxOb4EQuMzGDgfyYgvnDNpKxhOt4B0GKvLy\nxCWaTZ7SbpBqeJ+6woNU5x3cKFGRJcehVq2mSDscB1+Ngv7bR9Yn1uMKgeYCA33q\nuHRJ4dGKqP54dSwhP7R7HMfjHWsuiqbmT/mKG6tb49t0ZAMsNy7gmrAr6u7ub6aq\njK9+e3tvbdvr935WzmunDoEdCXJ1/ryKXM8GUOfC8zNyDIU5Aip6WVOUS1t/BZ7y\niSKzk8JZ284FUURh87ZhbVbb9RXsym51Bi9J7PEvlBHC984GzEd+c7/QH6Uc5ntT\nccv7dFA7AgMBAAECggEBAJKO64QM/4ZCLuXiF4Uk0lZCqeUWl8kWouk63Op8jSys\nhfznH15ega9QcTXyytsvfMY0wGzhVUQd7DF+Cx7K1+WpFrJSD8+4X1POHBn+bU37\newBHR6Rb5gesOZ944BWvbDHJRLaUcQEaF8if4N0qoRibPJSDnPXG//9OLHoKc/dg\ncUhEJ9s78TFZiBxzOErp/jMcAZPnFhe1Oqrfy0CzLeB9iwDbJpBYsah/9gniGnDS\nhaXOTE8CjA7QnRVPox2NNUpSbln0d/yYFqS75398EpuueKUNLgfQBlKWbDsUExhY\n3WkqgVL8ezXL65/3zUESu+5bnMt2ZrzG0W/DYXOjZMECgYEA1U7Y8/uE93Tnq1UW\n/JKTSSwnOlgwC2hNqRmTfh3DsSEwr0RL/n0UZnvYz60yWZ8h2YU0zKtNJSMx+OUN\nl1x/m3ZrmxChE7Bhog4KveqFcuVhWrsBxmcQ/CSHSLoFrlZ7Vu0Dv/Q2XsZMOr0C\n22FKMrCZ+EQQtRkIR1wbXq6yA2UCgYEA3I9nvRsM+4wFlduwdK6N3pZYekFoUJVm\nH2wWbz/jVR9bLG18/SYueF5q4A8NCNZUQPxb/4wetHip7u69rtNMvihzdUoWbGbC\nTT0ehlnUWDHRdeYBf13CuNB84vtV9mWzBr/3kXpSxlw8bhCjQmKUPSj1qgt5eP4e\noguwyH1dWx8CgYEAlS625zRyk0rMt/QjxnOQ1O0vZkvFFkVVgz2i/OI+OgSXcwzW\nBV9fRCm7wctE2o9D8kiKW9Y2dxG9YnB35/NGP+k7atDfhtCmB9vAQYDi4i9wvi7q\nF+N/aoj1oLSRQpOzYWEUbUyUNgaDy9TjSaEqbnc6x/p6oN2n/5h4f+i4EU0CgYBR\n9Hd3rTWFwuHQbXGD6diNfRAjXWqFhv4Lbv5nGDZAywX13Dk1V5qs32iXGQCe6AUm\noJ8OteIy7SM3xT52V82MWzuLuZvba9OHH87X3Ukp/Fj4lh64VP6l7dJ6BSpMBD4h\no/M5+1oGmv9ZZpVDdZ3fm/is1tasPsDjNbTayrqFwQKBgQCvCFJqVARZ47yXJDsO\ng++YLf0IeP+U103C2cgFzNE7p36oYlaiSsBX4Tj6/6bJI+uQsG2g4vm8iUMApkKG\nc8Q1TMzoFx8JafdVx6qwMPuDd9iXic0x12UzJJcjFr7XT/cUA1xTmg+FKRd17ov5\nnGic5g65y2duPAfKX3cngJyfUA==\n-----END PRIVATE KEY-----\n"
                }
            ]
        }
    ],
    "clients": [
        {
            "name": "wg-r66-tls-client",
            "protocol": "r66-tls"
        },
        {
            "name": "wg-sftp-client",
            "protocol": "sftp"
        },
        {
            "name": "wg-https-client",
            "protocol": "https"
        },
        {
            "name": "wg-ftps-client",
            "protocol": "ftps"
        },
        {
            "name": "wg-pesit-client",
            "protocol": "pesit-tls"
        }
    ],
    "remotes": [
        {
            "name": "wt-r66-tls-server",
            "protocol": "r66-tls",
            "address": "127.0.0.1:11066",
            "accounts": [
                {
                    "login": "waarp",
                    "password": "sesame"
                }
            ],
            "credentials": [
                {
                    "name": "wt-cert",
                    "type": "trusted_tls_certificate",
                    "value": "-----BEGIN CERTIFICATE-----\nMIIDNzCCAh+gAwIBAgIQVU94EFw5Hw4XQqMFZZy61jANBgkqhkiG9w0BAQsFADAS\nMRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw\nMDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A\nMIIBCgKCAQEAt8c/7Sd6PG7Dk3Xln9IIXJS/5BopJLWRGpiefjlnxQisTm+BELjM\nxg4H8mIL5wzaSsYTreAdBiry8sQlmk2e0m6QanifusKDVOcd3ChRkSXHoVatpkg7\nHAdfjYL+20fWJ9bjCoHmAgN96rh0SeHRiqj+eHUsIT+0exzH4x1rLoqm5k/5ihur\nW+PbdGQDLDcu4JqwK+ru7m+mqoyvfnt7b23b6/d+Vs5rpw6BHQlydf68ilzPBlDn\nwvMzcgyFOQIqellTlEtbfwWe8okis5PCWdvOBVFEYfO2YW1W2/UV7MpudQYvSezx\nL5QRwvfOBsxHfnO/0B+lHOZ7U3HL+3RQOwIDAQABo4GGMIGDMA4GA1UdDwEB/wQE\nAwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud\nDgQWBBS3C8GPVx+7k+/UJ8JrJviJh8s2GTAsBgNVHREEJTAjgglsb2NhbGhvc3SH\nBH8AAAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQELBQADggEBAKe4BlaY\ndfb6XTOmRc764SaHozqR3Jsa6qbvYZEIpEPowkKfg5Wa+A3qVRvbwo5vPuJ4BFWD\nndciuwoi0KWx1SfrvgairIki1qSTChOZGy4+VpoBTvSbdyQUI1rtafC47M984QWp\nwbec6zteONJJmB6THRCAkHuiFmO6gSkEAnpBuOH8CdoJbaD/7Y0DoukpLM5lfb/U\nDOd8t+oxMTHLEkgpD4J7mT/IEGAUg3J/7o0/kjRMBDP9EdBStgrWCdrX6wIe0Dd4\nw8jMX2C4WYi1g07fT3TujXTIlZi153Pj9YoLKYq7kjn3BpJ19pgEqLXpEZ4iOY9u\nKT1x3EqvZ6VFx1I=\n-----END CERTIFICATE-----\n"
                }
            ]
        }
    ],
    "rules": [
        {
            "name": "default",
            "isSend": true,
            "localDir": "default/out"
        },
        {
            "name": "default",
            "isSend": false,
            "localDir": "default/in"
        },
        {
            "name": "backup-and-delete",
            "isSend": true,
            "localDir": "backup-and-delete/out",
            "post": [
                {
                    "type": "DELETE"
                }
            ]
        },
        {
            "name": "backup-and-delete",
            "isSend": false,
            "localDir": "backup-and-delete/in",
            "post": [
                {
                    "type": "MOVERENAME",
                    "args": {
                        "path": "backup/#TRUEFILENAME#.back"
                    }
                }
            ]
        },
        {
            "name": "with-compression",
            "isSend": true,
            "localDir": "with-compression/out",
            "pre": [
                {
                    "type": "ARCHIVE",
                    "args": {
                        "files": "#TRUEFULLPATH#",
                        "outputPath": "#TRUEFULLPATH#.zip"
                    }
                }
            ]
        },
        {
            "name": "with-compression",
            "isSend": false,
            "localDir": "with-compression/in",
            "post": [
                {
                    "type": "EXTRACT",
                    "args": {
                        "outputDir": "with-compression/in"
                    }
                }
            ]
        },
        {
            "name": "with-encryption",
            "isSend": true,
            "localDir": "with-encryption/out",
            "pre": [
                {
                    "type": "ENCRYPT",
                    "args": {
                        "keepOriginal": "false",
                        "outputFile": "#TRUEFULLPATH#.pgp",
                        "method": "PGP",
                        "keyName": "pgp-pubkey"
                    }
                }
            ]
        },
        {
            "name": "with-encryption",
            "isSend": false,
            "localDir": "with-encryption/in",
            "post": [
                {
                    "type": "DECRYPT",
                    "args": {
                        "keepOriginal": "false",
                        "outputFile": "#BASEFILENAME#",
                        "method": "PGP",
                        "keyName": "pgp-privkey"
                    }
                }
            ]
        }
    ],
    "cryptoKeys": [
        {
            "name": "pgp-pubkey",
            "type": "PGP-PUBLIC",
            "key": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\n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\n=yUng\n-----END PGP PUBLIC KEY BLOCK-----\n"
        },
        {
            "name": "pgp-privkey",
            "type": "PGP-PRIVATE",
            "key": "-----BEGIN PGP PRIVATE KEY BLOCK-----\n\n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\nt2kQEnyiQDOoD+o/jFrAm+nUnnioUHXGMw==\n=fW5w\n-----END PGP PRIVATE KEY BLOCK-----\n"
        }
    ]
}`

	modeltest.AddDummyProtoConfig("r66-tls")
	modeltest.AddDummyProtoConfig("sftp")
	modeltest.AddDummyProtoConfig("https")
	modeltest.AddDummyProtoConfig("ftps")
	modeltest.AddDummyProtoConfig("pesit-tls")

	authtest.AddDummyAuthHandler("ssh_private_key", "sftp")

	database.BcryptRounds = 12

	return confJson
}
