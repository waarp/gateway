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

func (rc *r66Client) Connect(partner *api.OutPartner, account *api.OutRemoteAccount, addr string, insecure bool) error {
	var err error
	var partnerCreds []api.OutCred
	var accountCreds []api.OutCred

	restPath, urlErr := url.JoinPath(addr, "/api/partners", partner.Name, "credentials")
	if urlErr != nil {
		return fmt.Errorf("failed to build URL: %w", urlErr)
	}

	if partnerCreds, err = getCreds(partner.Credentials, restPath, insecure); err != nil {
		return fmt.Errorf("could not get partner %s credentials: %w", partner.Name, err)
	}

	restPath, err = url.JoinPath(addr, "/api/partners", partner.Name, "accounts", account.Login, "credentials")
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

	/*
		addr := conf.GetRealAddress(c.pip.TransCtx.RemoteAgent.Address.Host,
			utils.FormatUint(c.pip.TransCtx.RemoteAgent.Address.Port))
	*/

	conn, dialErr := net.Dial("tcp", partner.Address)
	if dialErr != nil {
		return fmt.Errorf("failed to connect to the R66 partner: %w", dialErr)
	}

	if partner.Protocol == string(gwr66.R66TLS) || (partnerConf.IsTLS != nil && *partnerConf.IsTLS) {
		tlsConf := getTLSConf(addr, 0, partnerCreds, accountCreds, nil)
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

	loginOK := utils.ConstantEqual(partnerLogin, authent.Login)
	pwdErr := bcrypt.CompareHashAndPassword([]byte(partnerPassword), authent.Password)

	if !loginOK {
		return fmt.Errorf("server %s authentication failed: %w", partner.Name, errWrongLogin)
	}

	if pwdErr != nil {
		return fmt.Errorf("server %s authentication failed: wrong password: %w", partner.Name, pwdErr)
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
