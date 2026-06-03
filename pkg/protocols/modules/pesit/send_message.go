package pesit

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/lib/pesit"
)

//nolint:gochecknoinits //init is required to wire the task callback

func init() {
	tasks.SendPeSITMessage = sendPeSITMessage
}

// sendPeSITMessage opens a standalone PeSIT connection to the given partner

// and sends a F.MESSAGE. The connection is closed after the message is sent.

// This is called by the SENDMESSAGE task via the global function pointer.

func sendPeSITMessage(db *database.DB, partnerName, accountLogin string,

	transferID uint32, message string, logger *log.Logger,
) error {
	// Resolve partner

	var partner model.RemoteAgent

	if err := db.Get(&partner, "name=?", partnerName).Owner().Run(); err != nil {
		return fmt.Errorf("partner %q not found: %w", partnerName, err)
	}

	// Resolve account

	var account model.RemoteAccount

	if err := db.Get(&account, "login=? AND remote_agent_id=?",

		accountLogin, partner.ID).Run(); err != nil {
		return fmt.Errorf("account %q not found for partner %q: %w",

			accountLogin, partnerName, err)
	}

	// Get account credentials (password)

	var creds model.Credentials

	if err := db.Select(&creds).Where("remote_account_id=?",

		account.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	var password string

	for _, cred := range creds {
		if cred.Type == auth.Password {

			password = cred.Value

			break

		}
	}

	// Parse partner config

	var partConf PartnerConfigTLS

	if err := utils.JSONConvert(partner.ProtoConfig, &partConf); err != nil {
		return fmt.Errorf("failed to parse partner config: %w", err)
	}

	// Dial TCP

	addr := conf.GetRealAddress(partner.Address.Host,

		utils.FormatUint(partner.Address.Port))

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	// Optional TLS

	isTLS := partner.Protocol == PesitTLS

	if isTLS {

		tlsConf := &tls.Config{
			//nolint:gosec // server name from config

			ServerName: partner.Address.Host,

			InsecureSkipVerify: true, // TODO: proper TLS config from partner creds

		}

		conn = tls.Client(conn, tlsConf)

	}

	// Create PeSIT client

	serverLogin := partner.Name

	if partConf.Login != "" {
		serverLogin = partConf.Login
	}

	client := pesit.NewClient(accountLogin, password, serverLogin)

	client.Logger = logger.AsStdLogger(log.LevelDebug)

	// Configure

	if partConf.CompatibilityMode == CompatibilityModeHistorique {
		client.SetHistoriqueMode(true)
	}

	if !partConf.DisablePreConnection && !isTLS {
		client.SetPreConnectionUsage(true)
	} else {
		client.SetPreConnectionUsage(false)
	}

	// Connect

	if err := client.Connect(conn); err != nil {

		conn.Close()

		return fmt.Errorf("PeSIT connect failed: %w", err)

	}

	// Send message

	if err := client.SendMessage(transferID, message); err != nil {

		client.Close(nil)

		return fmt.Errorf("SendMessage failed: %w", err)

	}

	// Close gracefully

	if err := client.Close(nil); err != nil {
		logger.Warningf("SENDMESSAGE: close warning: %v", err)
	}

	return nil
}
