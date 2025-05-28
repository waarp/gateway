// Package compatibility provides compatibility functions for the gateway's models.
package compatibility

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

//nolint:gochecknoinits //init is required here to avoid import cycles
func init() {
	tasks.GetDefaultTransferClient = GetDefaultTransferClient
}

func GetDefaultTransferClient(db *database.DB, remoteAccountID int64) (*model.Client, error) {
	// Retrieve the transfer's partner (to retrieve the transfer's protocol).
	var partner model.RemoteAgent
	if err := db.Get(&partner, "id=(SELECT remote_agent_id FROM remote_accounts WHERE id=?)",
		remoteAccountID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve transfer partner: %w", err)
	}

	// Retrieve all clients with the transfer's protocol.
	var clients model.Clients
	if err := db.Select(&clients).Where("protocol=?", partner.Protocol).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve potential transfer clients: %w", err)
	}

	// If more than one client was found, return an error to the user (because
	// we don't know which one to use, so we ask the user to specify one).
	if len(clients) > 1 {
		return nil, database.NewValidationError("the transfer is missing a client ID")
	}

	// If exactly one client was found, use it.
	if len(clients) == 1 {
		return clients[0], nil
	}

	// Finally, if no clients were found, create a new default one and use it.
	client := &model.Client{Protocol: partner.Protocol}
	if err := db.Insert(client).Run(); err != nil {
		return nil, fmt.Errorf("failed to create new transfer client: %w", err)
	}

	module := protocols.Get(client.Protocol)
	service := module.NewClient(db, client)
	services.Clients[client.Name] = service

	if err := service.Start(); err != nil {
		return nil, fmt.Errorf("failed to start transfer client: %w", err)
	}

	return client, nil
}
