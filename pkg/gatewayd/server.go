// Package gatewayd contains the "root" service responsible for starting all
// the other gateway services. It is the first service started when the gateway
// is launched.
package gatewayd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	defaultStopTimeout = 10 * time.Second
)

// WG is the top level service handler. It manages all other components.
type WG struct {
	*log.Logger

	dbService    *database.DB
	adminService *admin.Server
	snmpService  *snmp.Service
	controller   *controller.Controller
}

// NewWG creates a new application.
func NewWG() *WG {
	return &WG{
		Logger: logging.NewLogger("Waarp-Gateway"),
	}
}

func getDir(root *types.URL, dir string) (*types.URL, error) {
	if types.GetScheme(dir) != "" || filepath.IsAbs(dir) {
		if url, err := types.ParseURL(dir); err != nil {
			return nil, fmt.Errorf("failed to parse the url: %w", err)
		} else {
			return url, nil
		}
	}

	return root.JoinPath(dir), nil
}

func parseDirs() (root, in, out, tmp *types.URL, err error) {
	config := &conf.GlobalConfig.Paths

	root, rootErr := types.ParseURL(config.GatewayHome)
	if rootErr != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse root directory: %w", rootErr)
	}

	in, inErr := getDir(root, config.DefaultInDir)
	if inErr != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse in directory: %w", inErr)
	}

	out, outErr := getDir(root, config.DefaultOutDir)
	if outErr != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse out directory: %w", outErr)
	}

	tmp, tmpErr := getDir(root, config.DefaultTmpDir)
	if tmpErr != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse tmp directory: %w", tmpErr)
	}

	return root, in, out, tmp, nil
}

func (wg *WG) makeDirs() error {
	root, in, out, tmp, parseErr := parseDirs()
	if parseErr != nil {
		return parseErr
	}

	rootFS, fsErr := fs.GetFileSystem(wg.dbService, root)
	if fsErr != nil {
		return fmt.Errorf(`failed to instantiate gateway home file system: %w`, fsErr)
	}

	if err := fs.MkdirAll(rootFS, root); err != nil {
		return fmt.Errorf("failed to create gateway home directory: %w", err)
	}

	inFS := rootFS
	outFS := rootFS
	tmpFS := rootFS

	if !fs.IsOnSameFS(root, in) {
		if inFS, fsErr = fs.GetFileSystem(wg.dbService, in); fsErr != nil {
			return fmt.Errorf(`failed to instantiate gateway "in" file system: %w`, fsErr)
		}
	}

	if err := fs.MkdirAll(inFS, in); err != nil {
		return fmt.Errorf("failed to create gateway in directory: %w", err)
	}

	if !fs.IsOnSameFS(root, out) {
		if outFS, fsErr = fs.GetFileSystem(wg.dbService, out); fsErr != nil {
			return fmt.Errorf(`failed to instantiate gateway "out" file system: %w`, fsErr)
		}
	}

	if err := fs.MkdirAll(outFS, out); err != nil {
		return fmt.Errorf("failed to create gateway out directory: %w", err)
	}

	if !fs.IsOnSameFS(root, tmp) {
		if tmpFS, fsErr = fs.GetFileSystem(wg.dbService, tmp); fsErr != nil {
			return fmt.Errorf(`failed to instantiate gateway "out" file system: %w`, fsErr)
		}
	}

	if err := fs.MkdirAll(tmpFS, tmp); err != nil {
		return fmt.Errorf("failed to create gateway work directory: %w", err)
	}

	return nil
}

func (wg *WG) initServices() {
	wg.dbService = &database.DB{}
	wg.snmpService = &snmp.Service{DB: wg.dbService}
	wg.adminService = &admin.Server{DB: wg.dbService}
	gwController := controller.GatewayController{DB: wg.dbService}
	wg.controller = &controller.Controller{Action: gwController.Run}

	snmp.GlobalService = wg.snmpService
}

func (wg *WG) startServices() error {
	wg.initServices()

	if err := wg.dbService.Start(); err != nil {
		return fmt.Errorf("cannot start database service: %w", err)
	}

	if err := wg.snmpService.Start(); err != nil {
		return fmt.Errorf("cannot start snmp service: %w", err)
	}

	if err := wg.adminService.Start(); err != nil {
		return fmt.Errorf("cannot start admin service: %w", err)
	}

	if err := wg.controller.Start(); err != nil {
		return fmt.Errorf("cannot start controller service: %w", err)
	}

	if err := wg.makeDirs(); err != nil {
		return err
	}

	services.Core[database.ServiceName] = wg.dbService
	services.Core[controller.ServiceName] = wg.controller
	services.Core[admin.ServiceName] = wg.adminService
	services.Core[snmp.ServiceName] = wg.snmpService

	if err := wg.startServers(); err != nil {
		return err
	}

	if err := wg.startClients(); err != nil {
		return err
	}

	return nil
}

//nolint:dupl //too many differences
func (wg *WG) startServers() error {
	var servers model.LocalAgents
	if err := wg.dbService.Select(&servers).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return fmt.Errorf("failed to retrieve servers from the database: %w", err)
	}

	for _, server := range servers {
		module := protocols.Get(server.Protocol)
		if module == nil {
			wg.Logger.Error("Unknown protocol %q for server %q", server.Protocol, server.Name)

			continue
		}

		serverService := module.NewServer(wg.dbService, server)
		services.Servers[server.Name] = serverService

		if !server.Disabled {
			if err := serverService.Start(); err != nil {
				wg.Logger.Error("Error starting the %q server: %v", server.Name, err)
			}
		}
	}

	return nil
}

//nolint:dupl //too many differences
func (wg *WG) startClients() error {
	var dbClients model.Clients
	if err := wg.dbService.Select(&dbClients).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return fmt.Errorf("failed to retrieve clients from the database: %w", err)
	}

	for _, client := range dbClients {
		module := protocols.Get(client.Protocol)
		if module == nil {
			wg.Logger.Error("Unknown protocol %q for client %q", client.Protocol, client.Name)

			continue
		}

		clientService := module.NewClient(wg.dbService, client)
		services.Clients[client.Name] = clientService

		if !client.Disabled {
			if err := clientService.Start(); err != nil {
				wg.Logger.Error("Error starting the %q client: %v", client.Name, err)
			}
		}
	}

	return nil
}

func (wg *WG) stopServices() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStopTimeout)
	defer cancel()

	w := sync.WaitGroup{}
	stop := func(name string, s protocol.Server) {
		defer w.Done()

		if err := s.Stop(ctx); err != nil {
			wg.Logger.Warning("an error occurred while stopping the %q service: %v", name, err)
		}
	}

	for name, clientService := range services.Clients {
		if code, _ := clientService.State(); code != utils.StateRunning {
			continue
		}

		w.Add(1)
		stop(name, clientService)
	}

	for name, serverService := range services.Servers {
		if code, _ := serverService.State(); code != utils.StateRunning {
			continue
		}

		w.Add(1)
		stop(name, serverService)
	}

	w.Wait()

	if err := wg.controller.Stop(ctx); err != nil {
		wg.Logger.Warning("an error occurred while stopping the controller service: %v", err)
	}

	if err := wg.adminService.Stop(ctx); err != nil {
		wg.Logger.Warning("an error occurred while stopping the admin service: %v", err)
	}

	if err := wg.dbService.Stop(ctx); err != nil {
		wg.Logger.Warning("an error occurred while stopping the database service: %v", err)
	}
}

// Start starts the main service of the Gateway.
func (wg *WG) Start() error {
	gwName := conf.GlobalConfig.GatewayName

	wg.Info("Waarp Gateway '%s' is starting", gwName)

	if err := wg.startServices(); err != nil {
		return err
	}

	wg.Info("Waarp Gateway '%s' has started", gwName)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

mainloop:
	for {
		switch <-c {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			wg.stopServices()

			break mainloop
		}
	}

	wg.Info("Server is exiting...")

	return nil
}
