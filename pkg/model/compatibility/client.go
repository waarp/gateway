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

func GetDefaultTransferClient(db database.Access, protocol string) (*model.Client, error) {
	// Retrieve all clients with the transfer's protocol.
	var clients model.Clients
	if err := db.Select(&clients).Where("protocol=?", protocol).Owner().Run(); err != nil {
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
	client := &model.Client{Protocol: protocol}
	if err := db.Insert(client).Run(); err != nil {
		return nil, fmt.Errorf("failed to create new transfer client: %w", err)
	}

	module := protocols.Get(client.Protocol)
	service := module.NewClient(db.AsDB(), client) // Give the client its own db connection.
	services.Clients[client.Name] = service

	// Start the new client. This should be fine as long as the client does not
	// start a transaction while starting. It normally shouldn't need to, but
	// if it does, then it will fail to start.
	if err := service.Start(); err != nil {
		return nil, fmt.Errorf("failed to start transfer client: %w", err)
	}

	return client, nil
}
