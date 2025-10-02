package internal

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrServerNotRunning = errors.New("server is not running")
	ErrClientNotRunning = errors.New("client is not running")
)

func GetServerStatus(server *model.LocalAgent) (state utils.StateCode, reason string) {
	service, hasService := services.Servers[server.Name]
	if !hasService {
		return utils.StateOffline, ""
	}

	return service.State()
}

func GetClientStatus(client *model.Client) (state utils.StateCode, reason string) {
	service, hasService := services.Clients[client.Name]
	if !hasService {
		return utils.StateOffline, ""
	}

	return service.State()
}

func restartService(ctx context.Context, service services.Service) error {
	if code, _ := service.State(); code == utils.StateRunning {
		if err := service.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
	}

	if err := service.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func AddServer(db *database.DB, server *model.LocalAgent) error {
	if err := InsertServer(db, server); err != nil {
		return err
	}

	module := protocols.Get(server.Protocol)
	if module == nil {
		return fmt.Errorf("%w %q", ErrUnknownProtocol, server.Protocol)
	}

	services.Servers[server.Name] = module.NewServer(db, server)

	return nil
}

func AddClient(db *database.DB, client *model.Client) error {
	if err := InsertClient(db, client); err != nil {
		return err
	}

	module := protocols.Get(client.Protocol)
	if module == nil {
		return fmt.Errorf("%w %q", ErrUnknownProtocol, client.Protocol)
	}

	services.Clients[client.Name] = module.NewClient(db, client)

	return nil
}

func RestartServer(ctx context.Context, db *database.DB, server *model.LocalAgent) error {
	service, hasService := services.Servers[server.Name]
	if !hasService {
		module := protocols.Get(server.Protocol)
		if module == nil {
			return fmt.Errorf("%w %q", ErrUnknownProtocol, server.Protocol)
		}

		service = module.NewServer(db, server)
	}

	return restartService(ctx, service)
}

func RestartClient(ctx context.Context, db *database.DB, client *model.Client) error {
	service, hasService := services.Clients[client.Name]
	if !hasService {
		module := protocols.Get(client.Protocol)
		if module == nil {
			return fmt.Errorf("%w %q", ErrUnknownProtocol, client.Protocol)
		}

		service = module.NewClient(db, client)
	}

	return restartService(ctx, service)
}

func StopServer(ctx context.Context, server *model.LocalAgent) error {
	service, hasService := services.Servers[server.Name]
	if !hasService {
		return ErrServerNotRunning
	}

	if status, _ := service.State(); status != utils.StateRunning {
		return ErrServerNotRunning
	}

	if err := service.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	return nil
}

func StopClient(ctx context.Context, client *model.Client) error {
	service, hasService := services.Clients[client.Name]
	if !hasService {
		return ErrClientNotRunning
	}

	if status, _ := service.State(); status != utils.StateRunning {
		return ErrClientNotRunning
	}

	if err := service.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop client: %w", err)
	}

	return nil
}

func RemoveServer(ctx context.Context, db *database.DB, server *model.LocalAgent) error {
	if err := StopServer(ctx, server); err != nil && !errors.Is(err, ErrServerNotRunning) {
		return err
	}

	delete(services.Servers, server.Name)

	return DeleteServer(db, server)
}

func RemoveClient(ctx context.Context, db *database.DB, client *model.Client) error {
	if err := StopClient(ctx, client); err != nil && !errors.Is(err, ErrClientNotRunning) {
		return err
	}

	delete(services.Clients, client.Name)

	return DeleteClient(db, client)
}
