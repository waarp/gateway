// Package gatewayd contains the "root" service responsible for starting all
// the other gateway services. It is the first service started when the gateway
// is launched.
package gatewayd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/fs/backends" // import cloud backends
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

var ErrNonLocalTmpDir = errors.New("the tmp dir must be local")

const (
	defaultStopTimeout = 10 * time.Second
)

// WG is the top level service handler. It manages all other components.
type WG struct {
	Logger *log.Logger

	DBService    *database.DB
	AdminService *admin.Server
	SnmpService  *snmp.Service
	Controller   *controller.Controller
	Analytics    *analytics.Service
}

// NewWG creates a new application.
func NewWG() *WG {
	wg := &WG{
		Logger: logging.NewLogger("Waarp-Gateway"),
	}
	wg.initServices()

	return wg
}

func getDir(root, dir string) string {
	if fs.IsAbsPath(dir) {
		return dir
	}

	return path.Join(root, dir)
}

func parseDirs() (rootDir, inDir, outDir, tmpDir string, err error) {
	config := &conf.GlobalConfig.Paths

	root := config.GatewayHome
	in := getDir(root, config.DefaultInDir)
	out := getDir(root, config.DefaultOutDir)
	tmp := getDir(root, config.DefaultTmpDir)

	if !fs.IsLocalPath(tmp) {
		return "", "", "", "", ErrNonLocalTmpDir
	}

	return root, in, out, tmp, nil
}

func (wg *WG) makeDirs() error {
	root, in, out, tmp, dirErr := parseDirs()
	if dirErr != nil {
		return dirErr
	}

	if err := fs.MkdirAll(root); err != nil {
		return fmt.Errorf("failed to create gateway home directory: %w", err)
	}

	if err := fs.MkdirAll(in); err != nil {
		return fmt.Errorf("failed to create gateway in directory: %w", err)
	}

	if err := fs.MkdirAll(out); err != nil {
		return fmt.Errorf("failed to create gateway out directory: %w", err)
	}

	if err := fs.MkdirAll(tmp); err != nil {
		return fmt.Errorf("failed to create gateway work directory: %w", err)
	}

	return nil
}

func (wg *WG) initServices() {
	wg.DBService = &database.DB{}
	wg.Analytics = &analytics.Service{DB: wg.DBService}
	wg.SnmpService = &snmp.Service{DB: wg.DBService}
	wg.AdminService = &admin.Server{DB: wg.DBService}
	wg.Controller = &controller.Controller{DB: wg.DBService}

	snmp.GlobalService = wg.SnmpService
	analytics.GlobalService = wg.Analytics
}

func (wg *WG) startServices() error {
	if err := wg.DBService.Start(); err != nil {
		return fmt.Errorf("cannot start database service: %w", err)
	}

	if err := wg.Analytics.Start(); err != nil {
		return fmt.Errorf("cannot start analytics service: %w", err)
	}

	if err := wg.SnmpService.Start(); err != nil {
		return fmt.Errorf("cannot start SNMP service: %w", err)
	}

	if err := wg.AdminService.Start(); err != nil {
		return fmt.Errorf("cannot start admin service: %w", err)
	}

	if err := wg.Controller.Start(); err != nil {
		return fmt.Errorf("cannot start controller service: %w", err)
	}

	if err := wg.makeDirs(); err != nil {
		return err
	}

	services.Core.Add(wg.DBService)
	services.Core.Add(wg.Controller)
	services.Core.Add(wg.AdminService)
	services.Core.Add(wg.SnmpService)
	services.Core.Add(wg.Analytics)

	if err := wg.startServers(); err != nil {
		return err
	}

	if err := wg.startClients(); err != nil {
		return err
	}

	return wg.startClouds()
}

//nolint:dupl //too many differences
func (wg *WG) startServers() error {
	var servers model.LocalAgents
	if err := wg.DBService.Select(&servers).Owner().Run(); err != nil {
		return fmt.Errorf("failed to retrieve servers from the database: %w", err)
	}

	for _, server := range servers {
		serverService, mkErr := protocols.MakeServer(wg.DBService, server)
		if mkErr != nil {
			wg.Logger.Errorf("Failed to instantiate server %q: %v", server.Name, mkErr)

			continue
		}

		services.Servers.Add(server, serverService)

		if !server.Disabled {
			if err := serverService.Start(); err != nil {
				wg.Logger.Errorf("Error starting the %q server: %v", server.Name, err)
			}
		}
	}

	return nil
}

//nolint:dupl //too many differences
func (wg *WG) startClients() error {
	var dbClients model.Clients
	if err := wg.DBService.Select(&dbClients).Owner().Run(); err != nil {
		return fmt.Errorf("failed to retrieve clients from the database: %w", err)
	}

	for _, client := range dbClients {
		clientService, mkErr := protocols.MakeClient(wg.DBService, client)
		if mkErr != nil {
			wg.Logger.Errorf("Failed to instantiate client %q: %v", client.Name, mkErr)

			continue
		}

		services.Clients.Add(client, clientService)

		if !client.Disabled {
			if err := clientService.Start(); err != nil {
				wg.Logger.Errorf("Error starting the %q client: %v", client.Name, err)
			}
		}
	}

	return nil
}

func (wg *WG) startClouds() error {
	var clouds model.CloudInstances
	if err := wg.DBService.Select(&clouds).Owner().Run(); err != nil {
		return fmt.Errorf("failed to retrieve clouds instances from the database: %w", err)
	}

	for _, c := range clouds {
		fileSys, err := fs.NewFS(c.Name, c.Type, c.Key, c.Secret.String(), c.Options)
		if err != nil {
			wg.Logger.Errorf("Failed to instantiate cloud instance %q: %v", c.Name, err)
			continue
		}

		fs.FileSystems.Store(c.Name, fileSys)
	}

	return nil
}

func (wg *WG) stopServices() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStopTimeout)
	defer cancel()

	errCli := make(chan error)
	errSrv := make(chan error)

	go func() {
		errCli <- services.Clients.StopAll(ctx)
	}()

	go func() {
		errSrv <- services.Servers.StopAll(ctx)
	}()

	if err := <-errCli; err != nil {
		wg.Logger.Warningf("an error occurred while stopping the clients services: %v", err)
	}

	if err := <-errSrv; err != nil {
		wg.Logger.Warningf("an error occurred while stopping the servers services: %v", err)
	}

	if err := wg.Controller.Stop(ctx); err != nil {
		wg.Logger.Warningf("an error occurred while stopping the controller service: %v", err)
	}

	if err := wg.AdminService.Stop(ctx); err != nil {
		wg.Logger.Warningf("an error occurred while stopping the admin service: %v", err)
	}

	if err := wg.SnmpService.Stop(ctx); err != nil {
		wg.Logger.Warningf("an error occurred while stopping the SNMP service: %v", err)
	}

	if err := wg.DBService.Stop(ctx); err != nil {
		wg.Logger.Warningf("an error occurred while stopping the database service: %v", err)
	}
}

// Start starts the main service of the Gateway.
func (wg *WG) Start() error {
	gwName := conf.GlobalConfig.GatewayName

	wg.Logger.Infof("Waarp Gateway %q is starting", gwName)

	if err := wg.startServices(); err != nil {
		return err
	}

	wg.Logger.Infof("Waarp Gateway %q has started", gwName)

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

	wg.Logger.Info("Server is exiting...")

	return nil
}
