package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"

	"code.waarp.fr/lib/r66"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	gwr66 "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errWrongLogin = errors.New("wrong login")

type r66Client struct {
	r66Client  *r66.Client
	r66Session *r66.Session
}

func (rc *r66Client) Connect(partner *api.OutPartner, account *api.OutRemoteAccount, restAddr string,
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

	if partnerCreds, err = getCreds(partner.Credentials, restPath, insecure); err != nil {
		return fmt.Errorf("could not get partner %s credentials: %w", partner.Name, err)
	}

	restPath, err = url.JoinPath(restAddr, "/api/partners", partner.Name, "accounts", account.Login, "credentials")
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	if accountCreds, err = getCreds(account.Credentials, restPath, insecure); err != nil {
		return fmt.Errorf("could not get account %s credentials: %w", account.Login, err)
	}

	var partnerConf gwr66.PartnerConfig
	if err = utils.JSONConvert(partner.ProtoConfig, &partnerConf); err != nil {
		return fmt.Errorf("failed to parse R66 partner protocol configuration: %w", err)
	}

	restPath, err = url.JoinPath(restAddr, "/api/override/addresses")
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	if realAddr, err = getRealAddress(partner.Address, restPath, insecure); err != nil {
		return fmt.Errorf("could not get address for partner %s: %w", partner.Name, err)
	}

	conn, dialErr := net.Dial("tcp", realAddr)
	if dialErr != nil {
		return fmt.Errorf("failed to connect to the R66 partner: %w", dialErr)
	}

	if partner.Protocol == string(gwr66.R66TLS) || (partnerConf.IsTLS != nil && *partnerConf.IsTLS) {
		tlsConf := getTLSConf(partner.Address, 0, partnerCreds, accountCreds, nil)
		conn = tls.Client(conn, tlsConf)
	}

	client, err := r66.NewClient(conn, nil)
	if err != nil {
		return fmt.Errorf("could not open r66 connection to %s: %w", partner.Name, err)
	}
	rc.r66Client = client

	session, err := rc.r66Client.NewSession()
	if err != nil {
		return fmt.Errorf("could not open r66 connection to %s: %w", partner.Name, err)
	}
	rc.r66Session = session

	var accountPassword string

	for _, cred := range accountCreds {
		if cred.Type == auth.Password {
			accountPassword = cred.Value

			break
		}
	}

	authent, err := rc.r66Session.Authent(account.Login, []byte(accountPassword), nil)
	if err != nil {
		return fmt.Errorf("could not authent account %s: %w", account.Login, err)
	}

	var partnerPassword string

	for _, cred := range partnerCreds {
		if cred.Type == auth.Password {
			partnerPassword = cred.Value

			break
		}
	}

	partnerLogin := partner.Name
	if partnerConf.ServerLogin != "" {
		partnerLogin = partnerConf.ServerLogin
	}

	return validateServerAuthent(partnerLogin, authent.Login, []byte(partnerPassword), authent.Password)
}

func validateServerAuthent(partnerLogin, networkLogin string, partnerPassword, networkPassword []byte) error {
	loginOK := utils.ConstantEqual(partnerLogin, networkLogin)
	pwdErr := bcrypt.CompareHashAndPassword(partnerPassword, networkPassword)

	if !loginOK {
		return fmt.Errorf("server %s authentication failed: %w", partnerLogin, errWrongLogin)
	}

	if pwdErr != nil {
		return fmt.Errorf("server %s authentication failed: wrong password: %w", partnerLogin, pwdErr)
	}

	return nil
}

func (rc *r66Client) List(rule *api.OutRule, pattern string) ([]string, error) {
	resp, err := rc.r66Session.GetFileInfoV2(pattern, rule.Name, r66.InfoListDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	list := []string{}

	for i := range resp {
		if resp[i].Type != "directory" {
			list = append(list, resp[i].Name)
		}
	}

	return list, nil
}

func (rc *r66Client) Close() error {
	rc.r66Session.Close()
	rc.r66Client.Close()

	return nil
}
