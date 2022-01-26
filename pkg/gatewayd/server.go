// Package gatewayd contains the "root" service responsible for starting all
// the other gateway services. It is the first service started when the gateway
// is launched.
package gatewayd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

const (
	defaultStopTimeout = 10 * time.Second
)

type serviceConstructor func(db *database.DB, agent *model.LocalAgent, logger *log.Logger) service.ProtoService

// WG is the top level service handler. It manages all other components.
type WG struct {
	*log.Logger

	dbService     *database.DB
	adminService  *admin.Server
	controller    *controller.Controller
	ProtoServices map[string]service.ProtoService
}

// NewWG creates a new application.
func NewWG() *WG {
	return &WG{
		Logger: log.NewLogger("Waarp-Gateway"),
	}
}

func (wg *WG) makeDirs() error {
	config := &conf.GlobalConfig.Paths

	if err := os.MkdirAll(config.GatewayHome, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway home directory: %w", err)
	}

	if err := os.MkdirAll(config.DefaultInDir, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway in directory: %w", err)
	}

	if err := os.MkdirAll(config.DefaultOutDir, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway out directory: %w", err)
	}

	if err := os.MkdirAll(config.DefaultTmpDir, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway work directory: %w", err)
	}

	return nil
}

func (wg *WG) initServices() {
	core := make(map[string]service.Service)
	wg.ProtoServices = make(map[string]service.ProtoService)

	wg.dbService = &database.DB{}
	wg.adminService = &admin.Server{
		DB:            wg.dbService,
		CoreServices:  core,
		ProtoServices: wg.ProtoServices,
	}
	wg.controller = &controller.Controller{DB: wg.dbService}

	core[service.DatabaseServiceName] = wg.dbService
	core[service.AdminServiceName] = wg.adminService
	core[service.ControllerServiceName] = wg.controller
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
		l := log.NewLogger(servers[i].Name)

		constr, ok := ServiceConstructors[servers[i].Protocol]
		if !ok {
			wg.Logger.Warningf("Unknown protocol '%s' for server %s",
				servers[i].Protocol, servers[i].Name)

			continue
		}

		serv := constr(wg.dbService, &servers[i], l)
		wg.ProtoServices[servers[i].Name] = serv
	}

	for _, serv := range wg.ProtoServices {
		if err := serv.Start(); err != nil {
			wg.Logger.Errorf("Error starting service: %s", err)
		}
	}

	return nil
}

func (wg *WG) stopServices() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStopTimeout)
	defer cancel()

	w := sync.WaitGroup{}

	for _, wgService := range wg.ProtoServices {
		if code, _ := wgService.State().Get(); code != service.Running && code != service.Starting {
			continue
		}

		w.Add(1)

		go func(s service.Service) {
			defer w.Done()

			if err := s.Stop(ctx); err != nil {
				wg.Logger.Warningf("an error occurred while stopping the service: %v", err)
			}
		}(wgService)
	}

	w.Wait()

	if err := wg.controller.Stop(ctx); err != nil {
		wg.Logger.Warningf("an error occurred while stopping the controller service: %v", err)
	}

	if err := wg.adminService.Stop(ctx); err != nil {
		wg.Logger.Warningf("an error occurred while stopping the admin service: %v", err)
	}

	if err := wg.dbService.Stop(ctx); err != nil {
		wg.Logger.Warningf("an error occurred while stopping the database service: %v", err)
	}
}

// Start starts the main service of the Gateway.
func (wg *WG) Start() error {
	gwName := conf.GlobalConfig.GatewayName

	wg.Infof("Waarp Gateway '%s' is starting", gwName)

	if err := wg.startServices(); err != nil {
		return err
	}

	wg.Infof("Waarp Gateway '%s' has started", gwName)

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
