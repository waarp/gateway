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
	"code.waarp.fr/apps/gateway/gateway/pkg/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

const (
	defaultStopTimeout = 10 * time.Second
)

// WG is the top level service handler. It manages all other components.
type WG struct {
	*log.Logger
	Conf      *conf.ServerConfig
	Services  map[string]service.Service
	dbService *database.DB
}

// NewWG creates a new application.
func NewWG(config *conf.ServerConfig) *WG {
	return &WG{
		Logger: log.NewLogger("Waarp-Gateway"),
		Conf:   config,
	}
}

func (wg *WG) makeDirs() error {
	if err := os.MkdirAll(wg.Conf.Paths.GatewayHome, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway home directory: %w", err)
	}

	if err := os.MkdirAll(wg.Conf.Paths.InDirectory, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway in directory: %w", err)
	}

	if err := os.MkdirAll(wg.Conf.Paths.OutDirectory, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway out directory: %w", err)
	}

	if err := os.MkdirAll(wg.Conf.Paths.WorkDirectory, 0o744); err != nil {
		return fmt.Errorf("failed to create gateway work directory: %w", err)
	}

	return nil
}

func (wg *WG) initServices() {
	wg.Services = make(map[string]service.Service)

	wg.dbService = &database.DB{Conf: wg.Conf}
	adminService := &admin.Server{Conf: wg.Conf, DB: wg.dbService, Services: wg.Services}
	controllerService := &controller.Controller{Conf: wg.Conf, DB: wg.dbService}

	wg.Services[admin.ServiceName] = adminService
	wg.Services[controller.ServiceName] = controllerService
}

func (wg *WG) startServices() error {
	if err := wg.dbService.Start(); err != nil {
		return fmt.Errorf("cannot start database service: %w", err)
	}

	var servers model.LocalAgents
	if err := wg.dbService.Select(&servers).Where("owner=?", database.Owner).
		Run(); err != nil {
		return err
	}

	for i := range servers {
		l := log.NewLogger(servers[i].Name)

		switch servers[i].Protocol {
		case "sftp":
			wg.Services[servers[i].Name] = sftp.NewService(wg.dbService, &servers[i], l)
		case "r66":
			wg.Services[servers[i].Name] = r66.NewService(wg.dbService, &servers[i], l)
		default:
			wg.Logger.Warningf("Unknown server protocol '%s'", servers[i].Protocol)
		}
	}

	for _, serv := range wg.Services {
		if err := serv.Start(); err != nil {
			wg.Logger.Errorf("Error starting service: %s", err)
		}
	}

	wg.Services[database.ServiceName] = wg.dbService

	return nil
}

func (wg *WG) stopServices() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStopTimeout)
	defer cancel()

	delete(wg.Services, database.ServiceName)

	w := sync.WaitGroup{}

	for _, wgService := range wg.Services {
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

	if err := wg.dbService.Stop(ctx); err != nil {
		wg.Logger.Warningf("an error occurred while stopping the database service: %v", err)
	}
}

// Start starts the main service of the Gateway.
func (wg *WG) Start() error {
	wg.Infof("Waarp Gateway '%s' is starting", wg.Conf.GatewayName)

	if err := wg.makeDirs(); err != nil {
		return err
	}

	wg.initServices()

	if err := wg.startServices(); err != nil {
		return err
	}

	wg.Infof("Waarp Gateway '%s' has started", wg.Conf.GatewayName)

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
