package pesit

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var _ tasks.MessageSender = &transferHandler{}

func (t *transferHandler) SendMessage(db *database.DB, logger *log.Logger,
	_ *model.Client, partner *model.RemoteAgent, account *model.RemoteAccount,
	infos map[string]any, transferID, filename, message string,
) error {
	tID, idErr := utils.ParseUint[uint32](transferID)
	if idErr != nil {
		return fmt.Errorf("failed to parse transfer ID: %w", idErr)
	}

	return sendInitialMessage(db, logger, partner, account, infos, tID, filename,
		strings.NewReader(message))
}

func (c *clientTransfer) SendMessage(db *database.DB, logger *log.Logger,
	_ *model.Client, partner *model.RemoteAgent, account *model.RemoteAccount,
	infos map[string]any, transferID, filename, message string,
) error {
	tID, idErr := utils.ParseUint[uint32](transferID)
	if idErr != nil {
		return fmt.Errorf("failed to parse transfer ID: %w", idErr)
	}

	return sendInitialMessage(db, logger, partner, account, infos, tID, filename,
		strings.NewReader(message))
}

func sendInitialMessage(db database.ReadAccess, logger *log.Logger,
	partner *model.RemoteAgent, account *model.RemoteAccount,
	infos map[string]any, transferID uint32, filename string, message io.Reader,
) error {
	partnerCreds, dbErr := partner.GetCredentials(db)
	if dbErr != nil {
		return fmt.Errorf("failed to retrieve credentials for partner: %w", dbErr)
	}

	accountCreds, dbErr := account.GetCredentials(db)
	if dbErr != nil {
		return fmt.Errorf("failed to retrieve credentials for account: %w", dbErr)
	}

	var authorities model.Authorities
	if err := db.Select(&authorities).Run(); err != nil {
		return fmt.Errorf("failed to retrieve authorities: %w", err)
	}

	return sendMessage(db, logger, partner, account, partnerCreds, accountCreds,
		authorities, infos, transferID, filename, message)
}

func sendMessage(db database.ReadAccess, logger *log.Logger,
	partner *model.RemoteAgent, account *model.RemoteAccount,
	partnerCreds, accountCreds []*model.Credential, authorities []*model.Authority,
	infos map[string]any, transferID uint32, filename string, message io.Reader,
) error {
	var partConf PartnerConfigTLS
	if err := utils.JSONConvert(partner.ProtoConfig, &partConf); err != nil {
		return fmt.Errorf("failed to parse partner config: %w", err)
	}

	serverLogin := partner.Name
	if partConf.Login != "" {
		serverLogin = partConf.Login
	}

	pesitClient := pesit.NewClient(account.Login, getPassword(accountCreds), serverLogin)
	pesitClient.Logger = logger.AsStdLogger(log.LevelDebug)
	pesitClient.NetworkTrace = logger.AsStdLogger(log.LevelTrace)
	pesitClient.SetNSDUUsage(partConf.UseNSDU.Value)

	for _, cred := range accountCreds {
		if cred.Type != PreConnectionAuth {
			continue
		}

		pesitClient.SetPreConnectionUsage(true)
		pesitClient.SetPreConnectLogin(cred.Value)
		pesitClient.SetPreConnectPassword(cred.Value2)

		break
	}

	dialer := &protoutils.TraceDialer{Dialer: &net.Dialer{}}
	addr := protoutils.GetRealAddress(db.GetConfig().Overrides, partner.Address)
	client := &model.Client{ProtoConfig: map[string]any{}}

	var (
		conn    net.Conn
		connErr error
	)

	if partner.Protocol == Pesit {
		conn, connErr = dialer.Dial("tcp", addr)
	} else {
		tlsConfig, tlsErr := protoutils.GetClientTLSConf(logger, partner,
			client, partnerCreds, accountCreds, authorities)
		if tlsErr != nil {
			return fmt.Errorf("failed to create TLS config: %w", tlsErr)
		}

		conn, connErr = tls.Dial("tcp", addr, tlsConfig)
	}

	if connErr != nil {
		return fmt.Errorf("failed to connect to partner: %w", connErr)
	}

	defer conn.Close()

	if err := pesitClient.Connect(conn); err != nil {
		return fmt.Errorf("failed to establish PeSIT connection: %w", err)
	}

	defer pesitClient.Close(nil)

	var customerID, bankID string

	if custID, err := utils.GetAs[string](infos, customerIDKey); err == nil {
		customerID = custID
	}

	if bkID, err := utils.GetAs[string](infos, bankIDKey); err == nil {
		bankID = bkID
	}

	mw, msgErr := pesitClient.NewMessage(pesit.FileACK, filename, transferID,
		pesit.WithClientLogin(pesitClient.ClientLogin()),
		pesit.WithServerLogin(pesitClient.ServerLogin()),
		pesit.WithCreationTime(time.Now()),
		pesit.WithClientID(customerID),
		pesit.WithBankID(bankID))
	if msgErr != nil {
		return fmt.Errorf("failed to create PeSIT message: %w", msgErr)
	}

	if _, err := io.Copy(mw, message); err != nil {
		return fmt.Errorf("failed to write PeSIT message: %w", err)
	}

	if err := mw.Close(); err != nil {
		return fmt.Errorf("failed to close PeSIT message: %w", err)
	}

	return nil
}
