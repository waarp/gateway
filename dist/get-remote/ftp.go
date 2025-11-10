package main

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"path"
	"time"

	"code.waarp.fr/lib/goftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	gwftp "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const clientDefaultConnTimeout = 5 * time.Second // 5s

type ftpClient struct {
	ftpClient *goftp.Client
}

func (fc *ftpClient) Connect(partner *api.OutPartner, account *api.OutRemoteAccount, restAddr string,
	insecure bool,
) error {
	var err error
	var partnerCreds []api.OutCred
	var accountCreds []api.OutCred
	var realAddr string

	restPath, urlErr := url.JoinPath(restAddr, "/api/partners", partner.Name, "credentials")
	if urlErr != nil {
		return fmt.Errorf("failed to build URL: %w", urlErr)
	}

	if partnerCreds, urlErr = getCreds(partner.Credentials, restPath, insecure); urlErr != nil {
		return fmt.Errorf("could not get partner %s credentials: %w", partner.Name, urlErr)
	}

	restPath, err = url.JoinPath(restAddr, "/api/partners", partner.Name, "accounts", account.Login, "credentials")
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	if accountCreds, err = getCreds(account.Credentials, restPath, insecure); err != nil {
		return fmt.Errorf("could not get account %s credentials: %w", account.Login, err)
	}

	var partnerConf gwftp.PartnerConfigTLS

	if partner.Protocol == gwftp.FTP {
		if err = utils.JSONConvert(partner.ProtoConfig, &partnerConf.PartnerConfig); err != nil {
			return fmt.Errorf("failed to parse FTP partner protocol configuration: %w", err)
		}
	} else {
		if err = utils.JSONConvert(partner.ProtoConfig, &partnerConf); err != nil {
			return fmt.Errorf("failed to parse FTPS partner protocol configuration: %w", err)
		}
	}

	restPath, err = url.JoinPath(restAddr, "/api/override/addresses")
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	if realAddr, err = getRealAddress(partner.Address, restPath, insecure); err != nil {
		return fmt.Errorf("could not get address for partner %s: %w", partner.Name, err)
	}

	var accountPassword string

	for _, cred := range accountCreds {
		if cred.Type == auth.Password {
			accountPassword = cred.Value

			break
		}
	}

	var (
		tlsConfig *tls.Config
		tlsMode   goftp.TLSMode
	)

	if partner.Protocol == gwftp.FTPS {
		tlsMode = goftp.TLSExplicit
		tlsConfig = getTLSConf(realAddr, partnerConf.MinTLSVersion, partnerCreds, accountCreds, nil)

		if partnerConf.UseImplicitTLS {
			tlsMode = goftp.TLSImplicit
		}
	}

	ftpConf := goftp.Config{
		Timeout:   clientDefaultConnTimeout,
		User:      account.Login,
		Password:  accountPassword,
		TLSConfig: tlsConfig,
		TLSMode:   tlsMode,
	}

	cli, dialErr := goftp.DialConfig(ftpConf, partner.Address)
	if dialErr != nil {
		return fmt.Errorf("failed to connect to FTP server: %w", dialErr)
	}

	fc.ftpClient = cli

	return nil
}

func (fc *ftpClient) List(rule *api.OutRule, pattern string) ([]string, error) {
	dirPattern := path.Dir(pattern)
	filePattern := path.Base(pattern)

	fileInfos, listErr := fc.ftpClient.ReadDir(path.Join(rule.RemoteDir, dirPattern))
	if listErr != nil {
		return nil, fmt.Errorf("failed to list files: %w", listErr)
	}

	res := []string{}

	for _, fi := range fileInfos {
		ok, err := path.Match(filePattern, fi.Name())
		if err != nil {
			return nil, fmt.Errorf("bad pattern: %w", err)
		}

		if ok {
			res = append(res, path.Join(dirPattern, fi.Name()))
		}
	}

	return res, nil
}

func (fc *ftpClient) Close() error {
	fc.ftpClient.Close()

	return nil
}
