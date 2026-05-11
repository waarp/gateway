package main

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

const (
	Size10MB              = 10 * 1024 * 1024
	defaultR66Port        = 6666
	defaultR66TLSPort     = 6667
	defaultSFTPPort       = 6622
	defaultRESTPort       = 8080
	defaultR66MonitPort   = 8066
	defaultR66AdminPort   = 8067
	defaultConfigTemplate = 2
	aesKeySize            = 32
	defaultR66TLSCert     = `-----BEGIN CERTIFICATE-----
MIID8zCCAtugAwIBAgIUMnBOwx9aTHRZ/hQUpJDWsLsUS+kwDQYJKoZIhvcNAQEL
BQAwgYgxCzAJBgNVBAYTAkZSMQ8wDQYDVQQIDAZGcmFuY2UxETAPBgNVBAcMCE5h
bnRlcnJlMQ4wDAYDVQQKDAVXYWFycDESMBAGA1UECwwJVGVzdCBjZXJ0MRIwEAYD
VQQDDAkxMjcuMC4wLjExHTAbBgkqhkiG9w0BCQEWDmluZm9Ad2FhcnAub3JnMB4X
DTIyMDQxMTEzMjUyM1oXDTIzMDQxMTEzMjUyM1owgYgxCzAJBgNVBAYTAkZSMQ8w
DQYDVQQIDAZGcmFuY2UxETAPBgNVBAcMCE5hbnRlcnJlMQ4wDAYDVQQKDAVXYWFy
cDESMBAGA1UECwwJVGVzdCBjZXJ0MRIwEAYDVQQDDAkxMjcuMC4wLjExHTAbBgkq
hkiG9w0BCQEWDmluZm9Ad2FhcnAub3JnMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAhMO6oKhwSoMVNW4e6xCZvbgvmFoQ3i1JFtQsP00d7RJ6EtT7eRet
Kb3ubp98hsGP3kKGhy8xCEKnBCx+JM1r6lN0xOXBY3+MkTACB+mGqwOd2gj6VoZg
l54SEHLFRoVDrTxBcFRWKAJ2KvP9PlxuReWOZhYD1Fqvbn2958ToS5t9v/BLaQEp
Ns/3oio4fzhHYR2yXjkozq0dVeHz9XKdB1kxWomsyDmO6U8XV3T9j/hRih6Y5X6p
pWudrCuJm+Dv6jwwtvW/sjvKiTeXGm3/1MFGzbc4SWkLM2VjtnPd/QEP6W3xmmh5
dHus37VEybRXCUiuvpKyR7rbXUC0bf6rAQIDAQABo1MwUTAdBgNVHQ4EFgQURWlt
UoYSSB5cdDS03mXMuvfqZWwwHwYDVR0jBBgwFoAURWltUoYSSB5cdDS03mXMuvfq
ZWwwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAORc7P8rNqTrL
EsYusQ9sTe+LQzuU2aRZS+M+V0CYFEFuZ/07UKDWJDNnyCTeusV21rifNq9GF5rq
x1K1NmgnjVRmXin+OS/KWqn1AG+sVYLTRUpg9zWtt02icCk1vynWPWk+wR7djwsX
VieWqEu4klvOAuz0IFJrWj47FKKywbaDAXvteIdD1CNMXcIi1nskHQ9tG2eMJSjC
+pTJ0W4PQZBO7RMRJDDENuqt/VZHy9BEIMyna0IaIhPPeMZgRYTz02ZCRC1b81sB
4viITRhMmJPU7jYfSLM8EVxCI2olHequfneoyGx89E8livYt2IWTk6/mEfsdeOxX
8qAzhavy+g==
-----END CERTIFICATE-----`
	defaultR66TLSKey = `-----BEGIN PRIVATE KEY-----
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
)

var (
	errConfURLNotFound    = errors.New("configuration URL not found in Manager")
	errBadInitConfig      = errors.New("bad configuration to register Gateway in Manager")
	errManagerBadResponse = errors.New("manager answered with a bad response")
)

func importConfFromManager(serverConf *conf.ServerConfig, managerURL string) error {
	logger := getLogger()

	logger.Info("Synchronizing Waarp Gateway with Waarp Manager")

	confURL, urlErr := buildConfURL(managerURL, serverConf.GatewayName)
	if urlErr != nil {
		return fmt.Errorf("cannot build the URL of the configuration package: %w", urlErr)
	}

	logger.Info("Downloading configuration from Waarp Manager")

	zipFileContent, downErr := downloadConf(confURL)
	if downErr != nil {
		return downErr
	}

	logger.Info("Getting the configuration file from the zip package")

	buf, confErr := getConfFileFromZipContent(zipFileContent, serverConf.GatewayName)
	if confErr != nil {
		return fmt.Errorf("cannot extract the configuration file from the package: %w", confErr)
	}

	logger.Info("Importing configuration into Waarp Gateway")

	if err := importConf(buf); err != nil {
		return fmt.Errorf("cannot initialize the configuration from manager: %w", err)
	}

	return nil
}

func verifyCertificates(serverConf *conf.ServerConfig) error {
	logger := getLogger()

	logger.Info("Verifying the certificates for the admin interface")

	address := serverConf.Admin.Host
	if address == "0.0.0.0" {
		setStringFromEnvDefault("MANAGER_IP", &address, "127.0.0.1")
	}

	if !certificatesExist(serverConf.Admin.TLSKey, serverConf.Admin.TLSCert) {
		logger.Warning("Manager requires that the admin interface of the Gateway is protected with HTTPS")
		logger.Warning("Self-signed certificates will be generated")

		err := generateCertificates(serverConf.Admin.TLSKey, serverConf.Admin.TLSCert, address)
		if err != nil {
			return fmt.Errorf("cannot generate certificates: %w", err)
		}
	}

	return nil
}

func verifyR66Certificates(managerConf *managerConfigData) error {
	logger := getLogger()

	logger.Info("Verifying the certificates for the r66-tls interface")

	if !certificatesExist(managerConf.R66TLSKeyPath, managerConf.R66TLSCertPath) {
		logger.Warning("No certificate found, self-signed certificates will be generated")

		err := generateCertificates(managerConf.R66TLSKeyPath, managerConf.R66TLSCertPath, managerConf.RESTAddress)
		if err != nil {
			return fmt.Errorf("cannot generate certificates: %w", err)
		}
	}

	return nil
}

func buildConfURL(managerURL, gatewayName string) (string, error) {
	parsedURL, err := url.Parse(managerURL)
	if err != nil {
		return "", fmt.Errorf("the url for Waarp Manager is invalid: %w", err)
	}

	_, pwdok := parsedURL.User.Password()

	if parsedURL.User.Username() == "" || !pwdok {
		return "", ErrMissingUsernameOrPassword
	}

	parsedURL.Path = fmt.Sprintf("/api/partners/%s/conf", gatewayName)

	return parsedURL.String(), nil
}

func downloadConf(confURL string) ([]byte, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, confURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("the following error occurred while preparing the HTTP request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot download the configuration for the gateway: %w", err)
	}

	defer func() {
		if err2 := resp.Body.Close(); err2 != nil {
			getLogger().Warningf(
				"This error occurred while closing the HTTP request: %v", err2)
		}
	}()

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, errConfURLNotFound
	}

	zipContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read the configuration file from Manager: %w", err)
	}

	return zipContent, nil
}

func getConfFileFromZipContent(zipContent []byte, gatewayName string) (*bytes.Buffer, error) {
	zipFile, zErr := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if zErr != nil {
		return nil, fmt.Errorf("cannot open the zip configuration file: %w", zErr)
	}

	var file *zip.File

	for i := range zipFile.File {
		if zipFile.File[i].Name == gatewayName+".json" {
			file = zipFile.File[i]
		}
	}

	if file == nil {
		return nil, ErrNoConfFound
	}

	confReader, opErr := file.Open()
	if opErr != nil {
		return nil, fmt.Errorf("cannot open configuration from the configuration package: %w", opErr)
	}

	defer func() {
		if err2 := confReader.Close(); err2 != nil {
			getLogger().Warningf(
				"This error occurred while reading the configuration package: %v", err2)
		}
	}()

	var buf bytes.Buffer

	if _, err := io.CopyN(&buf, confReader, Size10MB); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("cannot read configuration file from the archive: %w", err)
	}

	return &buf, nil
}

func importConf(r io.Reader) error {
	logger := getLogger()

	cmdArgs := []string{"import", "--config", defaultConfigFile}

	logger.Debugf("Command used to import the configuration: %s %s",
		gatewaydBin, strings.Join(cmdArgs, " "))

	cmd := exec.Command(gatewaydBin, cmdArgs...)
	cmd.Stdin = r

	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Infof("Import command output: %s", out)

		return fmt.Errorf("configuration import failed: %w", err)
	}

	return nil
}

type managerConfigData struct {
	ID                                  int
	Type                                string
	Site                                int
	RESTAddress                         string
	RESTPort                            uint16
	RESTUsername                        string
	RESTPassword                        string
	Password                            string
	R66Port                             uint16
	R66TLSPort                          uint16
	SFTPPort                            uint16
	SSHPublicKey                        string
	SSHPrivateKey                       string
	SSHPublicKeyPath, SSHPrivateKeyPath string
	R66TLSKeyPath, R66TLSCertPath       string
	R66TLSKey, R66TLSCert               string
}

//nolint:tagliatelle // json names must follow manager names
type gwPartner struct {
	ID          int           `json:"id"`
	Type        string        `json:"type"`
	Site        int           `json:"site"`
	IsClient    bool          `json:"isClient"`
	IsServer    bool          `json:"isServer"`
	Name        string        `json:"name"`
	RESTAddress string        `json:"restAddress"`
	RESTPort    uint16        `json:"restPort"`
	Data        gwPartnerData `json:"config"`
}

type gwPartnerData map[string]string

//nolint:tagliatelle // json names must follow manager names
type gwFlow struct {
	Name        string     `json:"name"`
	Active      bool       `json:"active"`
	Origin      int        `json:"origin"`
	Destination []flowDest `json:"destinations"`
	Template    int        `json:"template"`
}

//nolint:tagliatelle // json names must follow manager names
type flowDest struct {
	Directory string `json:"destinationDir"`
	Partner   int    `json:"partnerId"`
}

//nolint:tagliatelle // json names must follow manager names
type gwInterface struct {
	Data     gwPartnerData `json:"config"`
	Protocol string        `json:"protocol"`
	ID       int64         `json:"id"`
	Partner  int64         `json:"partner"`
	IsClient bool          `json:"isClient"`
	IsServer bool          `json:"isServer"`
	Name     string        `json:"name"`
	Address  string        `json:"address"`
	Port     int64         `json:"port"`
	Priority int64         `json:"priority"`
}

func getConfigFromEnv() (*managerConfigData, error) {
	cfg := managerConfigData{}
	if err := setIntFromEnv("MANAGER_SITE", &cfg.Site); err != nil {
		return nil, fmt.Errorf("the value for WAARP_TRANSFER_MANAGER_SITE must be a number: %w", err)
	}

	setStringFromEnv("MANAGER_IP", &cfg.RESTAddress)
	setStringFromEnv("MANAGER_PASSWORD", &cfg.Password)

	if cfg.Password == "" {
		cfg.Password = generateRandomPassword()
	}

	if err := setPortFromEnvDefault("MANAGER_R66_PORT", &cfg.R66Port, defaultR66Port); err != nil {
		return nil, fmt.Errorf("the value for WAARP_TRANSFER_MANAGER_R66_PORT must be a valid port: %w", err)
	}

	if err := setPortFromEnvDefault("MANAGER_R66TLS_PORT", &cfg.R66TLSPort, defaultR66TLSPort); err != nil {
		return nil, fmt.Errorf("the value for WAARP_TRANSFER_MANAGER_R66TLS_PORT must be a valid port: %w", err)
	}

	// if err := setPortFromEnv("MANAGER_SFTP_PORT", &partner.SFTPPort); err != nil {
	// 	return fmt.Errorf("the value for WAARP_TRANSFER_MANAGER_SFTP_PORT must be a valid port: %w", err)
	// }

	if err := setPortFromEnv("ADMIN_PORT", &cfg.RESTPort); err != nil {
		return nil, fmt.Errorf("the value for WAARP_TRANSFER_ADMIN_PORT must be a valid port: %w", err)
	}

	setStringFromEnv("MANAGER_REST_USERNAME", &cfg.RESTUsername)
	setStringFromEnv("MANAGER_REST_PASSWORD", &cfg.RESTPassword)

	// setStringFromEnv("MANAGER_SSH_PUBLIC_KEY_PATH", &partner.SSHPublicKeyPath)
	// setStringFromEnv("MANAGER_SSH_PRIVATE_KEY_PATH", &partner.SSHPrivateKeyPath)
	setStringFromEnvDefault("MANAGER_R66_TLS_CERT_PATH", &cfg.R66TLSKeyPath, "/app/etc/r66.crt")
	setStringFromEnvDefault("MANAGER_R66_TLS_KEY_PATH", &cfg.R66TLSCertPath, "/app/etc/r66.key")

	return &cfg, nil
}

func initializeGatewayInManager(serverConfig *conf.ServerConfig, managerURL string) error {
	managerConfig, err := getConfigFromEnv()
	if err != nil {
		return fmt.Errorf("cannot get the configuration from environment: %w", err)
	}

	if errVerif := verifyR66Certificates(managerConfig); errVerif != nil {
		logger := getLogger()
		logger.Criticalf("there is an issue with the certificates: %v", errVerif)
		os.Exit(ExitCannotCreateCerts)
	}

	partner := generatePartner(managerConfig, serverConfig)

	if errVerify := verifyPartner(&partner); errVerify != nil {
		return errVerify
	}

	if _, errStat := os.Stat(serverConfig.Database.AESPassphrase); os.IsNotExist(errStat) {
		logger := getLogger()
		logger.Infof(
			"The AES passphrase file %q does not exist, creating it",
			serverConfig.Database.AESPassphrase,
		)

		aesPassphrase := make([]byte, aesKeySize)
		if _, errRandRead := rand.Read(aesPassphrase); errRandRead != nil {
			return fmt.Errorf("cannot generate the AES passphrase: %w", errRandRead)
		}

		if errWrite := os.WriteFile(serverConfig.Database.AESPassphrase, aesPassphrase, 0o600); errWrite != nil {
			return fmt.Errorf(
				"cannot write the AES passphrase file %q: %w",
				serverConfig.Database.AESPassphrase, errWrite,
			)
		}
	}

	// Read aes passphrase from file
	aesPassphrase, err := os.ReadFile(serverConfig.Database.AESPassphrase)
	if err != nil {
		return fmt.Errorf("cannot read the AES passphrase file %q: %w", serverConfig.Database.AESPassphrase, err)
	}

	partner.Data["gwAESKey"] = base64.StdEncoding.EncodeToString(aesPassphrase)

	// Read R66TLS cert and key
	if err2 := handleR66KeyCert(managerConfig); err2 != nil {
		return fmt.Errorf("cannot prepare certificates for the R66TLS server of Gateway: %w", err2)
	}

	// generate sftp keys if not exist and read them and delete them
	// if err2 := handleSSHKeys(&partner); err2 != nil {
	// 	return fmt.Errorf("cannot prepare SSH keys for the Gateway: %w", err2)
	// }

	return registerGatewayInManager(&partner, managerURL, managerConfig)
}

func generateRandomPassword() string {
	const length = 10

	pwd := ""
	char := make([]byte, 1)

	for len(pwd) < length {
		_, err := rand.Read(char)
		if err != nil {
			continue
		}

		if char[0] < 33 || char[0] > 126 {
			continue
		}

		pwd += string(char[0])
	}

	return pwd
}

func generatePartner(cfg *managerConfigData, serverConfig *conf.ServerConfig) gwPartner {
	partner := gwPartner{
		Type:        "gw",
		Site:        cfg.Site,
		IsClient:    true,
		IsServer:    true,
		Name:        serverConfig.GatewayName,
		RESTAddress: cfg.RESTAddress,
		RESTPort:    cfg.RESTPort,
		Data: gwPartnerData{
			"restUser":     cmp.Or(cfg.RESTUsername, "admin"),
			"restPassword": cmp.Or(cfg.RESTPassword, "admin_password"),
		},
	}

	return partner
}

func getDefaultFlow(partner *gwPartner) gwFlow {
	flow := gwFlow{
		Active:      true,
		Destination: make([]flowDest, 1),

		// default values
		Name:     fmt.Sprintf("conf-%s", partner.Name),
		Template: defaultConfigTemplate,
	}

	return flow
}

func handleR66KeyCert(cfg *managerConfigData) error {
	if cfg.R66TLSKeyPath != "" {
		tlsKey, err2 := os.ReadFile(cfg.R66TLSKeyPath)
		if err2 != nil {
			return fmt.Errorf("cannot read the R66 TLS key file %q: %w", cfg.R66TLSKeyPath, err2)
		}

		cfg.R66TLSKey = string(tlsKey)
	}

	if cfg.R66TLSCertPath != "" {
		tlsCert, err2 := os.ReadFile(cfg.R66TLSCertPath)
		if err2 != nil {
			return fmt.Errorf("cannot read the R66 TLS certificate file %q: %w", cfg.R66TLSCertPath, err2)
		}

		cfg.R66TLSCert = string(tlsCert)
	}

	return nil
}

func verifyPartner(partner *gwPartner) error {
	// Verify required site
	if partner.Site == 0 {
		return fmt.Errorf("the variable WAARP_GATEWAY_MANAGER_SITE is required: %w", errBadInitConfig)
	}
	// Verify required IP
	if partner.RESTAddress == "" {
		return fmt.Errorf("the variable WAARP_GATEWAY_MANAGER_IP is required: %w", errBadInitConfig)
	}

	return nil
}

func authenticate(c *httpClient, u *url.URL) error {
	u.Path = "/ajax/authenticate"
	msg := url.Values{
		"login": {u.User.Username()},
	}

	pwd, pwdok := u.User.Password()
	if !pwdok {
		return ErrMissingUsernameOrPassword
	}

	msg.Add("password", pwd)

	err := c.postForm(u.String(), msg)
	if err != nil {
		return fmt.Errorf("cannot login into manager: %w", err)
	}

	return nil
}

func createR66TLSInterface(
	client *httpClient,
	reqURL *url.URL,
	cfg *managerConfigData,
	partner *gwPartner,
) error {
	ifaceTLS := gwInterface{
		Protocol: "r66-tls",
		Partner:  int64(partner.ID),
		IsClient: partner.IsClient,
		IsServer: partner.IsServer,
		Name:     partner.Name + "-r66-tls",
		Address:  cfg.RESTAddress,
		Port:     int64(cfg.R66TLSPort),
		Priority: 0,
		Data: gwPartnerData{
			"clientLogin":       partner.Name + "-r66-tls",
			"clientPassword":    cfg.Password,
			"serverLogin":       partner.Name + "-r66-tls",
			"serverPassword":    cfg.Password,
			"serverPrivateKey":  cfg.R66TLSKey,
			"serverCertificate": cfg.R66TLSCert,
			"authentMode":       "password",
		},
	}

	reqURL.Path = "/api/local_servers"
	interfacesResp := map[string]gwInterface{}
	ifaceTLSMsg := map[string]gwInterface{"localServer": ifaceTLS}

	err := client.postJSON(reqURL.String(), ifaceTLSMsg, &interfacesResp)
	if err != nil {
		return fmt.Errorf("cannot create the r66-tls interface in manager: %w", err)
	}

	return nil
}

func getManagerClientID(client *httpClient, reqURL *url.URL) (int, error) {
	managerClient := "wm_client"
	setStringFromEnv("MANAGER_CLIENT", &managerClient)

	partnersResp := map[string][]gwPartner{}
	originID := 0

	err := client.getJSON(reqURL.String(), &partnersResp)
	if err != nil {
		return 0, fmt.Errorf("cannot create partner in manager: %w", err)
	}

	partners, ok := partnersResp["partners"]
	if !ok {
		return 0, fmt.Errorf("no partner in manager response: %w", errManagerBadResponse)
	}

	for i := range partners {
		p := partners[i]

		if p.Name == managerClient {
			originID = p.ID
		}
	}

	if originID == 0 {
		return 0, fmt.Errorf("cannot find partner %q in manager: %w",
			managerClient, errBadInitConfig)
	}

	return originID, nil
}

func addConfigFlow(client *httpClient, reqURL *url.URL, partner *gwPartner, originID int) error {
	flow := getDefaultFlow(partner)

	flow.Origin = originID
	flow.Destination[0].Partner = partner.ID
	reqURL.Path = "/api/flows"
	jsonMsg := map[string]any{
		"flow": flow,
	}

	err := client.postJSON(reqURL.String(), jsonMsg, nil)
	if err != nil {
		return fmt.Errorf("cannot create the configuration flow for the gateway in Manager: %w", err)
	}

	return nil
}

func registerGatewayInManager(partner *gwPartner, managerURL string, cfg *managerConfigData) error {
	logger := getLogger()

	parsedURL, err := url.Parse(managerURL)
	if err != nil {
		return fmt.Errorf("cannot parse the URL to Manager: %w", err)
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	// log into manager
	urlCopy := *parsedURL
	if err2 := authenticate(client, &urlCopy); err2 != nil {
		return err2
	}

	// Partner creation
	logger.Info("Creating a partner in Manager")

	parsedURL.Path = "/api/partners"
	jsonMsg := map[string]any{
		"partner": partner,
	}
	partnerResp := map[string]gwPartner{"partner": {}}

	err = client.postJSON(parsedURL.String(), jsonMsg, &partnerResp)
	if err != nil {
		return fmt.Errorf("cannot create partner in manager: %w", err)
	}

	partner.ID = partnerResp["partner"].ID

	// update certs in R66Tls interfaces
	logger.Info("Adding the certificates for the Gateway R66 server in Manager")

	urlCopy = *parsedURL
	if err2 := createR66TLSInterface(client, &urlCopy, cfg, partner); err2 != nil {
		return err2
	}

	// Get the id of manager client

	urlCopy = *parsedURL

	originID, err := getManagerClientID(client, &urlCopy)
	if err != nil {
		return err
	}

	// Create a config flow
	logger.Info("Creating a configuration flow in Manager")

	urlCopy = *parsedURL

	return addConfigFlow(client, &urlCopy, partner, originID)
}
