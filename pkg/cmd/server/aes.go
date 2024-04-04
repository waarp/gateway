package wgd

import (
	"crypto/cipher"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	parse "code.waarp.fr/apps/gateway/gateway/pkg/tk/config"
)

type ChangeAESPassphrase struct {
	ConfigFile string `short:"c" long:"config" description:"The configuration file to use"`
	NewFile    string `short:"f" long:"file" description:"The file containing the new AES passphrase"`
}

func (c *ChangeAESPassphrase) Execute([]string) error {
	db, _, initErr := initImportExport(c.ConfigFile, nil)
	if initErr != nil {
		return initErr
	}

	return c.run(db)
}

func (c *ChangeAESPassphrase) run(db *database.DB) error {
	newGCM, gcmErr := database.NewGCM(c.NewFile)
	if gcmErr != nil {
		return fmt.Errorf("failed to load the AES passphrase file: %w", gcmErr)
	}

	serverConfig := conf.ServerConfig{}

	parser, parsErr := parse.NewParser(&serverConfig)
	if parsErr != nil {
		return fmt.Errorf("failed to initialize the config parser: %w", parsErr)
	}

	if err := changeAgentsAESPassphrase(db, newGCM); err != nil {
		return err
	}

	conf.GlobalConfig.Database.AESPassphrase = c.NewFile
	serverConfig = conf.GlobalConfig

	if err := parser.WriteFile(c.ConfigFile); err != nil {
		return fmt.Errorf("failed to update the configuration file: %w", err)
	}

	return nil
}

func changeAgentsAESPassphrase(db *database.DB, newGCM cipher.AEAD) error {
	oldGCM := database.GCM
	owner := conf.GlobalConfig.GatewayName

	var servers model.LocalAgents
	if err := db.Select(&servers).Where("owner=?", conf.GlobalConfig.GatewayName).
		In("protocol", r66.R66, r66.R66TLS).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the R66 servers: %w", err)
	}

	var creds model.Credentials
	if err := db.Select(&creds).Where(`
		local_agent_id IN (SELECT id FROM local_agents WHERE owner=?) OR
		remote_agent_id IN (SELECT id FROM remote_agents WHERE owner=?) OR
		local_account_id IN (SELECT id FROM local_accounts WHERE 
			local_agent_id IN (SELECT id FROM local_agents WHERE owner=?)) OR
		remote_account_id IN (SELECT id FROM remote_accounts WHERE
			remote_agent_id IN (SELECT id FROM remote_agents WHERE owner=?))`,
		owner, owner, owner, owner).
		In("type", auth.Password, auth.TLSCertificate, sftp.AuthSSHPrivateKey).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the credentials: %w", err)
	}

	var clouds model.CloudInstances
	if err := db.Select(&clouds).Where("owner=?", conf.GlobalConfig.GatewayName).
		Where("secret<>''").Run(); err != nil {
		return fmt.Errorf("failed to retrieve the cloud instances: %w", err)
	}

	database.GCM = newGCM
	defer func() { database.GCM = oldGCM }()

	for _, server := range servers {
		if err := db.Update(server).Run(); err != nil {
			return fmt.Errorf("failed to update the R66 server %q: %w", server.Name, err)
		}
	}

	for _, cred := range creds {
		if err := db.Update(cred).Run(); err != nil {
			return fmt.Errorf("failed to update the credential %q: %w", cred.Name, err)
		}
	}

	for _, cloud := range clouds {
		if err := db.Update(cloud).Run(); err != nil {
			return fmt.Errorf("failed to update the cloud instance %q: %w", cloud.Name, err)
		}
	}

	return nil
}
