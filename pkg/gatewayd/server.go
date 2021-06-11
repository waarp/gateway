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

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/controller"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

type serviceConstructor func(db *database.DB, agent *model.LocalAgent, logger *log.Logger) service.Service

// ServiceConstructors is a map associating each protocol with a constructor for
// a client of said protocol. In order for the gateway to be able to perform
// client transfer with a protocol, a constructor must be added to this map, to
// allow a client to be instantiated.
var ServiceConstructors = map[string]serviceConstructor{}

// WG is the top level service handler. It manages all other components.
type WG struct {
	*log.Logger
	Conf      *conf.ServerConfig
	Services  map[string]service.Service
	dbService *database.DB
}

// NewWG creates a new application
func NewWG(config *conf.ServerConfig) *WG {
	return &WG{
		Logger: log.NewLogger("Waarp-Gateway"),
		Conf:   config,
	}
}

func (wg *WG) makeDirs() error {
	if err := os.MkdirAll(wg.Conf.Paths.GatewayHome, 0744); err != nil {
		return fmt.Errorf("failed to create gateway home directory: %s", err)
	}
	if err := os.MkdirAll(wg.Conf.Paths.InDirectory, 0744); err != nil {
		return fmt.Errorf("failed to create gateway in directory: %s", err)
	}
	if err := os.MkdirAll(wg.Conf.Paths.OutDirectory, 0744); err != nil {
		return fmt.Errorf("failed to create gateway out directory: %s", err)
	}
	if err := os.MkdirAll(wg.Conf.Paths.WorkDirectory, 0744); err != nil {
		return fmt.Errorf("failed to create gateway work directory: %s", err)
	}
	return nil
}

func (wg *WG) initServices() {
	wg.Services = make(map[string]service.Service)

	wg.dbService = &database.DB{Conf: wg.Conf}
	adminService := &admin.Server{Conf: wg.Conf, DB: wg.dbService, Services: wg.Services}
	controllerService := &controller.Controller{DB: wg.dbService}

	wg.Services[database.ServiceName] = wg.dbService
	wg.Services[admin.ServiceName] = adminService
	wg.Services[controller.ServiceName] = controllerService
}

func (wg *WG) startServices() error {
	if err := wg.dbService.Start(); err != nil {
		return err
	}

	var servers model.LocalAgents
	if err := wg.dbService.Select(&servers).Where("owner=?", database.Owner).
		Run(); err != nil {
		return err
	}

	for i, server := range servers {
		l := log.NewLogger(server.Name)
		constr, ok := ServiceConstructors[server.Protocol]
		if !ok {
			wg.Logger.Warningf("Unknown protocol '%s' for server %s",
				server.Protocol, server.Name)
		}
		constr(wg.dbService, &servers[i], l)
	}

	for _, serv := range wg.Services {
		if err := serv.Start(); err != nil {
			wg.Logger.Errorf("Error starting service: %s", err)
		}
	}

	return nil
}

func (wg *WG) stopServices() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
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
			_ = s.Stop(ctx)
		}(wgService)
	}
	w.Wait()

	_ = wg.dbService.Stop(ctx)
}

// Start starts the main service of the Gateway
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
