package main

import (
	"crypto/tls"
	"fmt"
	"time"

	"code.waarp.fr/lib/goftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	gwftp "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
)

const clientDefaultConnTimeout = 5 * time.Second // 5s

type ftpClient struct {
	ftpClient *goftp.Client
}

func (fc *ftpClient) Connect(partner *api.OutPartner, account *api.OutRemoteAccount, restAddr string,
	insecure bool,
) error {
	t, err := getTransferContext[*gwftp.PartnerConfigTLS](partner, account, restAddr, insecure)
	if err != nil {
		return err
	}

	accountPassword := getPassword(t.accountCreds)

	var (
		tlsConfig *tls.Config
		tlsMode   goftp.TLSMode
	)

	if partner.Protocol == gwftp.FTPS {
		tlsMode = goftp.TLSExplicit
		tlsConfig = getTLSConf(t.realAddr, t.partnerConf.MinTLSVersion, t.partnerCreds, t.accountCreds, nil)

		if t.partnerConf.UseImplicitTLS {
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
	return list(fc.ftpClient, rule, pattern)
}

func (fc *ftpClient) Close() error {
	fc.ftpClient.Close()

	return nil
}
