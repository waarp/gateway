package internal

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func GetServerStatus(server *model.LocalAgent) (state utils.StateCode, reason string) {
	service, hasService := services.Servers.Load(server)
	if !hasService {
		return utils.StateOffline, ""
	}

	return service.State()
}

func GetClientStatus(client *model.Client) (state utils.StateCode, reason string) {
	service, hasService := services.Clients.Load(client)
	if !hasService {
		return utils.StateOffline, ""
	}

	return service.State()
}

func AddServer(db *database.DB, server *model.LocalAgent) error {
	if err := InsertServer(db, server); err != nil {
		return err
	}

	service, err := protocols.MakeServer(db, server)
	if err != nil {
		return err
	}

	services.Servers.Add(server, service)

	return nil
}

func AddClient(db *database.DB, client *model.Client) error {
	if err := InsertClient(db, client); err != nil {
		return err
	}

	service, err := protocols.MakeClient(db, client)
	if err != nil {
		return err
	}

	services.Clients.Add(client, service)

	return nil
}

func RestartServer(ctx context.Context, db *database.DB, server *model.LocalAgent) error {
	newService, err := protocols.MakeServer(db, server)
	if err != nil {
		return err
	}

	return services.Servers.Restart(ctx, server, newService)
}

func RestartClient(ctx context.Context, db *database.DB, client *model.Client) error {
	newService, err := protocols.MakeClient(db, client)
	if err != nil {
		return err
	}

	return services.Clients.Restart(ctx, client, newService)
}

func StopServer(ctx context.Context, server *model.LocalAgent) error {
	return services.Servers.Stop(ctx, server, false)
}

func StopClient(ctx context.Context, client *model.Client) error {
	return services.Clients.Stop(ctx, client, false)
}

func RemoveServer(ctx context.Context, db *database.DB, server *model.LocalAgent) error {
	if err := services.Servers.Stop(ctx, server, true); err != nil {
		return err
	}

	return DeleteServer(db, server)
}

func RemoveClient(ctx context.Context, db *database.DB, client *model.Client) error {
	if err := services.Clients.Stop(ctx, client, false); err != nil {
		return err
	}

	return DeleteClient(db, client)
}
