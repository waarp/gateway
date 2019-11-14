// Package gatewayd contains the "root" service responsible for starting all
// the other gateway services. It is the first service started when the gateway
// is launched.
package gatewayd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	dbService *database.Db
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

	wg.dbService = &database.Db{Conf: wg.Conf}
	adminService := &admin.Server{Conf: wg.Conf, Db: wg.dbService, Services: wg.Services}
	controllerService := &controller.Controller{Conf: wg.Conf, Db: wg.dbService}

	wg.Services[database.ServiceName] = wg.dbService
	wg.Services[admin.ServiceName] = adminService
	wg.Services[controller.ServiceName] = controllerService
}

func (wg *WG) startServices() error {
	if err := wg.dbService.Start(); err != nil {
		return err
	}

	servers := []*model.LocalAgent{}
	if err := wg.dbService.Select(&servers, nil); err != nil {
		return err
	}
	for _, server := range servers {
		switch server.Protocol {
		case "sftp":
			wg.Services[server.Name] = &sftp.Server{Db: wg.dbService, Config: server}
		default:
			return fmt.Errorf("unknown server protocol '%s'", server.Protocol)
		}
	}

	for _, serv := range wg.Services {
		if state, _ := serv.State().Get(); state != service.Running {
			_ = serv.Start()
		}
	}
	return nil
}

func (wg *WG) stopServices() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	for _, wgService := range wg.Services {
		_ = wgService.Stop(ctx)
	}
}

// Start starts the main service of the Gateway
func (wg *WG) Start() error {
	if err := wg.SetOutput(wg.Conf.Log.LogTo, wg.Conf.Log.SyslogFacility); err != nil {
		return fmt.Errorf("log configuration failed: %s", err.Error())
	}
	if err := wg.SetLevel(wg.Conf.Log.Level); err != nil {
		return fmt.Errorf("log configuration failed: %s", err.Error())
	}
	wg.Info("Waarp Gateway NG is starting")
	wg.initServices()
	if err := wg.startServices(); err != nil {
		return err
	}
	wg.Infof("Waarp Gateway NG has started")

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
