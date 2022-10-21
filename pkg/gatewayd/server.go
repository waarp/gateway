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
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/constructors"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/names"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	defaultStopTimeout = 10 * time.Second
)

// WG is the top level service handler. It manages all other components.
type WG struct {
	*log.Logger

	dbService     *database.DB
	adminService  *admin.Server
	controller    *controller.Controller
	ProtoServices map[uint64]proto.Service
}

// NewWG creates a new application.
func NewWG() *WG {
	return &WG{
		Logger: conf.GetLogger("Waarp-Gateway"),
	}
}

func getDir(dir, home string) string {
	if filepath.IsAbs(dir) {
		return dir
	}

	return filepath.Join(home, dir)
}

func (wg *WG) makeDirs() error {
	config := &conf.GlobalConfig.Paths

	if err := os.MkdirAll(config.GatewayHome, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway home directory: %w", err)
	}

	if err := os.MkdirAll(getDir(config.DefaultInDir, config.GatewayHome), 0o744); err != nil {
		return fmt.Errorf("failed to create gateway in directory: %w", err)
	}

	if err := os.MkdirAll(getDir(config.DefaultOutDir, config.GatewayHome), 0o744); err != nil {
		return fmt.Errorf("failed to create gateway out directory: %w", err)
	}

	if err := os.MkdirAll(getDir(config.DefaultTmpDir, config.GatewayHome), 0o744); err != nil {
		return fmt.Errorf("failed to create gateway work directory: %w", err)
	}

	return nil
}

func (wg *WG) initServices() {
	core := make(map[string]service.Service)
	wg.ProtoServices = make(map[uint64]proto.Service)

	wg.dbService = &database.DB{}
	wg.adminService = &admin.Server{
		DB:            wg.dbService,
		CoreServices:  core,
		ProtoServices: wg.ProtoServices,
	}
	gwController := controller.GatewayController{DB: wg.dbService}
	wg.controller = &controller.Controller{
		Action: gwController.Run,
	}

	core[names.DatabaseServiceName] = wg.dbService
	core[names.AdminServiceName] = wg.adminService
	core[names.ControllerServiceName] = wg.controller
}

func (wg *WG) startServices() error {
	if err := wg.makeDirs(); err != nil {
		return err
	}

	wg.initServices()

	if err := wg.dbService.Start(); err != nil {
		return fmt.Errorf("cannot start database service: %w", err)
	}

	if err := wg.adminService.Start(); err != nil {
		return fmt.Errorf("cannot start admin service: %w", err)
	}

	if err := wg.controller.Start(); err != nil {
		return fmt.Errorf("cannot start controller service: %w", err)
	}

	var servers model.LocalAgents
	if err := wg.dbService.Select(&servers).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return err
	}

	for i := range servers {
		server := &servers[i]
		l := conf.GetLogger(server.Name)

		constr, ok := constructors.ServiceConstructors[server.Protocol]
		if !ok {
			wg.Logger.Warning("Unknown protocol '%s' for server %s",
				server.Protocol, server.Name)

			continue
		}

		protoService := constr(wg.dbService, l)
		wg.ProtoServices[server.ID] = protoService

		if server.Enabled {
			if err := protoService.Start(server); err != nil {
				wg.Logger.Error("Error starting the %q service: %v", server.Name, err)
			}
		}
	}

	return nil
}

func (wg *WG) stopServices() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStopTimeout)
	defer cancel()

	w := sync.WaitGroup{}

	for _, wgService := range wg.ProtoServices {
		if code, _ := wgService.State().Get(); code != state.Running && code != state.Starting {
			continue
		}

		w.Add(1)

		go func(s stopper) {
			defer w.Done()

			if err := s.Stop(ctx); err != nil {
				wg.Logger.Warning("an error occurred while stopping the service: %v", err)
			}
		}(wgService)
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

	wg.Info("Service is exiting...")

	return nil
}

type stopper interface {
	Stop(ctx context.Context) error
}
