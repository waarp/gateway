// Package gatewayd contains the "root" service responsible for starting all
// the other gateway services. It is the first service started when the gateway
// is launched.
package gatewayd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/controller"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

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

//
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
		return err
	}

	servers := []*model.LocalAgent{}
	filters := &database.Filters{Conditions: builder.Eq{"owner": database.Owner}}
	if err := wg.dbService.Select(&servers, filters); err != nil {
		return err
	}

	for _, server := range servers {
		switch server.Protocol {
		case "sftp":
			l := log.NewLogger(server.Name)
			wg.Services[server.Name] = sftp.NewService(wg.dbService, server, l)
		default:
			wg.Logger.Warningf("Unknown server protocol '%s'", server.Protocol)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	delete(wg.Services, database.ServiceName)

	w := sync.WaitGroup{}
	for _, wgService := range wg.Services {
		w.Add(1)
		go func(s service.Service) {
			_ = s.Stop(ctx)
			w.Done()
		}(wgService)
	}
	w.Wait()

	_ = wg.dbService.Stop(ctx)
}

// Start starts the main service of the Gateway
func (wg *WG) Start() error {
	wg.Infof("Waarp Gateway '%s' is starting", wg.Conf.GatewayName)
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
